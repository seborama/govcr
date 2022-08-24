package govcr

import (
	"fmt"
	"net/http"
	"os"

	"github.com/seborama/govcr/v10/cassette"
	"github.com/seborama/govcr/v10/encryption"
)

// CrypterProvider is the signature of a cipher provider function with default nonce generator.
// Examples are encryption.NewAESGCMWithRandomNonceGenerator and
// encryption.NewChaCha20Poly1305WithRandomNonceGenerator.
type CrypterProvider func(key []byte) (*encryption.Crypter, error)

// CrypterNonceProvider is the signature of a cipher provider function with custom nonce generator.
// Examples are encryption.NewAESGCM and encryption.NewChaCha20Poly1305.
type CrypterNonceProvider func(key []byte, nonceGenerator encryption.NonceGenerator) (*encryption.Crypter, error)

// CassetteMaker helps build a cassette to load in the VCR.
type CassetteMaker struct {
	cassetteName string
	opts         []cassette.Option
}

// NewCassetteMaker creates a new CassetteMaker, initialised with the cassette's name.
func NewCassetteMaker(cassetteName string) *CassetteMaker {
	return &CassetteMaker{
		cassetteName: cassetteName,
	}
}

// WithCassetteCrypto creates a cassette cryptographer with the specified cipher function
// and key file.
// Only use WithCassetteCrypto or WithCassetteCrypto*. Using both is ambiguous.
func (cb *CassetteMaker) WithCassetteCrypto(crypter CrypterProvider, keyFile string) *CassetteMaker {
	f := func(key []byte, nonceGenerator encryption.NonceGenerator) (*encryption.Crypter, error) {
		// a "CrypterProvider" is a CrypterNonceProvider with a pre-defined / default nonceGenerator
		return crypter(key)
	}

	return cb.WithCassetteCryptoCustomNonce(f, keyFile, nil)
}

// WithCassetteCryptoCustomNonce creates a cassette cryptographer with the specified key file and
// customer nonce generator.
// Only use WithCassetteCrypto or WithCassetteCrypto*. Using both is ambiguous.
func (cb *CassetteMaker) WithCassetteCryptoCustomNonce(crypterNonce CrypterNonceProvider, keyFile string, nonceGenerator encryption.NonceGenerator) *CassetteMaker {
	key, err := os.ReadFile(keyFile)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	cr, err := crypterNonce(key, nonceGenerator)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	cb.opts = append(cb.opts, cassette.WithCassetteCrypter(cr))

	return cb
}

// WithCassette is an optional functional parameter to provide a VCR with
// a cassette to load.
// Cassette options may be provided (e.g. cryptography).
func (cb *CassetteMaker) make() *cassette.Cassette {
	if cb == nil {
		panic("please select a cassette for the VCR")
	}

	return cassette.LoadCassette(cb.cassetteName, cb.opts...)
}

// NewVCR creates a new VCR.
func NewVCR(cassetteMaker *CassetteMaker, settings ...Setting) *ControlPanel {
	var vcrSettings VCRSettings

	vcrSettings.cassette = cassetteMaker.make()

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
	if vcrSettings.requestMatcher == nil {
		vcrSettings.requestMatcher = NewStrictRequestMatcher()
	}

	// create VCR's HTTP client
	vcrClient := &http.Client{
		Transport: &vcrTransport{
			pcb: &PrintedCircuitBoard{
				requestMatcher:         vcrSettings.requestMatcher,
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
