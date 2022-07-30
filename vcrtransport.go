package govcr

import (
	"log"
	"net/http"

	"github.com/pkg/errors"

	"github.com/seborama/govcr/v5/cassette"
	"github.com/seborama/govcr/v5/cassette/track"
	"github.com/seborama/govcr/v5/stats"
)

// vcrTransport is the heart of VCR. It implements
// http.RoundTripper that wraps over the default
// one provided by Go's http package or a custom one
// if provided when calling NewVCR.
type vcrTransport struct {
	pcb       *PrintedCircuitBoard
	cassette  *cassette.Cassette
	transport http.RoundTripper
}

// RoundTrip is an implementation of http.RoundTripper.
func (t *vcrTransport) RoundTrip(httpRequest *http.Request) (*http.Response, error) {
	// Note: by convention resp should be nil if an error occurs with HTTP
	var httpResponse *http.Response

	httpRequestClone := track.CloneHTTPRequest(httpRequest)
	if response, err := t.pcb.seekTrack(t.cassette, httpRequestClone); response != nil || err != nil {
		// TODO: two thoughts
		//     1- err can be set either from a runtime issue or as the track recorded error in the original HTTP request
		//     2- response is not mutated using pcb.trackReplayingMutators
		return response, err
	}

	httpResponse, reqErr := t.transport.RoundTrip(httpRequest)
	response := track.FromHTTPResponse(httpResponse)

	request := track.FromHTTPRequest(httpRequestClone)

	newTrack := track.NewTrack(request, response, reqErr)
	t.pcb.mutateTrackRecording(newTrack)
	if err := cassette.AddTrackToCassette(t.cassette, newTrack); err != nil {
		// TODO: this should probably be a panic as it's abnormal
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

	k7 := cassette.LoadCassette(cassetteName)
	t.cassette = k7

	return nil
}

func (t *vcrTransport) ejectCassette() {
	t.cassette = nil
}

// AddRecordingMutators adds a set of recording Track Mutator's to the VCR.
func (t *vcrTransport) AddRecordingMutators(mutators ...track.Mutator) {
	t.pcb.AddRecordingMutators(mutators...)
}

// AddReplayingMutators adds a set of replaying Track Mutator's to the VCR.
// Replaying happens AFTER the request has been matched. As such, while the track's Request
// could be mutated, it will have no effect.
// However, the Request data can be referenced as part of mutating the Response.
func (t *vcrTransport) AddReplayingMutators(mutators ...track.Mutator) {
	t.pcb.AddReplayingMutators(mutators...)
}

func (t *vcrTransport) stats() *stats.Stats {
	return t.cassette.Stats()
}
