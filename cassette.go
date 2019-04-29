package govcr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
)

// cassette contains a set of tracks.
type cassette struct {
	Tracks []track

	name            string        `json:"-"`
	trackSliceMutex *sync.RWMutex `json:"-"`
	tracksLoaded    int32         `json:"-"`
}

// newCassette creates a ready to use new cassette.
func newCassette(name string) *cassette {
	return &cassette{name: name, trackSliceMutex: &sync.RWMutex{}}
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

func (k7 *cassette) replayResponse(trackNumber int32) (*response, error) {
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

func (k7 *cassette) addTrack(track *track) {
	k7.trackSliceMutex.Lock()
	defer k7.trackSliceMutex.Unlock()

	k7.Tracks = append(k7.Tracks, *track)
}

func (k7 *cassette) save() error {
	k7.trackSliceMutex.Lock()
	defer k7.trackSliceMutex.Unlock()

	data, err := json.MarshalIndent(k7, "", "  ")
	if err != nil {
		return err
	}

	// TODO: this may not be required anymore...
	tData, err := transformInterfacesInJSON(data)
	if err != nil {
		return err
	}

	path := filepath.Dir(k7.name)
	if err := os.MkdirAll(path, 0750); err != nil {
		return err
	}

	// TODO: gzip data
	return ioutil.WriteFile(k7.name, tData, 0640)
}

// Track retrieves the requested track number.
// '0' is the first track.
func (k7 *cassette) Track(trackNumber int32) track {
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
func transformInterfacesInJSON(jsonString []byte) ([]byte, error) {
	// TODO: this may not be required anymore...
	// TODO: precompile this regexp perhaps via a receiver
	regex, err := regexp.Compile(`("PublicKey":{"N":)([0-9]+),`)
	if err != nil {
		return []byte{}, err
	}

	return []byte(regex.ReplaceAllString(string(jsonString), `$1"$2",`)), nil
}

// recordNewTrackToCassette saves a new track to a cassette.
func recordNewTrackToCassette(cassette *cassette, req *request, resp *response, httpErr error) error {
	// create track
	track, err := newTrack(req, resp, httpErr)
	if err != nil {
		return err
	}

	// mark track as replayed since it's coming from a live request!
	track.replayed = true

	// add track to cassette
	cassette.addTrack(track)

	// save cassette
	return cassette.save()
}

func loadCassette(cassetteName string) (*cassette, error) {
	k7, err := readCassetteFromFile(cassetteName)
	if err != nil {
		return nil, err
	}

	// provide an empty cassette as a minimum
	if k7 == nil {
		k7 = newCassette(cassetteName)
	}

	// initial stats
	k7.tracksLoaded = k7.NumberOfTracks()

	return k7, nil
}

// readCassetteFromFile reads the cassette file, if present or
// returns a blank cassette.
func readCassetteFromFile(cassetteName string) (*cassette, error) {
	k7 := newCassette(cassetteName)

	data, err := ioutil.ReadFile(cassetteName)
	if os.IsNotExist(err) {
		return k7, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to read cassette data from file")
	}

	// NOTE: Properties which are of type 'interface{}' are not handled very well
	if err := json.Unmarshal(data, k7); err != nil {
		return nil, errors.Wrap(err, "failed to interpret cassette data in file")
	}

	return k7, nil
}
