package encryption

import (
	"crypto/rand"
	"io"
)

// RandomNonceGenerator is a random generator of nonce of the specified size.
type RandomNonceGenerator struct {
	size int
}

// NewRandomNonceGenerator creates a new initialised RandomNonceGenerator of specified size.
func NewRandomNonceGenerator(size int) *RandomNonceGenerator {
	return &RandomNonceGenerator{
		size: size,
	}
}

// For a 12-byte nonce, never use more than 2^32 random nonces with a given key
// because of the risk of a repeat and cipher .
func (ng RandomNonceGenerator) Generate() ([]byte, error) {
	nonce := make([]byte, ng.size)

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return nonce, nil
}
