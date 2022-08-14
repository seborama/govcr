package govcr

import (
	"net/http"
	"os"

	"github.com/seborama/govcr/v8/cassette"
	"github.com/seborama/govcr/v8/cassette/track"
	"github.com/seborama/govcr/v8/encryption"
)

// Setting defines an optional functional parameter as received by NewVCR().
type Setting func(vcrSettings *VCRSettings)

// WithClient is an optional functional parameter to provide a VCR with
// a custom HTTP client.
func WithClient(httpClient *http.Client) Setting {
	return func(vcrSettings *VCRSettings) {
		vcrSettings.client = httpClient
	}
}

// CassetteConfig contains various configurable elements of a cassette.
type CassetteConfig struct {
	Crypter cassette.Crypter
}

// CassetteOption allows to modify a cassette config.
type CassetteOption func(cfg *CassetteConfig)

// WithCassetteCrypto creates a cassette cryptographer with the specified key file.
func WithCassetteCrypto(keyFile string) CassetteOption {
	return func(cfg *CassetteConfig) {
		key, err := os.ReadFile(keyFile)
		if err != nil {
			panic(err)
		}

		crypter, err := encryption.NewAESCGM(key, nil)
		if err != nil {
			panic(err)
		}

		cfg.Crypter = crypter
	}
}

// WithCassetteCryptoCustomNonce creates a cassette cryptographer with the specified key file and
// customer nonce generator.
func WithCassetteCryptoCustomNonce(keyFile string, nonceGenerator encryption.NonceGenerator) CassetteOption {
	return func(cfg *CassetteConfig) {
		key, err := os.ReadFile(keyFile)
		if err != nil {
			panic(err)
		}

		crypter, err := encryption.NewAESCGM(key, nonceGenerator)
		if err != nil {
			panic(err)
		}

		cfg.Crypter = crypter
	}
}

// ToCassetteOptions takes a list of CassetteOption and returns a slice of
// cassette.Option's, ready to pass to cassette initialisation.
func ToCassetteOptions(opts ...CassetteOption) []cassette.Option {
	cfg := &CassetteConfig{}

	for _, opt := range opts {
		opt(cfg)
	}

	var k7Opts []cassette.Option

	if cfg.Crypter != nil {
		k7Opts = append(k7Opts, cassette.WithCassetteCrypter(cfg.Crypter))
	}

	return k7Opts
}

// WithCassette is an optional functional parameter to provide a VCR with
// a cassette to load.
// Cassette options may be provided (e.g. cryptography).
func WithCassette(cassetteName string, opts ...CassetteOption) Setting {
	return func(vcrSettings *VCRSettings) {
		k7Opts := ToCassetteOptions(opts...)

		k7 := cassette.LoadCassette(cassetteName, k7Opts...)
		vcrSettings.cassette = k7
	}
}

// WithRequestMatcher is an optional functional parameter to provide a VCR with
// a RequestMatcher applied when matching an HTTP/S request to an existing track
// on a cassette.
func WithRequestMatcher(matcher RequestMatcher) Setting {
	return func(vcrSettings *VCRSettings) {
		vcrSettings.requestMatcher = matcher
	}
}

// WithTrackRecordingMutators is an optional functional parameter to provide a VCR with
// a set of track mutators applied when recording a track to a cassette.
func WithTrackRecordingMutators(trackRecordingMutators ...track.Mutator) Setting {
	return func(vcrSettings *VCRSettings) {
		vcrSettings.trackRecordingMutators = vcrSettings.trackRecordingMutators.Add(trackRecordingMutators...)
	}
}

// WithTrackReplayingMutators is an optional functional parameter to provide a VCR with
// a set of track mutators applied when replaying a track to a cassette.
// Replaying happens AFTER the request has been matched. As such, while the track's Request could be
// mutated, it will have no effect.
// However, the Request data can be referenced as part of mutating the Response.
func WithTrackReplayingMutators(trackReplayingMutators ...track.Mutator) Setting {
	return func(vcrSettings *VCRSettings) {
		vcrSettings.trackReplayingMutators = vcrSettings.trackReplayingMutators.Add(trackReplayingMutators...)
	}
}

// WithLiveOnlyMode sets the VCR to make live calls only, do not replay from cassette even
// if a track would exist.
// Perhaps more useful when used in combination with 'readOnly' to by-pass govcr entirely.
func WithLiveOnlyMode() Setting {
	return func(vcrSettings *VCRSettings) {
		vcrSettings.httpMode = HTTPModeLiveOnly
	}
}

// WithReadOnlyMode sets the VCR to replay tracks from cassette, if present, or make live
// calls but do not records new tracks.
func WithReadOnlyMode() Setting {
	return func(vcrSettings *VCRSettings) {
		vcrSettings.readOnly = true
	}
}

// WithOfflineMode sets the VCR to replay tracks from cassette, if present, but do not make
// live calls.
// govcr will return a transport error if no track was found.
func WithOfflineMode() Setting {
	return func(vcrSettings *VCRSettings) {
		vcrSettings.httpMode = HTTPModeOffline
	}
}

// VCRSettings holds a set of options for the VCR.
type VCRSettings struct {
	client                 *http.Client
	cassette               *cassette.Cassette
	requestMatcher         RequestMatcher
	trackRecordingMutators track.Mutators
	trackReplayingMutators track.Mutators
	httpMode               HTTPMode
	readOnly               bool
}
