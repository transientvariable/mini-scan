package messaging

import "context"

type Handler[T any] interface {
	Name() string
	Next() Handler[T]
	OnMessage(ctx context.Context, msg T) error
	SetNext(handler Handler[T])
}

type Publisher[T any] interface {
	Publish(msg T) error
	Close() error
}

type Subscriber[T any] interface {
	Subscribe(handlers ...Handler[T]) error
	Add(handler Handler[T]) bool
	Close() error
}
