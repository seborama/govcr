package cassette

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/seborama/govcr/v17/cassette/track"
	"github.com/seborama/govcr/v17/compression"
	cryptoerr "github.com/seborama/govcr/v17/encryption/errors"
	govcrerr "github.com/seborama/govcr/v17/errors"
	"github.com/seborama/govcr/v17/fileio"
	"github.com/seborama/govcr/v17/stats"
)

// Cassette contains a set of tracks.
type Cassette struct {
	Tracks []track.Track `json:"Tracks"`

	name            string
	trackSliceMutex sync.RWMutex
	tracksLoaded    int32
	// crypter provides an encryption abstraction for cassette read/write operations.
	crypter Crypter
	// store provides a storage backend abstraction: file system, cloud storage, etc
	store FileIO
}

type FileIO interface {
	MkdirAll(path string, perm os.FileMode) error
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	NotExist(name string) (bool, error)
}

const (
	encryptedCassetteHeaderMarkerV1 = "$ENC$" // legacy aesgcm V1 signature
	encryptedCassetteHeaderMarkerV2 = "$ENC:V2$"
)

// Crypter defines encryption behaviour.
type Crypter interface {
	Encrypt(plaintext []byte) ([]byte, []byte, error)
	Decrypt(ciphertext, nonce []byte) ([]byte, error)
	Kind() string
}

// Option defines a signature for options that can be passed
// to create a new Cassette.
type Option func(*Cassette)

// WithCrypter provides a crypter to encrypt/decrypt cassette content.
func WithCrypter(crypter Crypter) Option {
	return func(k7 *Cassette) {
		if k7.crypter != nil {
			slog.Info("notice: setting a crypter but another one had already been registered - this is incorrect usage")
		}

		k7.crypter = crypter
	}
}

// WithStore provides a dedicated storage engine for the cassette data.
func WithStore(store FileIO) Option {
	return func(k7 *Cassette) {
		if k7.store != nil {
			slog.Info("notice: setting a storer but another one had already been registered - this is incorrect usage")
		}

		k7.store = store
	}
}

