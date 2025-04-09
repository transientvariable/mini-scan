package schema

import "time"

// Event defines the behavior for an event.
type Event interface {
	ID() string
	Type() string
	Metadata() map[string]any
	Name() string
	Timestamp() *time.Time
}
