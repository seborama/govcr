package govcr

import (
	"testing"
)

func Test_cassette_NumberOfTracks_ZeroWhenNoCassette(t *testing.T) {
	var unit *cassette

	if got := unit.NumberOfTracks(); got != 0 {
		t.Errorf("cassette.NumberOfTracks() = %v, want 0", got)
	}
}
