package pubsub

// Enumeration of common errors that may be returned during pub/sub operations.
const (
	ErrPublisherClosed  = pubsubError("publisher is closed")
	ErrSubscriberClosed = pubsubError("subscriber is closed")
)

// pubsubError defines the type for errors that may be returned during pub/sub operations.
type pubsubError string

// Error returns the cause of the pub/sub error.
func (e pubsubError) Error() string {
	return string(e)
}
