package scan

// Enumeration of common errors that may be returned during event operations.
const (
	ErrClosed                 = eventError("already closed")
	ErrDataFormatInvalid      = eventError("data format invalid")
	ErrDataVersionUnsupported = eventError("data version unsupported")
	ErrInvalid                = eventError("invalid argument")
)

// eventError defines the type for errors that may be returned during event operations.
type eventError string

// Error returns the cause of the event error.
func (e eventError) Error() string {
	return string(e)
}
