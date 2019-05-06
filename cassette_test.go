package govcr_test

import (
	"bytes"
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

func Test_cassette_GzipFilter(t *testing.T) {
	tests := []struct {
		name         string
		cassetteName string
		tracks       []govcr.Track
		trackData    bytes.Buffer
		want         []byte
		wantErr      bool
	}{
		{
			name:         "Should not compress data",
			cassetteName: "cassette",
			trackData:    *bytes.NewBufferString(`data`),
			want:         []byte(`data`),
			wantErr:      false,
		},
		{
			name:         "Should compress data when cassette name is *.gz",
			cassetteName: "cassette.gz",
			trackData:    *bytes.NewBufferString(`data`),
			want:         []byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 74, 73, 44, 73, 4, 4, 0, 0, 255, 255, 99, 243, 243, 173, 4, 0, 0, 0},
			wantErr:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k7 := govcr.NewCassette(tt.cassetteName)
			for _, track := range tt.tracks {
				k7.AddTrack(&track)
			}

			got, err := k7.GzipFilter(tt.trackData)
			require.Equal(t, tt.wantErr, err != nil)
			assert.EqualValues(t, tt.want, got)
		})
	}
}

func Test_cassette_isLongPlay(t *testing.T) {
	tests := []struct {
		name         string
		cassetteName string
		want         bool
	}{
		{
			name:         "Should detect Long Play cassette (i.e. compressed)",
			cassetteName: "cassette.gz",
			want:         true,
		},
		{
			name:         "Should detect Normal Play cassette (i.e. not compressed)",
			cassetteName: "cassette",
			want:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k7 := govcr.NewCassette(tt.cassetteName)

			got := k7.IsLongPlay()
			assert.EqualValues(t, tt.want, got)
		})
	}
}

func Test_cassette_gunzipFilter(t *testing.T) {
	tests := []struct {
		name         string
		cassetteName string
		tracks       []govcr.Track
		trackData    []byte
		want         []byte
		wantErr      bool
	}{
		{
			name:         "Should not compress data",
			cassetteName: "cassette",
			trackData:    []byte(`data`),
			want:         []byte(`data`),
			wantErr:      false,
		},
		{
			name:         "Should de-compress data when cassette name is *.gz",
			cassetteName: "cassette.gz",
			trackData:    []byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 74, 73, 44, 73, 4, 4, 0, 0, 255, 255, 99, 243, 243, 173, 4, 0, 0, 0},
			want:         []byte(`data`),
			wantErr:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k7 := govcr.NewCassette(tt.cassetteName)
			for _, track := range tt.tracks {
				k7.AddTrack(&track)
			}

			got, err := k7.GunzipFilter(tt.trackData)
			require.Equal(t, tt.wantErr, err != nil)
			assert.EqualValues(t, tt.want, got)
		})
	}
}
