package cassette

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/seborama/govcr/v8/cassette/track"
	"github.com/seborama/govcr/v8/compression"
	cryptoerr "github.com/seborama/govcr/v8/encryption/errors"
	govcrerr "github.com/seborama/govcr/v8/errors"
	"github.com/seborama/govcr/v8/stats"
)

// Cassette contains a set of tracks.
// nolint: govet
type Cassette struct {
	Tracks []track.Track

	name            string
	trackSliceMutex sync.RWMutex
	tracksLoaded    int32
	crypter         Crypter
}

const encryptedCassetteHeader = "$ENC$"

// Crypter defines encryption behaviour.
type Crypter interface {
	Encrypt(plaintext []byte) ([]byte, []byte, error)
	Decrypt(ciphertext, nonce []byte) ([]byte, error)
}

// Option defines a signature for options that can be passed
// to create a new Cassette.
type Option func(*Cassette)

// WithCassetteCrypter provides a crypter to encrypt/decrypt cassette content.
func WithCassetteCrypter(crypter Crypter) Option {
	return func(k7 *Cassette) {
		k7.crypter = crypter
	}
}

// NewCassette creates a ready to use new cassette.
func NewCassette(name string, opts ...Option) *Cassette {
	k7 := Cassette{name: name, trackSliceMutex: sync.RWMutex{}}

	for _, option := range opts {
		option(&k7)
	}

	return &k7
}

// Stats returns the cassette's Stats.
func (k7 *Cassette) Stats() *stats.Stats {
	if k7 == nil {
		return nil
	}

	s := stats.Stats{
		TotalTracks: k7.NumberOfTracks(),
	}
	s.TracksLoaded = atomic.LoadInt32(&k7.tracksLoaded)
	s.TracksRecorded = k7.NumberOfTracks() - s.TracksLoaded
	s.TracksPlayed = k7.tracksPlayed() - s.TracksRecorded

	return &s
}

func (k7 *Cassette) tracksPlayed() int32 {
	replayed := int32(0)

	k7.trackSliceMutex.RLock()
	defer k7.trackSliceMutex.RUnlock()

	for i := range k7.Tracks {
		if k7.Tracks[i].IsReplayed() {
			replayed++
		}
	}

	return replayed
}

// NumberOfTracks returns the number of tracks contained in the cassette.
func (k7 *Cassette) NumberOfTracks() int32 {
	k7.trackSliceMutex.RLock()
	defer k7.trackSliceMutex.RUnlock()

	return int32(len(k7.Tracks))
}

// ReplayTrack returns the specified track number, as recorded on cassette.
func (k7 *Cassette) ReplayTrack(trackNumber int32) (*track.Track, error) {
	if trackNumber < 0 || trackNumber >= k7.NumberOfTracks() {
		return nil, govcrerr.NewErrGoVCR(fmt.Sprintf("invalid track number %d (only %d available) (track #0 stands for first track)", trackNumber, k7.NumberOfTracks()))
	}

	k7.trackSliceMutex.Lock()
	defer k7.trackSliceMutex.Unlock()

	trk := &k7.Tracks[trackNumber]

	// mark the track as replayed so it doesn't get re-used
	trk.SetReplayed(true)

	return trk, nil
}

// AddTrack to cassette.
// Note that the Track does not receive mutations here, it must be mutated
// before passed to the cassette for recording.
func (k7 *Cassette) AddTrack(trk *track.Track) {
	k7.trackSliceMutex.Lock()
	defer k7.trackSliceMutex.Unlock()

	if trk.UUID == "" {
		trk.UUID = uuid.NewString()
	}

	k7.Tracks = append(k7.Tracks, *trk)
}

// IsLongPlay returns true if the cassette content is compressed.
func (k7 *Cassette) IsLongPlay() bool {
	return strings.HasSuffix(k7.name, ".gz")
}

func (k7 *Cassette) wantEncrypted() bool {
	return k7.crypter != nil
}

// saveCassette writes a cassette to file.
func (k7 *Cassette) save() error {
	k7.trackSliceMutex.Lock()
	defer k7.trackSliceMutex.Unlock()

	data, err := json.MarshalIndent(k7, "", "  ")
	if err != nil {
		return errors.WithStack(err)
	}

	// compress before encryption to get better results
	gData, err := k7.GzipFilter(*bytes.NewBuffer(data))
	if err != nil {
		return errors.WithStack(err)
	}

	eData, err := k7.EncryptionFilter(gData)
	if err != nil {
		return errors.WithStack(err)
	}

	path := filepath.Dir(k7.name)
	if err = os.MkdirAll(path, 0o750); err != nil {
		return errors.Wrap(err, path)
	}

	err = os.WriteFile(k7.name, eData, 0o600)
	return errors.Wrap(err, k7.name)
}

