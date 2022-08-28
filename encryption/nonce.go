package encryption

import (
	"crypto/rand"
	"fmt"
	"io"

	"github.com/pkg/errors"
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

func validateNonceGenerator(nonceGenerator NonceGenerator) error {
	nonces := make(map[string]struct{})

	for i := 1; i <= 1000; i++ {
		n, err := nonceGenerator.Generate()
		if err != nil {
			return errors.Wrap(err, "nonceGenerator failure")
		}

		nStr := fmt.Sprintf("%x", n)
		if _, ok := nonces[nStr]; ok {
			return errors.New("nonceGenerator produces frequent duplicates")
		}

		nonces[nStr] = struct{}{}
	}

	return nil
}
