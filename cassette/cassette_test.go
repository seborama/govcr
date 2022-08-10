package cassette_test

import (
	"bytes"
	"encoding/base64"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/v7/cassette"
	"github.com/seborama/govcr/v7/cassette/track"
	"github.com/seborama/govcr/v7/encryption"
)

func Test_cassette_GzipFilter(t *testing.T) {
	tests := []struct {
		name         string
		cassetteName string
		tracks       []track.Track
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
			for _, aTrack := range tt.tracks {
				k7.AddTrack(&aTrack)
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
		tracks       []track.Track
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
			for _, aTrack := range tt.tracks {
				k7.AddTrack(&aTrack)
			}

			got, err := k7.GunzipFilter(tt.trackData)
			require.Equal(t, tt.wantErr, err != nil)
			assert.EqualValues(t, tt.want, got)
		})
	}
}

func Test_cassette_Encryption(t *testing.T) {
	const cassetteName = "temp-fixtures/Test_cassette_Encryption"

	_ = os.Remove(cassetteName)

	keyB64 := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))
	c, err := encryption.NewAESCGM(keyB64, nil)
	require.NoError(t, err)

	k7 := cassette.NewCassette(cassetteName, cassette.WithCassetteCrypter(c))

	trk := &track.Track{}

	err = cassette.AddTrackToCassette(k7, trk)
	require.NoError(t, err)

	var k8 *cassette.Cassette
	require.NotPanics(t, func() {
		k8 = cassette.LoadCassette(cassetteName, cassette.WithCassetteCrypter(c))
	})

	data, err := os.ReadFile(cassetteName) //nolint:gosec
	require.NoError(t, err)

	const encryptedCassetteHeader = "$ENC$"

	require.True(t, bytes.HasPrefix(data, []byte(encryptedCassetteHeader)))

	nonceLen := int(data[len(encryptedCassetteHeader)])
	nonce := data[len(encryptedCassetteHeader)+1 : len(encryptedCassetteHeader)+1+nonceLen]

	t.Logf("nonce: %x\n", nonce)

	require.Equal(t, k7.NumberOfTracks(), k8.NumberOfTracks())

	for i := range k8.Tracks {
		k8.Tracks[i].SetReplayed(true) // so to match k7
	}

	require.Equal(t, k7.Tracks, k8.Tracks)
}