// GzipFilter compresses the cassette data in gzip format if the cassette
// is set for compression, otherwise data is left as is.
func (k7 *Cassette) GzipFilter(data bytes.Buffer) ([]byte, error) {
	if k7.IsLongPlay() {
		return compression.Compress(data.Bytes())
	}
	return data.Bytes(), nil
}

// GunzipFilter de-compresses the cassette data from gzip format if the cassette
// is set for compression, otherwise data is left as is.
func (k7 *Cassette) GunzipFilter(data []byte) ([]byte, error) {
	if k7.IsLongPlay() {
		return compression.Decompress(data)
	}
	return data, nil
}

// EncryptionFilter encrypts the cassette data if a cryptographer Crypter
// was supplied, otherwise data is left as is.
func (k7 *Cassette) EncryptionFilter(data []byte) ([]byte, error) {
	if !k7.wantEncrypted() {
		return data, nil
	}

	ciphertext, nonce, err := k7.crypter.Encrypt(data)
	if err != nil {
		return nil, err
	}

	nonceLen := len(nonce)
	if nonceLen > 255 {
		return nil, errors.New("nonce is too long, must be 255 max")
	}

	headerData := []byte(encryptedCassetteHeader)
	headerData = append(headerData, byte(nonceLen))
	headerData = append(headerData, nonce...)

	eData := append(headerData, ciphertext...)

	return eData, nil
}

// DecryptionFilter decrypts the cassette data if a cryptographer Crypter
// was supplied and the encryption marker is found, otherwise data is left as is.
func (k7 *Cassette) DecryptionFilter(data []byte) ([]byte, error) {
	hasEncryptionMarker := bytes.HasPrefix(data, []byte(encryptedCassetteHeader))

	if !k7.wantEncrypted() {
		if hasEncryptionMarker {
			return nil, cryptoerr.NewErrCrypto("cassette has encryption marker but no cryptographer was supplied")
		}

		return data, nil
	}

	if !hasEncryptionMarker {
		// We're going off the chance that the cassette file is not encrypted yet but that from next save it should be.
		return data, nil
	}

	return Decrypt(data, k7.crypter)
}

// Track retrieves the requested track number.
// '0' is the first track.
func (k7 *Cassette) Track(trackNumber int32) track.Track {
	k7.trackSliceMutex.RLock()
	defer k7.trackSliceMutex.RUnlock()

	return k7.Tracks[trackNumber]
}

// Name retrieves cassette name.
func (k7 *Cassette) Name() string {
	return k7.name
}

// readCassetteFile reads the cassette file, if present or
// returns a blank cassette.
func (k7 *Cassette) readCassetteFile(cassetteName string) error {
	data, err := os.ReadFile(cassetteName) // nolint:gosec
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrap(err, "failed to read cassette data from file")
	}

	dData, err := k7.DecryptionFilter(data)
	if err != nil {
		return errors.WithStack(err)
	}

	gData, err := k7.GunzipFilter(dData)
	if err != nil {
		return errors.WithStack(err)
	}

	// NOTE: Properties which are of type 'interface{} / any' are not handled very well
	if err = json.Unmarshal(gData, k7); err != nil {
		return errors.Wrap(err, "failed to interpret cassette data in file")
	}

	return nil
}

// Decrypt is a utility function that decrypts the cassette raw data
// with the use of the supplied crypter.
func Decrypt(data []byte, crypter Crypter) ([]byte, error) {
	hasEncryptionMarker := bytes.HasPrefix(data, []byte(encryptedCassetteHeader))

	if !hasEncryptionMarker {
		return nil, errors.New("encrypted cassette header marker not recognised")
	}

	// Header:
	// - marker
	// - nonce length (1 byte)
	// - nonce
	// - ciphertext

	nonceLen := int(data[len(encryptedCassetteHeader)])
	nonce := data[len(encryptedCassetteHeader)+1 : len(encryptedCassetteHeader)+1+nonceLen]

	headerSize := len(encryptedCassetteHeader) + 1 + len(nonce)

	return crypter.Decrypt(data[headerSize:], nonce)
}

// AddTrackToCassette saves a new track using the specified details to a cassette.
func AddTrackToCassette(cassette *Cassette, trk *track.Track) error {
	// mark track as replayed since it's coming from a live Request!
	trk.SetReplayed(true)

	// add track to cassette
	cassette.AddTrack(trk)

	// save cassette
	return cassette.save()
}

// LoadCassette loads a cassette from file and initialises its associated stats.
// It panics when a cassette exists but cannot be loaded because that indicates
// corruption (or a severe bug).
func LoadCassette(cassetteName string, opts ...Option) *Cassette {
	k7 := NewCassette(cassetteName, opts...)

	err := k7.readCassetteFile(cassetteName)
	if err != nil {
		panic(fmt.Sprintf("unable to load corrupted cassette '%s': %+v", cassetteName, err))
	}

	// initial stats
	atomic.StoreInt32(&k7.tracksLoaded, k7.NumberOfTracks())

	return k7
}
