package govcr

import (
	"fmt"
	"net/http"
	"os"

	"github.com/pkg/errors"

	"github.com/seborama/govcr/v13/cassette"
	"github.com/seborama/govcr/v13/encryption"
)

// CrypterProvider is the signature of a cipher provider function with default nonce generator.
// Examples are encryption.NewAESGCMWithRandomNonceGenerator and
// encryption.NewChaCha20Poly1305WithRandomNonceGenerator.
type CrypterProvider func(key []byte) (*encryption.Crypter, error)

// CrypterNonceProvider is the signature of a cipher provider function with custom nonce generator.
// Examples are encryption.NewAESGCM and encryption.NewChaCha20Poly1305.
type CrypterNonceProvider func(key []byte, nonceGenerator encryption.NonceGenerator) (*encryption.Crypter, error)

// CassetteLoader helps build a cassette to load in the VCR.
type CassetteLoader struct {
	cassetteName string
	opts         []cassette.Option
}

// NewCassetteLoader creates a new CassetteLoader, initialised with the cassette's name.
func NewCassetteLoader(cassetteName string) *CassetteLoader {
	return &CassetteLoader{
		cassetteName: cassetteName,
	}
}

// WithCipher creates a cassette cryptographer with the specified cipher function
// and key file.
// Using more than one WithCipher* on the same cassette is ambiguous.
func (cb *CassetteLoader) WithCipher(crypter CrypterProvider, keyFile string) *CassetteLoader {
	f := func(key []byte, nonceGenerator encryption.NonceGenerator) (*encryption.Crypter, error) {
		// a "CrypterProvider" is a CrypterNonceProvider with a pre-defined / default nonceGenerator
		return crypter(key)
	}

	return cb.WithCipherCustomNonce(f, keyFile, nil)
}

// WithCipherCustomNonce creates a cassette cryptographer with the specified key file and
// customer nonce generator.
// Using more than one WithCipher* on the same cassette is ambiguous.
func (cb *CassetteLoader) WithCipherCustomNonce(crypterNonce CrypterNonceProvider, keyFile string, nonceGenerator encryption.NonceGenerator) *CassetteLoader {
	cr, err := makeCrypter(crypterNonce, keyFile, nonceGenerator)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	cb.opts = append(cb.opts, cassette.WithCrypter(cr))

	return cb
}

func (cb *CassetteLoader) load() *cassette.Cassette {
	if cb == nil {
		panic("please select a cassette for the VCR")
	}

	return cassette.LoadCassette(cb.cassetteName, cb.opts...)
}

func makeCrypter(crypterNonce CrypterNonceProvider, keyFile string, nonceGenerator encryption.NonceGenerator) (*encryption.Crypter, error) {
	if crypterNonce == nil {
		return nil, errors.New("a cipher must be supplied for encryption, `nil` is not permitted")
	}

	key, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	cr, err := crypterNonce(key, nonceGenerator)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return cr, nil
}

// NewVCR creates a new VCR.
func NewVCR(cassetteLoader *CassetteLoader, settings ...Setting) *ControlPanel {
	var vcrSettings VCRSettings

	vcrSettings.cassette = cassetteLoader.load()

	for _, option := range settings {
		option(&vcrSettings)
	}

	// use a default client if none provided
	if vcrSettings.client == nil {
		vcrSettings.client = http.DefaultClient
	}

	// use a default vcrTransport if none provided
	if vcrSettings.client.Transport == nil {
		vcrSettings.client.Transport = http.DefaultTransport
	}

	// use a default RequestMatcher if none provided
	if vcrSettings.requestMatchers == nil {
		vcrSettings.requestMatchers = NewStrictRequestMatchers()
	}

	// create VCR's HTTP client
	vcrClient := &http.Client{
		Transport: &vcrTransport{
			pcb: &PrintedCircuitBoard{
				requestMatchers:        vcrSettings.requestMatchers,
				trackRecordingMutators: vcrSettings.trackRecordingMutators,
				trackReplayingMutators: vcrSettings.trackReplayingMutators,
				httpMode:               vcrSettings.httpMode,
				readOnly:               vcrSettings.readOnly,
			},
			cassette:  vcrSettings.cassette,
			transport: vcrSettings.client.Transport,
		},

		// copy the attributes of the original http.Client
		CheckRedirect: vcrSettings.client.CheckRedirect,
		Jar:           vcrSettings.client.Jar,
		Timeout:       vcrSettings.client.Timeout,
	}

	// return
	return &ControlPanel{
		client: vcrClient,
	}
}
