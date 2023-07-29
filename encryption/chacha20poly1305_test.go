package encryption_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/v14/encryption"
)

func TestCryptor_ChaCha20Poly1305(t *testing.T) {
	key := []byte("this is a test key______________")

	cc20px, err := encryption.NewChaCha20Poly1305WithRandomNonceGenerator(key)
	require.NoError(t, err)

	inputData := []byte("My little secret!")

	ciphertext, nonce, err := cc20px.Encrypt(inputData)
	require.NoError(t, err)

	plaintext, err := cc20px.Decrypt(ciphertext, nonce)
	require.NoError(t, err)
	assert.Equal(t, inputData, plaintext)
}
