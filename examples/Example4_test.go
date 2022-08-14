package examples_test

import (
	"os"
	"testing"

	"github.com/seborama/govcr/v8"
	"github.com/seborama/govcr/v8/stats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const exampleCassetteName4 = "temp-fixtures/TestExample4.cassette.json"

// TestExample4 is a simple example use of govcr with cassette encryption.
// Do _NOT_ ever use the test key from this example, it is clearly not private!
func TestExample4(t *testing.T) {
	_ = os.Remove(exampleCassetteName4)

	vcr := govcr.NewVCR(
		govcr.WithCassette(
			exampleCassetteName4,
			govcr.WithCassetteCrypto("test-fixtures/TestExample4.unsafe.key"),
		),
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
	vcr.EjectCassette()
	err := vcr.LoadCassette(
		exampleCassetteName4,
		govcr.WithCassetteCrypto("test-fixtures/TestExample4.unsafe.key"),
	)
	require.NoError(t, err)

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
