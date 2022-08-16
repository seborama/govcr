package encryption_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/v8/encryption"
)

func TestCryptor(t *testing.T) {
	key := []byte("this is a test key______________")

	aesgcm, err := encryption.NewAESGCMWithRandomNonceGenerator(key)
	require.NoError(t, err)

	inputData := []byte("My little secret!")

	ciphertext, nonce, err := aesgcm.Encrypt(inputData)
	require.NoError(t, err)

	plaintext, err := aesgcm.Decrypt(ciphertext, nonce)
	require.NoError(t, err)
	assert.Equal(t, inputData, plaintext)
}
