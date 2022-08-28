package encryption

import (
	"github.com/pkg/errors"
	"golang.org/x/crypto/chacha20poly1305"
)

// NewChaCha20Poly1305WithRandomNonceGenerator creates a new Cryptor initialised with a
// ChaCha20Poly1305 cipher from the supplied key and the default nonce generator.
func NewChaCha20Poly1305WithRandomNonceGenerator(key []byte) (*Crypter, error) {
	return NewChaCha20Poly1305(key, nil)
}

// NewChaCha20Poly1305 creates a new Cryptor initialised with a ChaCha20Poly1305 cipher
// from the supplied key.
// The key is sensitive, never share it openly.
//
// The key should be 32 bytes long.
//
// If you want to convert a passphrase to a key, you can use a function such as Argon2.
func NewChaCha20Poly1305(key []byte, nonceGenerator NonceGenerator) (*Crypter, error) {
	cc20px, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	if nonceGenerator == nil {
		nonceGenerator = NewRandomNonceGenerator(cc20px.NonceSize())
	}

	if err = validateNonceGenerator(nonceGenerator); err != nil {
		return nil, errors.Wrap(err, "nonce generator is not valid")
	}

	return NewCrypter(cc20px, "chacha20poly1305", nonceGenerator), nil
}
