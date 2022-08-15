package cassette_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/v8/cassette"
	"github.com/seborama/govcr/v8/cassette/track"
	"github.com/seborama/govcr/v8/encryption"
)

func Test_cassette_GzipFilter(t *testing.T) {
	tt := []*struct {
		name         string
		cassetteName string
		tracks       []*track.Track
		trackData    bytes.Buffer
		want         []byte
	}{
		{
			name:         "Should not compress data",
			cassetteName: "cassette",
			trackData:    *bytes.NewBufferString(`data`),
			want:         []byte(`data`),
		},
		{
			name:         "Should compress data when cassette name is *.gz",
			cassetteName: "cassette.gz",
			trackData:    *bytes.NewBufferString(`data`),
			want:         []byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 74, 73, 44, 73, 4, 4, 0, 0, 255, 255, 99, 243, 243, 173, 4, 0, 0, 0},
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			k7 := cassette.NewCassette(tc.cassetteName)
			for _, aTrack := range tc.tracks {
				k7.AddTrack(aTrack)
			}

			got, err := k7.GzipFilter(tc.trackData)
			require.NoError(t, err)
			assert.EqualValues(t, tc.want, got)
		})
	}
}

func Test_cassette_IsLongPlay(t *testing.T) {
	tt := []*struct {
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

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			k7 := cassette.NewCassette(tc.cassetteName)

			got := k7.IsLongPlay()
			assert.EqualValues(t, tc.want, got)
		})
	}
}

func Test_cassette_GunzipFilter(t *testing.T) {
	tt := []*struct {
		name         string
		cassetteName string
		tracks       []*track.Track
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

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			k7 := cassette.NewCassette(tc.cassetteName)
			for i := range tc.tracks {
				k7.AddTrack(tc.tracks[i])
			}

			got, err := k7.GunzipFilter(tc.trackData)
			require.Equal(t, tc.wantErr, err != nil)
			assert.EqualValues(t, tc.want, got)
		})
	}
}

func Test_cassette_Encryption(t *testing.T) {
	const cassetteName = "temp-fixtures/Test_cassette_Encryption"

	_ = os.Remove(cassetteName)

	// STEP 1: create encrypted cassette.
	key := []byte("12345678901234567890123456789012")
	c, err := encryption.NewAESCGM(key, nil)
	require.NoError(t, err)

	k7 := cassette.NewCassette(cassetteName, cassette.WithCassetteCrypter(c))

	trk := &track.Track{}

	err = cassette.AddTrackToCassette(k7, trk)
	require.NoError(t, err)

	// STEP 2: ensure cassette loads.
	var k8 *cassette.Cassette
	require.NotPanics(t, func() {
		k8 = cassette.LoadCassette(cassetteName, cassette.WithCassetteCrypter(c))
	})

	// STEP 3: perform high and low-level validation checks on cassette file.
	data, err := os.ReadFile(cassetteName) // nolint:gosec
	require.NoError(t, err)

	const encryptedCassetteHeader = "$ENC$"

	require.True(t, bytes.HasPrefix(data, []byte(encryptedCassetteHeader)))

	nonceLen := int(data[len(encryptedCassetteHeader)])
	nonce := data[len(encryptedCassetteHeader)+1 : len(encryptedCassetteHeader)+1+nonceLen]

	t.Logf("nonce: %x\n", nonce)

	require.Equal(t, k7.NumberOfTracks(), k8.NumberOfTracks())

	for i := range k7.Tracks {
		k7.Tracks[i].SetReplayed(false) // so to match k8
	}

	require.Equal(t, k7.Tracks, k8.Tracks)
}

func Test_cassette_CanEncryptPlainCassette(t *testing.T) {
	const cassetteName = "temp-fixtures/Test_cassette_CanEncryptPlainCassette"

	_ = os.Remove(cassetteName)

	// STEP 1a: create a non-encrypted cassette.
	// This is not required for cassette encryption, this is for the purpose of confirming
	// that a non-encrypted cassette will convert to an encrypted cassette seamlessly.
	k7 := cassette.NewCassette(cassetteName)

	trk := &track.Track{UUID: "trk-1"}

	err := cassette.AddTrackToCassette(k7, trk)
	require.NoError(t, err)

	// STEP 1b: add track to cassette, this time encrypt the cassette.
	key := []byte("12345678901234567890123456789012")
	c, err := encryption.NewAESCGM(key, nil)
	require.NoError(t, err)

	k7 = cassette.LoadCassette(cassetteName, cassette.WithCassetteCrypter(c))

	trk = &track.Track{UUID: "trk-2"}

	err = cassette.AddTrackToCassette(k7, trk)
	require.NoError(t, err)

	// STEP 2: ensure cassette loads.
	var k8 *cassette.Cassette
	require.NotPanics(t, func() {
		k8 = cassette.LoadCassette(cassetteName, cassette.WithCassetteCrypter(c))
	})

	// STEP 3: perform high and low-level validation checks on cassette file.
	data, err := os.ReadFile(cassetteName) // nolint:gosec
	require.NoError(t, err)

	const encryptedCassetteHeader = "$ENC$"

	require.True(t, bytes.HasPrefix(data, []byte(encryptedCassetteHeader)))

	nonceLen := int(data[len(encryptedCassetteHeader)])
	nonce := data[len(encryptedCassetteHeader)+1 : len(encryptedCassetteHeader)+1+nonceLen]

	t.Logf("nonce: %x\n", nonce)

	require.Equal(t, k7.NumberOfTracks(), k8.NumberOfTracks())

	for i := range k7.Tracks {
		k7.Tracks[i].SetReplayed(false) // so to match k8
	}

	require.Equal(t, k7.Tracks, k8.Tracks)
}
