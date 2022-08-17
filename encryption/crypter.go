package encryption

import (
	"crypto/cipher"
)

// Crypter contains the AEAD cipher to use for encryption and decryption.
type Crypter struct {
	aead           cipher.AEAD
	nonceGenerator NonceGenerator
	kind           string
}

// NonceGenerator defines the behaviour of a Nonce Generator type.
type NonceGenerator interface {
	Generate() ([]byte, error)
}

// NewCrypter creates a new initialised Crypter.
func NewCrypter(aead cipher.AEAD, kind string, nonceGenerator NonceGenerator) *Crypter {
	return &Crypter{
		aead:           aead,
		kind:           kind,
		nonceGenerator: nonceGenerator,
	}
}

func (c Crypter) Kind() string {
	return c.kind
}

// Encrypt performs the encryption of the provided plaintext with the key
// associated with this Crypter and the supplied nonce.
// The nonce is generated from c.nonceGenerator.
func (c Crypter) Encrypt(plaintext []byte) (ciphertext, nonce []byte, err error) {
	nonce, err = c.nonceGenerator.Generate()
	if err != nil {
		return nil, nil, err
	}

	ciphertext = c.aead.Seal(nil, nonce, plaintext, nil)

	return ciphertext, nonce, nil
}

// Decrypt performs the decryption of the provided ciphertext with the key
// associated with this Crypter and the supplied nonce. This must be the same
// nonce that was used to encrypt the ciphertext.
// The nonce is not sensitive.
func (c Crypter) Decrypt(ciphertext, nonce []byte) ([]byte, error) {
	text, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return text, nil
}
