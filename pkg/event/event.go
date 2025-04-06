package event

import "context"

// Event ...
type Event[P any] interface {
	ID() string
	Payload() P
	Type() string
}

// Handler ...
type Handler[E Event[P], P any] interface {
	OnEvent(ctx context.Context, event E) error
	Name() string
	Next() *Handler[E, P]
}

// Publisher ...
type Publisher[E Event[P], P any] interface {
	Publish(ctx context.Context) error
}

// Subscriber ...
type Subscriber[E Event[P], P any] interface {
	Subscribe(handlers ...Handler[E, P])
}
