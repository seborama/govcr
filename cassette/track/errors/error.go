package trkerr

import (
	"fmt"
)

// ErrTransportFailure is an error that indicates a transport failure during the HTTP dialogue.
type ErrTransportFailure struct {
	errType string
	errMsg  string
}

// NewErrTransportFailure creates a new initialised ErrTransportFailure.
func NewErrTransportFailure(errType, errMsg string) *ErrTransportFailure {
	return &ErrTransportFailure{
		errType: errType,
		errMsg:  errMsg,
	}
}

func (e ErrTransportFailure) Error() string {
	return fmt.Sprint(e.errType + ": " + e.errMsg)
}
