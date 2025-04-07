package miniscan

// Enumeration of common errors that may be returned during mini-scan operations.
const (
	ErrClosed  = miniScanError("already closed")
	ErrInvalid = miniScanError("invalid argument")
)

// miniScanError defines the type for errors that may be returned during mini-scan operations.
type miniScanError string

// Error returns the cause of the mini-scan error.
func (e miniScanError) Error() string {
	return string(e)
}
