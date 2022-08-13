package encryption_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/v7/encryption"
)

func TestCryptor(t *testing.T) {
	keyB64 := base64.StdEncoding.EncodeToString([]byte("this is a test key______________"))

	aescgm, err := encryption.NewAESCGM(keyB64, encryption.DefaultNonceGenerator{})
	require.NoError(t, err)

	inputData := []byte("My little secret!")

	ciphertext, nonce, err := aescgm.Encrypt(inputData)
	require.NoError(t, err)

	plaintext, err := aescgm.Decrypt(ciphertext, nonce)
	require.NoError(t, err)
	assert.Equal(t, inputData, plaintext)
}
