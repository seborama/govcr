package cassette_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/cassette"
)

func Test_cassette_GzipFilter(t *testing.T) {
	tests := []struct {
		name         string
		cassetteName string
		tracks       []cassette.Track
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
			k7 := cassette.NewCassette(tt.cassetteName)
			for _, track := range tt.tracks {
				k7.AddTrack(&track)
			}

			got, err := k7.GzipFilter(tt.trackData)
			require.Equal(t, tt.wantErr, err != nil)
			assert.EqualValues(t, tt.want, got)
		})
	}
}

func Test_cassette_IsLongPlay(t *testing.T) {
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
			k7 := cassette.NewCassette(tt.cassetteName)

			got := k7.IsLongPlay()
			assert.EqualValues(t, tt.want, got)
		})
	}
}

func Test_cassette_GunzipFilter(t *testing.T) {
	tests := []struct {
		name         string
		cassetteName string
		tracks       []cassette.Track
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
			k7 := cassette.NewCassette(tt.cassetteName)
			for _, track := range tt.tracks {
				k7.AddTrack(&track)
			}

			got, err := k7.GunzipFilter(tt.trackData)
			require.Equal(t, tt.wantErr, err != nil)
			assert.EqualValues(t, tt.want, got)
		})
	}
}
