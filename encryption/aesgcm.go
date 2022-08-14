package encryption

import (
	"crypto/aes"
	"crypto/cipher"

	cryptoerr "github.com/seborama/govcr/v7/encryption/errors"
)

// NewAESCGM creates a new Cryptor initialised with an AES-CGM cipher from the
// supplied key.
// The key is sensitive, never share it openly.
//
// When decoded the key should be 16 bytes (AES-128) or 32 (AES-256).
//
// If you want to convert a passphrase to a key, use a suitable
// package like bcrypt or scrypt.
// TODO: as nonceGenerator is not required, make it optional with a functional opt.
// TODO: add a nonceGenerator validator i.e. call it 1000 times, ensures no dupes.
func NewAESCGM(key []byte, nonceGenerator NonceGenerator) (*Crypter, error) {
	if len(key) != 16 && len(key) != 32 {
		return nil, cryptoerr.NewErrCrypto("key size is not 16 or 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if nonceGenerator == nil {
		nonceGenerator = &DefaultNonceGenerator{}
	}

	return &Crypter{
			aead:           aesgcm,
			nonceGenerator: nonceGenerator,
		},
		nil
}

// Crypter contains the AEAD cipher to use for encryption and decryption.
type Crypter struct {
	aead           cipher.AEAD
	nonceGenerator NonceGenerator
}

type NonceGenerator interface {
	Generate() ([]byte, error)
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
