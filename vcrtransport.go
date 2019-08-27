package govcr

import (
	"log"
	"net/http"

	"github.com/pkg/errors"

	"github.com/seborama/govcr/cassette"
	"github.com/seborama/govcr/cassette/track"
	"github.com/seborama/govcr/stats"
)

// vcrTransport is the heart of VCR. It implements
// http.RoundTripper that wraps over the default
// one provided by Go's http package or a custom one
// if provided when calling NewVCR.
type vcrTransport struct {
	pcb       *pcb
	cassette  *cassette.Cassette
	transport http.RoundTripper
}

// RoundTrip is an implementation of http.RoundTripper.
func (t *vcrTransport) RoundTrip(httpRequest *http.Request) (*http.Response, error) {
	// Note: by convention resp should be nil if an error occurs with HTTP
	var httpResponse *http.Response

	httpRequestClone := track.CloneHTTPRequest(httpRequest)
	if response, err := t.pcb.seekTrack(t.cassette, httpRequestClone); response != nil || err != nil {
		return response, err
	}

	httpResponse, reqErr := t.transport.RoundTrip(httpRequest)
	response := track.FromHTTPResponse(httpResponse)

	request := track.FromHTTPRequest(httpRequestClone)

	newTrack := track.NewTrack(request, response, reqErr)
	t.pcb.mutateTrack(newTrack)
	if err := cassette.AddTrackToCassette(t.cassette, newTrack); err != nil {
		log.Printf("RoundTrip failed to AddTrackToCassette: %v\n", err)
	}

	return httpResponse, reqErr
}

// NumberOfTracks returns the number of tracks contained in the cassette.
func (t *vcrTransport) NumberOfTracks() int32 {
	return t.cassette.NumberOfTracks()
}

func (t *vcrTransport) loadCassette(cassetteName string) error {
	if t.cassette != nil {
		return errors.Errorf("failed to load cassette '%s': another cassette ('%s') is already loaded", cassetteName, t.cassette.Name())
	}

	k7, err := cassette.LoadCassette(cassetteName)
	if err != nil {
		return errors.Wrapf(err, "failed to load contents of cassette '%s'", cassetteName)
	}

	t.cassette = k7

	return nil
}

func (t *vcrTransport) ejectCassette() {
	t.cassette = nil
}

// AddRecordingMutators adds a set of recording TrackMutator's to the VCR.
func (t *vcrTransport) AddRecordingMutators(mutators ...TrackMutator) {
	t.pcb.AddRecordingMutators(mutators...)
}

func (t *vcrTransport) stats() *stats.Stats {
	return t.cassette.Stats()
}