// NewCassette creates a ready to use new cassette.
// When no storage backend (store) is provided, the default OSFile storage is used.
func NewCassette(name string, opts ...Option) *Cassette {
	k7 := Cassette{
		name:            name,
		trackSliceMutex: sync.RWMutex{},
	}

	for _, option := range opts {
		option(&k7)
	}

	if k7.store == nil {
		k7.store = &fileio.OSFile{}
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

	return int32(len(k7.Tracks)) //nolint:gosec // int32 can more than sufficiently hold the number of tracks on a cassette.
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

// saveCassette writes a cassette to storage.
func (k7 *Cassette) save() error {
	k7.trackSliceMutex.Lock()
	defer k7.trackSliceMutex.Unlock()

	if k7.store == nil {
		k7.store = &fileio.OSFile{}
	}

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
	if err = k7.store.MkdirAll(path, 0o750); err != nil {
		return errors.Wrap(err, path)
	}

	err = k7.store.WriteFile(k7.name, eData, 0o600)
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

	kindLen := len(k7.crypter.Kind())
	if kindLen > 255 {
		return nil, errors.New("cipher kind is too long, must be 255 max")
	}

	nonceLen := len(nonce)
	if nonceLen > 255 {
		return nil, errors.New("nonce is too long, must be 255 max")
	}

	// first add header
	eData := []byte(encryptedCassetteHeaderMarkerV2)
	eData = append(eData, byte(kindLen))
	eData = append(eData, []byte(k7.crypter.Kind())...)
	eData = append(eData, byte(nonceLen))
	eData = append(eData, nonce...)

	// then add cassette data
	eData = append(eData, ciphertext...)

	return eData, nil
}

// DecryptionFilter decrypts the cassette data if a cryptographer Crypter
// was supplied and the encryption marker is found, otherwise data is left as is.
func (k7 *Cassette) DecryptionFilter(data []byte) ([]byte, error) {
	if !k7.wantEncrypted() {
		if getEncryptionMarker(data) != "" {
			return nil, cryptoerr.NewErrCrypto("cassette has encryption marker prefix but no cryptographer was supplied")
		}

		return data, nil
	}

	if getEncryptionMarker(data) == "" {
		// We're going on the off chance that the cassette file is not encrypted yet
		// but that from next save it needs to be.
		return data, nil
	}

	return Decrypt(data, k7.crypter)
}

// SetCrypter sets the cassette Crypter.
// This can be used to set a cipher when none is present (which already happens automatically
// when loading a cassette) or change the cipher when one is already present.
// The cassette is saved to persist the change with the new selected cipher.
func (k7 *Cassette) SetCrypter(crypter Crypter) error {
	k7.crypter = crypter
	return k7.save()
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

// readCassette reads the cassette source, if present or else nil data.
func (k7 *Cassette) readCassette(cassetteName string) ([]byte, error) {
	if cassetteName == "" {
		return nil, errors.New("a cassette name is required")
	}

	if k7.store == nil {
		k7.store = &fileio.OSFile{}
	}

	if notExist, err := k7.store.NotExist(cassetteName); err != nil {
		return nil, errors.Wrap(err, "failed to check cassette existence")
	} else if notExist {
		return nil, nil // not found, return nil data
	}

	data, err := k7.store.ReadFile(cassetteName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cassette data from source")
	}

	dData, err := k7.DecryptionFilter(data)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	gData, err := k7.GunzipFilter(dData)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return gData, nil
}

func getEncryptionMarker(data []byte) string {
	if len(data) < 3 || data[0] != '$' {
		return ""
	}

	marker := ""
	for i, b := range data[1:] {
		if i > 255 {
			// give up: we should have already met with the closing `$` a long time ago
			break
		}

		if b == '$' {
			marker = string(data[:i+2])
			break
		}
	}

	return marker
}

// Decrypt is a utility function that decrypts the cassette raw data
// with the use of the supplied crypter.
func Decrypt(data []byte, crypter Crypter) ([]byte, error) {
	encMarker := getEncryptionMarker(data)
	markerLen := len(encMarker)

	var nonce []byte

	pos := markerLen

	switch encMarker {
	case encryptedCassetteHeaderMarkerV1:
		// Header V1 (aes gcm only, will automatically convert to V2 on save):
		// - marker ($ENC$)
		// - nonce length (1 byte)
		// - nonce
		nonceLen := int(data[pos])
		pos++

		nonce = data[pos : pos+nonceLen]
		pos += nonceLen

	case encryptedCassetteHeaderMarkerV2:
		// Header V2:
		// - marker
		// - cipher name length (1 byte)
		// - cipher name
		// - nonce length (1 byte)
		// - nonce
		cipherKindLen := int(data[pos])
		pos++

		cipherKind := data[pos : pos+cipherKindLen]
		pos += cipherKindLen
		if string(cipherKind) != crypter.Kind() {
			return nil, errors.Errorf("cassette crypter is '%s' but cassette data indicates '%s'", crypter.Kind(), string(cipherKind))
		}

		nonceLen := int(data[pos])
		pos++

		nonce = data[pos : pos+nonceLen]
		pos += nonceLen

	case "":
		return nil, errors.New("missing encrypted cassette header marker")

	default:
		return nil, errors.Errorf("encrypted cassette header marker not recognised: '%s'", encMarker)
	}

	return crypter.Decrypt(data[pos:], nonce)
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

// LoadCassette loads a cassette from source and initialises its associated stats.
// It panics when a cassette exists but cannot be loaded because that indicates
// corruption (or a severe bug).
func LoadCassette(cassetteName string, opts ...Option) *Cassette {
	k7 := NewCassette(cassetteName, opts...)

	data, err := k7.readCassette(cassetteName)
	if err != nil {
		panic(fmt.Sprintf("unable to load invalid / corrupted cassette '%s': %+v", cassetteName, err))
	}

	if data != nil {
		// NOTE: Properties which are of type 'interface{} / any' are not handled very well
		if err = json.Unmarshal(data, k7); err != nil {
			panic(fmt.Sprintf("failed to interpret cassette data in source '%s': %+v", cassetteName, err))
		}
	}

	// initial stats
	atomic.StoreInt32(&k7.tracksLoaded, k7.NumberOfTracks())

	return k7
}

// DumpCassette loads a cassette from source and returns its (decrypted) contents.
// It panics when a cassette exists but cannot be loaded because that indicates
// corruption (or a severe bug).
func DumpCassette(cassetteName string, opts ...Option) []byte {
	k7 := NewCassette(cassetteName, opts...)

	data, err := k7.readCassette(cassetteName)
	if err != nil {
		panic(fmt.Sprintf("unable to invalid / load corrupted cassette '%s': %+v", cassetteName, err))
	}

	return data
}
