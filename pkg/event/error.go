package event

// Enumeration of common errors that may be returned during event operations.
const (
	ErrDataFormatInvalid      = eventError("data format invalid")
	ErrDataVersionUnsupported = eventError("data version unsupported")
)

// eventError defines the type for errors that may be returned during event operations.
type eventError string

// Error returns the cause of the event error.
func (e eventError) Error() string {
	return string(e)
}
