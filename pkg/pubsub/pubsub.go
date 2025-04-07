package pubsub

import "context"

type Handler[T any] interface {
	Name() string
	Next() Handler[T]
	OnMessage(ctx context.Context, msg T) error
	SetNext(handler Handler[T])
}

type Publisher[T any] interface {
	Publish(ctx context.Context, msg T) error
}

type Subscriber[T any] interface {
	Subscribe(ctx context.Context, handlers ...Handler[T]) error
}
