package gcp

// Enumeration of common errors that may be returned during GCP pub/sub operations.
const (
	ErrTimeout = gcpError("timeout waiting for resource")
)

// gcpError defines the type for errors that may be returned during GCP pub/sub operations.
type gcpError string

// Error returns the cause of the GCP pub/sub error.
func (e gcpError) Error() string {
	return string(e)
}
