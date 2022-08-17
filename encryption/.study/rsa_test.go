package encryption

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromPrivateKeyFile(t *testing.T) {
	privKey, _, pubKey, err := readSSHRSAPrivateKeyFile("fixtures/id_rsa_2048", "passphrase")
	require.NoError(t, err)

	// encrypt
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, []byte("super secret message"), []byte("OAEP Encrypted"))
	require.NoError(t, err)
	ciphertextB64 := base64.StdEncoding.EncodeToString(ciphertext)
	fmt.Println("eMsg:", ciphertextB64)

	// decrypt
	ciphertext, err = base64.StdEncoding.DecodeString(ciphertextB64)
	require.NoError(t, err)
	dMsg, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privKey, ciphertext, []byte("OAEP Encrypted"))
	require.NoError(t, err)
	fmt.Println("dMsg:", string(dMsg))
}

func TestFromPublicKeyFile(t *testing.T) {
	pubKey, err := readSSHRSAPublicKeyFile("fixtures/id_rsa_2048.pub")
	require.NoError(t, err)

	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, []byte("super secret message"), []byte("OAEP Encrypted"))
	require.NoError(t, err)

	ciphertextB64 := base64.StdEncoding.EncodeToString(ciphertext)
	fmt.Println("eMsg:", ciphertextB64)
}
