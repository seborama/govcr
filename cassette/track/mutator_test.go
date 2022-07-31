package track_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/v5/cassette/track"
)

func Test_Mutator_OnNoErr_WhenNoErr(t *testing.T) {
	aMutator := track.Mutator(
		func(tk *track.Track) {
			tk.Request.Method = tk.Request.Method + " has been mutated"
			tk.Response.Status = tk.Response.Status + " has been mutated"
			tk.ErrType = strPtr("ErrType was mutated")
			tk.ErrMsg = strPtr("ErrMsg was mutated")
		}).OnNoErr()

	trk := track.NewTrack(&track.Request{
		Method: "BadMethod",
	}, &track.Response{
		Status: "BadStatus",
	}, nil)

	aMutator(trk)

	require.Equal(t, "BadMethod has been mutated", trk.Request.Method)
	require.Equal(t, "BadStatus has been mutated", trk.Response.Status)
	require.Equal(t, strPtr("ErrType was mutated"), trk.ErrType)
	require.Equal(t, strPtr("ErrMsg was mutated"), trk.ErrMsg)
}

func Test_Mutator_OnNoErr_WhenErr(t *testing.T) {
	aMutator := track.Mutator(
		func(tk *track.Track) {
			tk.Request.Method = tk.Request.Method + " has been mutated"
			tk.Response.Status = tk.Response.Status + " has been mutated"
			tk.ErrType = strPtr("ErrType was mutated")
			tk.ErrMsg = strPtr("ErrMsg was mutated")
		}).OnNoErr()

	trk := track.NewTrack(&track.Request{
		Method: "BadMethod",
	}, &track.Response{
		Status: "BadStatus",
	}, errors.New("an error"))

	aMutator(trk)

	require.Equal(t, "BadMethod", trk.Request.Method)
	require.Equal(t, "BadStatus", trk.Response.Status)
	require.Equal(t, strPtr("*errors.errorString"), trk.ErrType)
	require.Equal(t, strPtr("an error"), trk.ErrMsg)
}

func Test_Mutator_OnErr_WhenErr(t *testing.T) {
	errorMutator := track.Mutator(
		func(tk *track.Track) {
			tk.Request.Method = tk.Request.Method + " has been mutated"
			tk.Response.Status = tk.Response.Status + " has been mutated"
			tk.ErrType = strPtr("ErrType was mutated")
			tk.ErrMsg = strPtr("ErrMsg was mutated")
		}).OnErr()

	trk := track.NewTrack(&track.Request{
		Method: "BadMethod",
	}, &track.Response{
		Status: "BadStatus",
	}, errors.New("an error"))

	errorMutator(trk)

	require.Equal(t, "BadMethod has been mutated", trk.Request.Method)
	require.Equal(t, "BadStatus has been mutated", trk.Response.Status)
	require.Equal(t, strPtr("ErrType was mutated"), trk.ErrType)
	require.Equal(t, strPtr("ErrMsg was mutated"), trk.ErrMsg)
}

func Test_Mutator_OnErr_WhenNoErr(t *testing.T) {
	errorMutator := track.Mutator(
		func(tk *track.Track) {
			tk.Request.Method = tk.Request.Method + " has been mutated"
			tk.Response.Status = tk.Response.Status + " has been mutated"
			tk.ErrType = strPtr("ErrType was mutated")
			tk.ErrMsg = strPtr("ErrMsg was mutated")
		}).OnErr()

	trk := track.NewTrack(&track.Request{
		Method: "BadMethod",
	}, &track.Response{
		Status: "BadStatus",
	}, nil)

	errorMutator(trk)

	require.Equal(t, "BadMethod", trk.Request.Method)
	require.Equal(t, "BadStatus", trk.Response.Status)
	require.Nil(t, trk.ErrType)
	require.Nil(t, trk.ErrMsg)
}

func strPtr(s string) *string { return &s }
