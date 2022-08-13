package errors

import (
	"fmt"
)

// ErrCrypto is an error that indicates a cryptographic failure.
type ErrCrypto struct {
	message string
}

// NewErrCrypto creates a new initialised ErrCrypto.
func NewErrCrypto(message string) *ErrCrypto {
	return &ErrCrypto{
		message: message,
	}
}

func (e ErrCrypto) Error() string {
	return fmt.Sprint(e.message)
}
