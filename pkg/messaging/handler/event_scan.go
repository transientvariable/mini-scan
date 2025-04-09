package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/censys/scan-takehome/pkg/messaging"
	"github.com/censys/scan-takehome/pkg/scanning"
	"github.com/censys/scan-takehome/pkg/schema/scan"

	"github.com/cenkalti/backoff/v4"
	"github.com/transientvariable/anchor"
	"github.com/transientvariable/cadre/ecs"
	"github.com/transientvariable/hold"
	"github.com/transientvariable/hold/trie"
	"github.com/transientvariable/log-go"

	gcpubsub "cloud.google.com/go/pubsub"
	json "github.com/json-iterator/go"
	opensearch "github.com/transientvariable/repository-opensearch-go"
	gcdpubsub "gocloud.dev/pubsub"
)

const (
	findForUpdateMaxRetries = 3
	indexLogsEventScan      = "logs-event-scan"
	indexMetadataScan       = "metadata-scan"
	resultSizeMax           = 1000
)

var (
	_ messaging.Handler[any] = (*scanEventHandler)(nil)
)

type scanEventHandler struct {
	docs      *opensearch.Repository
	mutex     sync.Mutex
	next      messaging.Handler[any]
	processed trie.Trie
}

// NewScanEventHandler creates a new messaging.Handler for processing scan events.
//
// Idempotency: This handler is idempotent and will discard duplicate messages.
func NewScanEventHandler(options ...func(*scanEventHandler)) (messaging.Handler[any], error) {
	processed, err := trie.New()
	if err != nil {
		return nil, fmt.Errorf("scan_event_handler: %w", err)
	}

	h := &scanEventHandler{processed: processed}
	for _, opts := range options {
		opts(h)
	}

	if h.docs == nil {
		h.docs = opensearch.New()
	}
	return h, nil
}

func (h *scanEventHandler) Name() string {
	return "scan-event-handler"
}

func (h *scanEventHandler) Next() messaging.Handler[any] {
	return h.next
}

func (h *scanEventHandler) OnMessage(ctx context.Context, msg any) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	m, err := decodeScanMetadata(msg)
	if err != nil {
		return fmt.Errorf("scan_event_handler: %w", err)
	}

	// Handle idempotency
	if h.processed.Len() > 0 {
		if _, err := h.processed.Entry(m.ID); err != nil {
			if !errors.Is(err, hold.ErrNotFound) {
				return fmt.Errorf("scan_event_handler: %w", err)
			}
		} else {
			return nil
		}
	}

	log.Info("[handler:scan_event] processing scan metadata",
		log.String("id", m.ID),
		log.String("ip", m.Source.IP),
		log.Int64("port", m.Source.Port),
		log.String("service", m.Network.Protocol),
		log.Time("ts", *m.Timestamp),
		log.Int("processed", h.processed.Len()))

	if err := h.processed.Add(m.ID); err != nil {
		return fmt.Errorf("scan_event_handler: processed_add: %w", err)
	}

	eventType := ecs.EventTypeChange
	if _, err := h.findByID(ctx, indexMetadataScan, m.ID); err != nil {
		if !errors.Is(err, ErrMetadataNotFound) {
			return fmt.Errorf("scan_event_handler: %w", err)
		}
		eventType = ecs.EventTypeCreation
	}

	switch eventType {
	case ecs.EventTypeCreation:
		err = h.create(ctx, indexMetadataScan, m)
		break
	case ecs.EventTypeChange:
		err = h.update(ctx, indexMetadataScan, m)
		break
	case ecs.EventTypeDeletion:
		err = h.delete(ctx, indexMetadataScan, m.ID)
		break
	default:
		err = &Error{
			Source:  "scan_event_handler",
			Message: fmt.Sprintf("event type is not supported: %s", eventType),
		}
	}
	if err != nil {
		return err
	}

	e, err := scan.NewEvent(eventType, m)
	if err != nil {
		return fmt.Errorf("scan_event_handler: %w", err)
	}

	doc := opensearch.NewDocument(
		opensearch.WithIndex(indexLogsEventScan),
		opensearch.WithContent(anchor.ToJSON(e)),
	)

	log.Trace(fmt.Sprintf("[handler:scan_event] creating scan event with document:\n%s", doc))

	if _, err := h.docs.Create(ctx, doc); err != nil {
		return err
	}
	return nil
}

func (h *scanEventHandler) SetNext(handler messaging.Handler[any]) {
	h.next = handler
}

func (h *scanEventHandler) create(ctx context.Context, index string, metadata *scan.Metadata) error {
	doc := opensearch.NewDocument(
		opensearch.WithDocumentID(metadata.ID),
		opensearch.WithIndex(index),
		opensearch.WithContent(anchor.ToJSON(metadata)),
	)
	if _, err := h.docs.Create(ctx, doc); err != nil {
		return err
	}
	return nil
}

func (h *scanEventHandler) update(ctx context.Context, index string, metadata *scan.Metadata) error {
	_, err := h.findByID(ctx, index, metadata.ID)
	if err != nil {
		if errors.Is(err, ErrMetadataNotFound) {
			return h.create(ctx, index, metadata)
		}
		return err
	}

	fields := map[string]any{
		"network": metadata.Network,
		"source":  metadata.Source,
	}

	if _, err := h.docs.Update(
		ctx,
		opensearch.NewDocument(opensearch.WithIndex(index), opensearch.WithDocumentID(metadata.ID)),
		opensearch.WithFields(fields),
	); err != nil {
		return err
	}
	return nil
}

func (h *scanEventHandler) delete(ctx context.Context, index string, id string) error {
	log.Info("[handler:scan_event] delete",
		log.String("index", index),
		log.String("id", id))
	_, err := h.docs.Delete(ctx, index, opensearch.WithTerm("id", id, opensearch.BoolPredicateFilter))
	if err != nil {
		return err
	}
	return nil
}

func (h *scanEventHandler) findByID(ctx context.Context, index string, id string) (*scan.Metadata, error) {
	return h.findByQuery(func() (*opensearch.Result, error) {
		log.Trace("[handler:scan_event] locating scan event document",
			log.String("index", index),
			log.String("id", id))
		return h.docs.Search(
			ctx,
			index,
			opensearch.WithSize(resultSizeMax),
			opensearch.WithSource(true),
			opensearch.WithTerm("id", id, opensearch.BoolPredicateFilter),
		)
	})
}

func (h *scanEventHandler) findByQuery(queryFn func() (*opensearch.Result, error)) (*scan.Metadata, error) {
	var m *scan.Metadata
	err := backoff.Retry(func() error {
		r, err := queryFn()
		if err != nil {
			return err
		}

		if len(r.Documents) > 0 {
			doc := r.Documents[0]
			if err := json.NewDecoder(doc.Reader()).Decode(&m); err != nil {
				return err
			}
			return nil
		}
		return ErrMetadataNotFound
	}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), uint64(findForUpdateMaxRetries)))
	if err != nil {
		return nil, err
	}
	return m, nil
}

func decodeScanMetadata(msg any) (*scan.Metadata, error) {
	var b []byte
	switch m := msg.(type) {
	case *gcpubsub.Message:
		b = m.Data
	case *gcdpubsub.Message:
		b = m.Body
	default:
		return nil, ErrDataFormatInvalid
	}

	var s scanning.Scan
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&s); err != nil {
		return nil, err
	}
	return scan.NewMetadata(&s)
}

// WithDocsRepository sets the document repository to use for the scan event handler.
func WithDocsRepository(docs *opensearch.Repository) func(*scanEventHandler) {
	return func(h *scanEventHandler) {
		h.docs = docs
	}
}
