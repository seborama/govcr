package examples_test

import (
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/v17"
	"github.com/seborama/govcr/v17/fileio"
	"github.com/seborama/govcr/v17/stats"
)

// TestExample5 is a simple example use of govcr with a AWS S3 cassette storage.
func TestExample5(t *testing.T) {
	bucketName := "example5-" + uuid.New().String() // warning: max length: 63 chars
	slog.Info("AWS info", slog.String("bucketName:", bucketName))
	exampleCassetteName5 := "/" + bucketName + "/temp-fixtures/TestExample5.cassette.json"

	s3Client, err := makeS3ClientWithBucket(bucketName)
	require.NoError(t, err)

	s3f := fileio.NewAWS(s3Client)

	vcr := govcr.NewVCR(govcr.
		NewCassetteLoader(exampleCassetteName5).
		WithStore(s3f),
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
		govcr.NewCassetteLoader(exampleCassetteName5).
			WithStore(s3f),
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
