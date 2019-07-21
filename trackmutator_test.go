package govcr_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr"
	"github.com/seborama/govcr/cassette"
)

func Test_TrackMutator_OnNoErr_WhenNoErr(t *testing.T) {
	aMutator := govcr.TrackMutator(
		func(t *cassette.Track) {
			t.Request.Method = t.Request.Method + " has been mutated"
			t.Response.Status = t.Response.Status + " has been mutated"
			t.ErrType = "ErrType was mutated"
			t.ErrMsg = "ErrMsg was mutated"
		}).OnNoErr()

	aTrack := cassette.NewTrack(&cassette.Request{
		Method: "BadMethod",
	}, &cassette.Response{
		Status: "BadStatus",
	}, nil)

	aMutator(aTrack)

	require.EqualValues(t, "BadMethod has been mutated", aTrack.Request.Method)
	require.EqualValues(t, "BadStatus has been mutated", aTrack.Response.Status)
	require.EqualValues(t, "ErrType was mutated", aTrack.ErrType)
	require.EqualValues(t, "ErrMsg was mutated", aTrack.ErrMsg)
}

func Test_TrackMutator_OnNoErr_WhenErr(t *testing.T) {
	aMutator := govcr.TrackMutator(
		func(t *cassette.Track) {
			t.Request.Method = t.Request.Method + " has been mutated"
			t.Response.Status = t.Response.Status + " has been mutated"
			t.ErrType = "ErrType was mutated"
			t.ErrMsg = "ErrMsg was mutated"
		}).OnNoErr()

	aTrack := cassette.NewTrack(&cassette.Request{
		Method: "BadMethod",
	}, &cassette.Response{
		Status: "BadStatus",
	}, errors.New("an error"))

	aMutator(aTrack)

	require.EqualValues(t, "BadMethod", aTrack.Request.Method)
	require.EqualValues(t, "BadStatus", aTrack.Response.Status)
	require.Contains(t, aTrack.ErrType, "error")
	require.EqualValues(t, "an error", aTrack.ErrMsg)
}

func Test_TrackMutator_OnErr_WhenErr(t *testing.T) {
	errorMutator := govcr.TrackMutator(
		func(t *cassette.Track) {
			t.Request.Method = t.Request.Method + " has been mutated"
			t.Response.Status = t.Response.Status + " has been mutated"
			t.ErrType = "ErrType was mutated"
			t.ErrMsg = "ErrMsg was mutated"
		}).OnErr()

	aTrack := cassette.NewTrack(&cassette.Request{
		Method: "BadMethod",
	}, &cassette.Response{
		Status: "BadStatus",
	}, errors.New("an error"))

	errorMutator(aTrack)

	require.EqualValues(t, "BadMethod has been mutated", aTrack.Request.Method)
	require.EqualValues(t, "BadStatus has been mutated", aTrack.Response.Status)
	require.EqualValues(t, "ErrType was mutated", aTrack.ErrType)
	require.EqualValues(t, "ErrMsg was mutated", aTrack.ErrMsg)
}

func Test_TrackMutator_OnErr_WhenNoErr(t *testing.T) {
	errorMutator := govcr.TrackMutator(
		func(t *cassette.Track) {
			t.Request.Method = t.Request.Method + " has been mutated"
			t.Response.Status = t.Response.Status + " has been mutated"
			t.ErrType = "ErrType was mutated"
			t.ErrMsg = "ErrMsg was mutated"
		}).OnErr()

	aTrack := cassette.NewTrack(&cassette.Request{
		Method: "BadMethod",
	}, &cassette.Response{
		Status: "BadStatus",
	}, nil)

	errorMutator(aTrack)

	require.EqualValues(t, "BadMethod", aTrack.Request.Method)
	require.EqualValues(t, "BadStatus", aTrack.Response.Status)
	require.EqualValues(t, "", aTrack.ErrType)
	require.EqualValues(t, "", aTrack.ErrMsg)
}
