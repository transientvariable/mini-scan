package scan

const (
	ErrInvalidData = scanDataError("invalid data")
)

// scanDataError defines the type for errors that may be returned when generating schema types for scans.
type scanDataError string

// Error returns the cause of the scan data error.
func (e scanDataError) Error() string {
	return string(e)
}
