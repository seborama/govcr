package encryption

import (
	"crypto/rand"
	"io"
)

type DefaultNonceGenerator struct{}

// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
func (ng DefaultNonceGenerator) Generate() ([]byte, error) {
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return nonce, nil
}
