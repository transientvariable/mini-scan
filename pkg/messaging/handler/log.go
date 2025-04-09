package handler

import (
	"context"

	"github.com/censys/scan-takehome/pkg/messaging"

	"github.com/transientvariable/log-go"

	gcpubsub "cloud.google.com/go/pubsub"
	gcdpubsub "gocloud.dev/pubsub"
)

var (
	_ messaging.Handler[any] = (*logHandler)(nil)
)

type logHandler struct {
	next messaging.Handler[any]
}

func NewLogHandler() messaging.Handler[any] {
	return &logHandler{}
}

func (h *logHandler) Name() string {
	return "log-handler"
}

func (h *logHandler) Next() messaging.Handler[any] {
	return h.next
}

func (h *logHandler) OnMessage(ctx context.Context, msg any) error {
	switch m := msg.(type) {
	case *gcpubsub.Message:
		log.Info("[handler:log] GCP message received",
			log.Any("attrs", m.Attributes),
			log.String("id", m.ID),
			log.Int("size", len(m.Data)))
	case *gcdpubsub.Message:
		log.Info("[handler:log] GDK message received",
			log.Any("metadata", m.Metadata),
			log.String("id", m.LoggableID),
			log.Int("size", len(m.Body)))
	default:
		// no-op
	}

	if h.Next() != nil {
		return h.Next().OnMessage(ctx, msg)
	}
	return nil
}

func (h *logHandler) SetNext(handler messaging.Handler[any]) {
	h.next = handler
}
