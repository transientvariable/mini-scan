package gcp

import (
	"context"

	"github.com/censys/scan-takehome/pkg/pubsub"

	"github.com/transientvariable/log-go"

	gcpubsub "cloud.google.com/go/pubsub"
	gcdpubsub "gocloud.dev/pubsub"
)

var (
	_ pubsub.Handler[constr] = (*LogHandler[constr])(nil)
)

type LogHandler[M Message] struct {
	next pubsub.Handler[M]
}

func (h *LogHandler[M]) Name() string {
	return "gcp_log_handler"
}

func (h *LogHandler[M]) Next() pubsub.Handler[M] {
	return h.next
}

func (h *LogHandler[M]) OnMessage(ctx context.Context, msg M) error {
	switch m := any(msg).(type) {
	case *gcpubsub.Message:
		log.Info("[gcp_log_handler] GCP message received",
			log.Any("attrs", m.Attributes),
			log.String("id", m.ID),
			log.Int("size", len(m.Data)))
	case *gcdpubsub.Message:
		log.Info("[gcp_log_handler] GDK message received",
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

func (h *LogHandler[M]) SetNext(handler pubsub.Handler[M]) {
	h.next = handler
}
