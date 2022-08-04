package cassette

import (
	"testing"

	"github.com/seborama/govcr/v6/stats"
	"github.com/stretchr/testify/assert"
)

func Test_cassette_NumberOfTracks_PanicsWhenNoCassette(t *testing.T) {
	var unit *Cassette

	assert.Panics(t, func() { unit.NumberOfTracks() })
}

func Test_cassette_Stats_ZeroWhenNoCassette(t *testing.T) {
	var unit *Cassette

	got := unit.Stats()

	assert.Nil(t, got)
}

func Test_cassette_Stats_ZeroWhenEmptyCassette(t *testing.T) {
	unit := LoadCassette("temp-fixtures/Test_cassette_Stats_ZeroWhenEmptyCassette.json")

	got := unit.Stats()

	expected := &stats.Stats{
		TotalTracks:    0,
		TracksLoaded:   0,
		TracksRecorded: 0,
		TracksPlayed:   0,
	}

	assert.Equal(t, expected, got)
}
