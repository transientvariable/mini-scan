package event

type Event interface {
	ID() string
	Type() string
}
