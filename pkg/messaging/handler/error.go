package handler

import "fmt"

// handlerError defines the type for errors that may be returned during handler operations.
type handlerError string

// Error returns the cause of the handler error.
func (e handlerError) Error() string {
	return string(e)
}

// Enumeration of common errors that may be returned during handler operations.
const (
	ErrDataFormatInvalid = handlerError("data format invalid")
	ErrMetadataNotFound  = handlerError("metadata not found")
)

// Error defines the type for errors that may be returned during handler operations.
type Error struct {
	Source    string
	Operation string
	Message   string
}

// Error returns the cause of the handler Error.
func (e *Error) Error() string {
	if e.Operation != "" {
		return fmt.Sprintf("%s_%s: %s", e.Source, e.Operation, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Source, e.Message)
}
