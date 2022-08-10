package encryption

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

func readSSHRSAPrivateKeyFile(privKeyFile, passphrase string) (*rsa.PrivateKey, ssh.Signer, *rsa.PublicKey, error) {
	keyData, err := os.ReadFile(privKeyFile)
	if err != nil {
		return nil, nil, nil, err
	}

	sshSigner, err := ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(passphrase))
	if err != nil {
		return nil, nil, nil, err
	}

	if sshSigner.PublicKey().Type() != ssh.KeyAlgoRSA {
		return nil, nil, nil, errors.Errorf("'%s' not supported, only '%s' supported", sshSigner.PublicKey().Type(), ssh.KeyAlgoRSA)
	}

	sshPrivKey, err := ssh.ParseRawPrivateKeyWithPassphrase(keyData, []byte(passphrase))
	if err != nil {
		return nil, nil, nil, err
	}

	rsaPrivKey := sshPrivKey.((*rsa.PrivateKey))

	sshPubKey, err := ssh.ParsePublicKey(sshSigner.PublicKey().Marshal())
	parsedCryptoKey := sshPubKey.(ssh.CryptoPublicKey)
	pubCrypto := parsedCryptoKey.CryptoPublicKey()
	rsaPubKey := pubCrypto.(*rsa.PublicKey)

	return rsaPrivKey, sshSigner, rsaPubKey, err
}

func readSSHRSAPublicKeyFile(pubKeyFile string) (*rsa.PublicKey, error) {
	keyData, err := os.ReadFile(pubKeyFile)
	if err != nil {
		return nil, err
	}

	sshPubKey, _, _, _, err := ssh.ParseAuthorizedKey(keyData)
	if err != nil {
		return nil, err
	}

	if sshPubKey.Type() != ssh.KeyAlgoRSA {
		return nil, errors.Errorf("'%s' not supported, only '%s' supported", sshPubKey.Type(), ssh.KeyAlgoRSA)
	}

	parsedCryptoKey := sshPubKey.(ssh.CryptoPublicKey)
	pubCrypto := parsedCryptoKey.CryptoPublicKey()
	pubKey := pubCrypto.(*rsa.PublicKey)

	return pubKey, nil
}

func EncryptFromSSHSigner(sshSigner ssh.Signer) (string, error) {
	if sshSigner.PublicKey().Type() != ssh.KeyAlgoRSA {
		return "", errors.Errorf("'%s' not supported, only '%s' supported", sshSigner.PublicKey().Type(), ssh.KeyAlgoRSA)
	}

	serSSHPubKey, err := ssh.ParsePublicKey(sshSigner.PublicKey().Marshal())
	if err != nil {
		return "", err
	}

	parsedCryptoKey := serSSHPubKey.(ssh.CryptoPublicKey)
	pubCrypto := parsedCryptoKey.CryptoPublicKey()
	pub := pubCrypto.(*rsa.PublicKey)
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, []byte("super secret message"), []byte("OAEP Encrypted"))
	if err != nil {
		return "", err
	}

	ciphertextB64 := base64.StdEncoding.EncodeToString(ciphertext)

	return ciphertextB64, nil
}
