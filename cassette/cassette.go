package cassette

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/seborama/govcr/v7/cassette/track"
	"github.com/seborama/govcr/v7/compression"
	"github.com/seborama/govcr/v7/stats"
)

// Cassette contains a set of tracks.
type Cassette struct {
	Tracks []track.Track

	name            string
	trackSliceMutex sync.RWMutex
	tracksLoaded    int32
}

// Options defines a signature for Options that can be passed
// to create a new Cassette.
type Options func(*Cassette)

// NewCassette creates a ready to use new cassette.
func NewCassette(name string, options ...Options) *Cassette {
	k7 := Cassette{name: name, trackSliceMutex: sync.RWMutex{}}
	for _, option := range options {
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

	for _, t := range k7.Tracks {
		if t.IsReplayed() {
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
	if trackNumber >= k7.NumberOfTracks() {
		//nolint: goerr113
		return nil, fmt.Errorf("invalid track number %d (only %d available) (track #0 stands for first track)", trackNumber, k7.NumberOfTracks())
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
// This is simply based on the extension of the cassette filename.
func (k7 *Cassette) IsLongPlay() bool {
	return strings.HasSuffix(k7.name, ".gz")
}

// saveCassette writes a cassette to file.
func (k7 *Cassette) save() error {
	k7.trackSliceMutex.Lock()
	defer k7.trackSliceMutex.Unlock()

	data, err := json.MarshalIndent(k7, "", "  ")
	if err != nil {
		return errors.WithStack(err)
	}

	gData, err := k7.GzipFilter(*bytes.NewBuffer(data))
	if err != nil {
		return errors.WithStack(err)
	}

	path := filepath.Dir(k7.name)
	if err := os.MkdirAll(path, 0o750); err != nil {
		return errors.Wrap(err, path)
	}

	err = ioutil.WriteFile(k7.name, gData, 0o640)
	return errors.Wrap(err, k7.name)
}

// GzipFilter compresses the cassette data in gzip format if the cassette
// name ends with '.gz', otherwise data is left as is (i.e. de-compressed).
func (k7 *Cassette) GzipFilter(data bytes.Buffer) ([]byte, error) {
	if k7.IsLongPlay() {
		return compression.Compress(data.Bytes())
	}
	return data.Bytes(), nil
}

// GunzipFilter de-compresses the cassette data in gzip format if the cassette
// name ends with '.gz', otherwise data is left as is (i.e. de-compressed).
func (k7 *Cassette) GunzipFilter(data []byte) ([]byte, error) {
	if k7.IsLongPlay() {
		return compression.Decompress(data)
	}
	return data, nil
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
// It panics when a cassette exists but cannot be loaded because that indicates corruption
// (or a severe bug).
func LoadCassette(cassetteName string) *Cassette {
	k7, err := readCassetteFile(cassetteName)
	if err != nil {
		panic(fmt.Sprintf("unable to load corrupted cassette '%s': %v", cassetteName, err))
	}

	// initial stats
	atomic.StoreInt32(&k7.tracksLoaded, k7.NumberOfTracks())

	return k7
}

// readCassetteFile reads the cassette file, if present or
// returns a blank cassette.
func readCassetteFile(cassetteName string) (*Cassette, error) {
	k7 := NewCassette(cassetteName)

	data, err := ioutil.ReadFile(cassetteName) //nolint:gosec
	if os.IsNotExist(err) {
		return k7, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to read cassette data from file")
	}

	cData, err := k7.GunzipFilter(data)
	if err != nil {
		return nil, err
	}

	// NOTE: Properties which are of type 'interface{} / any' are not handled very well
	if err := json.Unmarshal(cData, k7); err != nil {
		return nil, errors.Wrap(err, "failed to interpret cassette data in file")
	}

	return k7, nil
}
