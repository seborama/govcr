package govcr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
)

// cassette contains a set of tracks.
type cassette struct {
	Tracks []Track

	name                  string
	trackSliceMutex       *sync.RWMutex
	tracksLoaded          int32
	trackRecordingMutater TrackRecordingMutater
}

// CassetteOptions defines a signature for Options that can be passed
// to create a new Cassette.
type CassetteOptions func(*cassette)

// WithTrackRecordingMutator is an option used to provide a TrackRecordingMutater
// when creating a new Cassette.
func WithTrackRecordingMutator(mutater TrackRecordingMutater) CassetteOptions {
	return func(k7 *cassette) {
		k7.trackRecordingMutater = mutater
	}
}

// NewCassette creates a ready to use new cassette.
func NewCassette(name string, options ...CassetteOptions) *cassette {
	k7 := cassette{name: name, trackSliceMutex: &sync.RWMutex{}}
	for _, option := range options {
		option(&k7)
	}
	return &k7
}

// Stats returns the cassette's Stats.
func (k7 *cassette) Stats() *Stats {
	if k7 == nil {
		return nil
	}

	stats := Stats{}
	stats.TracksLoaded = atomic.LoadInt32(&k7.tracksLoaded)
	stats.TracksRecorded = k7.NumberOfTracks() - stats.TracksLoaded
	stats.TracksPlayed = k7.tracksPlayed() - stats.TracksRecorded
	return &stats
}

func (k7 *cassette) tracksPlayed() int32 {
	replayed := int32(0)

	k7.trackSliceMutex.RLock()
	defer k7.trackSliceMutex.RUnlock()

	for _, t := range k7.Tracks {
		if t.replayed {
			replayed++
		}
	}

	return replayed
}

// NumberOfTracks returns the number of tracks contained in the cassette.
func (k7 *cassette) NumberOfTracks() int32 {
	if k7 == nil {
		return 0
	}

	k7.trackSliceMutex.RLock()
	defer k7.trackSliceMutex.RUnlock()

	return int32(len(k7.Tracks))
}

func (k7 *cassette) replayResponse(trackNumber int32) (*Response, error) {
	if trackNumber >= k7.NumberOfTracks() {
		return nil, fmt.Errorf("invalid track number %d (only %d available)", trackNumber, k7.NumberOfTracks())
	}

	k7.trackSliceMutex.Lock()
	defer k7.trackSliceMutex.Unlock()

	track := &k7.Tracks[trackNumber]

	// mark the track as replayed so it doesn't get re-used
	track.replayed = true

	return track.response()
}

func (k7 *cassette) AddTrack(track *Track) {
	k7.trackSliceMutex.Lock()
	defer k7.trackSliceMutex.Unlock()

	if k7.trackRecordingMutater != nil {
		k7.trackRecordingMutater.Mutate(track)
	}

	k7.Tracks = append(k7.Tracks, *track)
}

// isLongPlay returns true if the cassette content is compressed.
func (k7 *cassette) IsLongPlay() bool {
	return strings.HasSuffix(k7.name, ".gz")
}

// saveCassette writes a cassette to file.
func (k7 *cassette) save() error {
	k7.trackSliceMutex.Lock()
	defer k7.trackSliceMutex.Unlock()

	data, err := json.MarshalIndent(k7, "", "  ")
	if err != nil {
		return err
	}

	// TODO: this may not be required anymore...
	tData := transformInterfacesInJSON(data)

	gData, err := k7.GzipFilter(*bytes.NewBuffer(tData))
	if err != nil {
		return err
	}

	path := filepath.Dir(k7.name)
	if err := os.MkdirAll(path, 0750); err != nil {
		return err
	}

	return ioutil.WriteFile(k7.name, gData, 0640)
}

// GzipFilter compresses the cassette data in gzip format if the cassette
// name ends with '.gz', otherwise data is left as is (i.e. de-compressed)
func (k7 *cassette) GzipFilter(data bytes.Buffer) ([]byte, error) {
	if k7.IsLongPlay() {
		return compress(data.Bytes())
	}
	return data.Bytes(), nil
}

// GunzipFilter de-compresses the cassette data in gzip format if the cassette
// name ends with '.gz', otherwise data is left as is (i.e. de-compressed)
func (k7 *cassette) GunzipFilter(data []byte) ([]byte, error) {
	if k7.IsLongPlay() {
		return decompress(data)
	}
	return data, nil
}

// Track retrieves the requested track number.
// '0' is the first track.
func (k7 *cassette) Track(trackNumber int32) Track {
	k7.trackSliceMutex.RLock()
	defer k7.trackSliceMutex.RUnlock()
	return k7.Tracks[trackNumber]
}

// transformInterfacesInJSON looks for known properties in the JSON that are defined as interface{}
// in their original Go structure and don't Unmarshal correctly.
//
// Example x509.Certificate.PublicKey:
// When the type is rsa.PublicKey, Unmarshal attempts to map property "N" to a float64 because it is a number.
// However, it really is a big.Int which does not fit float64 and makes Unmarshal fail.
//
// This is not an ideal solution but it works. In the future, we could consider adding a property that
// records the original type and re-creates it post Unmarshal.
func transformInterfacesInJSON(jsonString []byte) []byte {
	// TODO: this may not be required anymore...
	// TODO: precompile this regexp perhaps via a receiver
	regex := regexp.MustCompile(`("PublicKey":{"N":)([0-9]+),`)

	return []byte(regex.ReplaceAllString(string(jsonString), `$1"$2",`))
}

// recordNewTrackToCassette saves a new track to a cassette.
func recordNewTrackToCassette(cassette *cassette, req *Request, resp *Response, httpErr error) error {
	// create track
	track := NewTrack(req, resp, httpErr)

	// mark track as replayed since it's coming from a live request!
	track.replayed = true

	// add track to cassette
	cassette.AddTrack(track)

	// save cassette
	return cassette.save()
}

// LoadCassette loads a cassette from file and initialises
// its associated stats.
func LoadCassette(cassetteName string) (*cassette, error) {
	k7, err := readCassetteFile(cassetteName)
	if err != nil {
		return nil, err
	}

	// initial stats
	k7.tracksLoaded = k7.NumberOfTracks()

	return k7, nil
}

// readCassetteFile reads the cassette file, if present or
// returns a blank cassette.
func readCassetteFile(cassetteName string) (*cassette, error) {
	k7 := NewCassette(cassetteName)

	data, err := ioutil.ReadFile(cassetteName)
	if os.IsNotExist(err) {
		return k7, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to read cassette data from file")
	}

	cData, err := k7.GunzipFilter(data)
	if err != nil {
		return nil, err
	}

	// NOTE: Properties which are of type 'interface{}' are not handled very well
	if err := json.Unmarshal(cData, k7); err != nil {
		return nil, errors.Wrap(err, "failed to interpret cassette data in file")
	}

	return k7, nil
}
