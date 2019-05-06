package govcr_test

import (
	"os"
	"testing"

	"github.com/seborama/govcr"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func Test_recordNewTrackToCassette_WithMutation(t *testing.T) {
	errTypeOverwriterMutator := func(nextMutator govcr.TrackRecordingMutater) govcr.TrackRecordingMutater {
		return govcr.TrackRecordingMutaterFunc(func(t *govcr.Track) {
			t.ErrType = "ErrType was mutated"
			t.ErrMsg = "ErrMsg was mutated"
			nextMutator.Mutate(t)
		}).OnNoErr()
	}

	requestMethodMutator := func(t *govcr.Track) {
		t.Request.Method = t.Request.Method + " has been mutated"
	}

	cassetteName := "test-fixtures/Test_recordNewTrackToCassette_WithMutation.cassette"
	_ = os.Remove(cassetteName)
	defer func() { _ = os.Remove(cassetteName) }()

	mutater := errTypeOverwriterMutator(govcr.TrackRecordingMutaterFunc(requestMethodMutator))
	k7 := govcr.NewCassette(cassetteName, govcr.WithTrackRecordingMutator(mutater))
	k7.AddTrack(govcr.NewTrack(&govcr.Request{
		Method: "BadMethod",
	}, &govcr.Response{
		Status: "BadStatus",
	}, nil))

	require.EqualValues(t, 1, k7.NumberOfTracks())
	track := k7.Track(0)
	assert.EqualValues(t, "BadMethod has been mutated", track.Request.Method)
	assert.EqualValues(t, "BadStatus", track.Response.Status)
	assert.EqualValues(t, "ErrType was mutated", track.ErrType)
	assert.EqualValues(t, "ErrMsg was mutated", track.ErrMsg)
}
