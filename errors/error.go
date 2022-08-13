package errors

import (
	"fmt"
)

// ErrGoVCR is an error that indicates a govcr failure.
type ErrGoVCR struct {
	message string
}

// NewErrGoVCR creates a new initialised ErrGoVCR.
func NewErrGoVCR(message string) *ErrGoVCR {
	return &ErrGoVCR{
		message: message,
	}
}

func (e ErrGoVCR) Error() string {
	return fmt.Sprint(e.message)
}
