package scan

import (
	"context"

	"cloud.google.com/go/pubsub"
	"github.com/transientvariable/log-go"

	gcdpubsub "gocloud.dev/pubsub"
)

type Message interface {
	~*pubsub.Message | ~*gcdpubsub.Message
}

// Handler defines the behavior for pub/sub message handler.
type Handler[T Message] interface {
	Name() string
	Next() Handler[T]
	OnMessage(ctx context.Context, message T) error
	SetNext(handler Handler[T])
}

type LogHandler[T Message] struct {
	next Handler[T]
}

func (h *LogHandler[T]) Name() string {
	return "log_handler"
}

func (h *LogHandler[T]) Next() Handler[T] {
	return h.next
}

func (h *LogHandler[T]) OnMessage(ctx context.Context, message T) error {
	if m, ok := any(message).(*gcdpubsub.Message); ok {
		log.Info("[log_handler]", log.Any("message", m))
	}

	if h.Next() != nil {
		return h.Next().OnMessage(ctx, message)
	}
	return nil
}

func (h *LogHandler[T]) SetNext(handler Handler[T]) {
	h.next = handler
}
