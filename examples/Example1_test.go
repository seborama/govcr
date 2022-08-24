package examples_test

import (
	"os"
	"testing"

	"github.com/seborama/govcr/v10"
	"github.com/seborama/govcr/v10/stats"
	"github.com/stretchr/testify/assert"
)

const exampleCassetteName1 = "temp-fixtures/TestExample1.cassette.json"

// TestExample1 is a simple example use of govcr.
func TestExample1(t *testing.T) {
	_ = os.Remove(exampleCassetteName1)

	vcr := govcr.NewVCR(
		govcr.NewCassetteMaker(exampleCassetteName1),
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
		govcr.NewCassetteMaker(exampleCassetteName1),
		govcr.WithRequestMatcher(govcr.NewMethodURLRequestMatcher()), // use a "relaxed" request matcher
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
