package examples_test

import (
	"os"
	"testing"

	"github.com/seborama/govcr/v11"
	"github.com/seborama/govcr/v11/encryption"
	"github.com/seborama/govcr/v11/stats"
	"github.com/stretchr/testify/assert"
)

const exampleCassetteName4 = "temp-fixtures/TestExample4.cassette.json"

// TestExample4 is a simple example use of govcr with cassette encryption.
// Do _NOT_ ever use the test key from this example, it is clearly not private!
func TestExample4(t *testing.T) {
	_ = os.Remove(exampleCassetteName4)

	vcr := govcr.NewVCR(
		govcr.NewCassetteLoader(exampleCassetteName4).
			WithCassetteCrypto(
				encryption.NewChaCha20Poly1305WithRandomNonceGenerator,
				"test-fixtures/TestExample4.unsafe.key"),
		govcr.WithRequestMatcher(govcr.NewMethodURLRequestMatcher()), // use a "relaxed" request matcher
	)

	// The first request will be live and transparently recorded by govcr since the cassette is empty
	vcr.HTTPClient().Get("http://example.com/foo")
	assert.Equal(
		t,
		&stats.Stats{
			TotalTracks:    1,
			TracksLoaded:   0,
			TracksRecorded: 1,
			TracksPlayed:   0,
		},
		vcr.Stats(),
	)

	// The second request will be transparently replayed from the cassette by govcr
	// No live HTTP request is placed to the live server
	vcr = govcr.NewVCR(
		govcr.NewCassetteLoader(exampleCassetteName4).
			WithCassetteCrypto(
				encryption.NewChaCha20Poly1305WithRandomNonceGenerator,
				"test-fixtures/TestExample4.unsafe.key"),
	)

	vcr.HTTPClient().Get("http://example.com/foo")
	assert.Equal(
		t,
		&stats.Stats{
			TotalTracks:    1,
			TracksLoaded:   1,
			TracksRecorded: 0,
			TracksPlayed:   1,
		},
		vcr.Stats(),
	)
}
