package encryption

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"

	cryptoerr "github.com/seborama/govcr/v15/encryption/errors"
)

// nolint:deadcode
// TODO: offer ability to supply the key via an environment variable in base64 format.
func readSSHRSAPrivateKeyFile(privKeyFile, passphrase string) (rsaPrivKey *rsa.PrivateKey, sshSigner ssh.Signer, rsaPubKey *rsa.PublicKey, err error) {
	keyData, err := os.ReadFile(privKeyFile)
	if err != nil {
		return nil, nil, nil, errors.WithStack(err)
	}

	sshSigner, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(passphrase))
	if err != nil {
		return nil, nil, nil, errors.WithStack(err)
	}

	if sshSigner.PublicKey().Type() != ssh.KeyAlgoRSA {
		return nil, nil, nil, cryptoerr.NewErrCrypto(fmt.Sprintf("'%s' not supported, only '%s' supported", sshSigner.PublicKey().Type(), ssh.KeyAlgoRSA))
	}

	sshPrivKey, err := ssh.ParseRawPrivateKeyWithPassphrase(keyData, []byte(passphrase))
	if err != nil {
		return nil, nil, nil, errors.WithStack(err)
	}

	rsaPrivKey, ok := sshPrivKey.((*rsa.PrivateKey))
	if !ok {
		return nil, nil, nil, cryptoerr.NewErrCrypto("cannot convert ssh private key to rsa private key")
	}

	sshPubKey, err := ssh.ParsePublicKey(sshSigner.PublicKey().Marshal())
	if err != nil {
		return nil, nil, nil, errors.WithStack(err)
	}

	parsedCryptoKey, ok := sshPubKey.(ssh.CryptoPublicKey)
	if !ok {
		return nil, nil, nil, cryptoerr.NewErrCrypto("SSH public key is not SSH crypto public key")
	}

	pubCrypto := parsedCryptoKey.CryptoPublicKey()

	rsaPubKey, ok = pubCrypto.(*rsa.PublicKey)
	if !ok {
		return nil, nil, nil, cryptoerr.NewErrCrypto("public key is not RSA public key")
	}

	return rsaPrivKey, sshSigner, rsaPubKey, nil
}

// nolint:deadcode
// TODO: offer ability to supply the key via an environment variable in base64 format.
func readSSHRSAPublicKeyFile(pubKeyFile string) (*rsa.PublicKey, error) {
	keyData, err := os.ReadFile(pubKeyFile)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	sshPubKey, _, _, _, err := ssh.ParseAuthorizedKey(keyData)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if sshPubKey.Type() != ssh.KeyAlgoRSA {
		return nil, cryptoerr.NewErrCrypto(fmt.Sprintf("'%s' not supported, only '%s' supported", sshPubKey.Type(), ssh.KeyAlgoRSA))
	}

	parsedCryptoKey, ok := sshPubKey.(ssh.CryptoPublicKey)
	if !ok {
		return nil, cryptoerr.NewErrCrypto("SSH public key is not SSH crypto public key")
	}

	pubCrypto := parsedCryptoKey.CryptoPublicKey()

	pubKey, ok := pubCrypto.(*rsa.PublicKey)
	if !ok {
		return nil, cryptoerr.NewErrCrypto("public key is not RSA public key")
	}

	return pubKey, nil
}

func EncryptFromSSHSigner(sshSigner ssh.Signer) (string, error) {
	if sshSigner.PublicKey().Type() != ssh.KeyAlgoRSA {
		return "", cryptoerr.NewErrCrypto(fmt.Sprintf("'%s' not supported, only '%s' supported", sshSigner.PublicKey().Type(), ssh.KeyAlgoRSA))
	}

	serSSHPubKey, err := ssh.ParsePublicKey(sshSigner.PublicKey().Marshal())
	if err != nil {
		return "", errors.WithStack(err)
	}

	parsedCryptoKey, ok := serSSHPubKey.(ssh.CryptoPublicKey)
	if !ok {
		return "", cryptoerr.NewErrCrypto("SSH public key is not SSH crypto public key")
	}

	pubCrypto := parsedCryptoKey.CryptoPublicKey()

	pub, ok := pubCrypto.(*rsa.PublicKey)
	if !ok {
		return "", cryptoerr.NewErrCrypto("public key is not RSA public key")
	}

	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, []byte("super secret message"), []byte("OAEP Encrypted"))
	if err != nil {
		return "", errors.WithStack(err)
	}

	ciphertextB64 := base64.StdEncoding.EncodeToString(ciphertext)

	return ciphertextB64, nil
}
