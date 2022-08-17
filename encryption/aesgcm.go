package encryption

import (
	"crypto/aes"
	"crypto/cipher"

	cryptoerr "github.com/seborama/govcr/v8/encryption/errors"
)

// NewAESGCMWithRandomNonceGenerator creates a new Cryptor initialised with an
// AES-GCM cipher from the supplied key and the default nonce generator.
func NewAESGCMWithRandomNonceGenerator(key []byte) (*Crypter, error) {
	return NewAESGCM(key, nil)
}

// NewAESGCM creates a new Cryptor initialised with an AES-GCM cipher from the
// supplied key.
// The key is sensitive, never share it openly.
//
// The key should be 16 bytes (AES-128) or 32 bytes (AES-256) long.
//
// If you want to convert a passphrase to a key, use a suitable
// package like bcrypt or scrypt.
//
// TODO: add a nonceGenerator validator i.e. call it 1000 times, ensures no dupes.
func NewAESGCM(key []byte, nonceGenerator NonceGenerator) (*Crypter, error) {
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
		nonceGenerator = NewRandomNonceGenerator(aesgcm.NonceSize())
	}

	return NewCrypter(aesgcm, "aesgcm", nonceGenerator), nil
}
