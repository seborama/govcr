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
// Note: by convention resp should be nil if an error occurs with HTTP
func (t *vcrTransport) RoundTrip(httpRequest *http.Request) (*http.Response, error) {
	httpRequestClone := track.CloneHTTPRequest(httpRequest)

	trk, seekErr := t.pcb.seekTrack(t.cassette, httpRequestClone)
	if seekErr != nil {
		log.Printf("error retrieving track from cassette, continuing with live request (will not record): %v", seekErr)
	} else {
		if trk != nil {
			t.pcb.mutateTrackReplaying(trk)

			httpResponse := trk.ToHTTPResponse()
			httpError := trk.ToErr()

			return httpResponse, httpError //nolint: wrapcheck
		}
	}

	httpResponse, reqErr := t.transport.RoundTrip(httpRequest)
	if seekErr == nil {
		// record track only if previously seek cassette
		trkResponse := track.ToResponse(httpResponse)
		trkRequest := track.ToRequest(httpRequestClone)
		newTrack := track.NewTrack(trkRequest, trkResponse, reqErr)

		t.pcb.mutateTrackRecording(newTrack)

		if err := cassette.AddTrackToCassette(t.cassette, newTrack); err != nil {
			// TODO: this should probably be a panic as it's abnormal
			log.Printf("RoundTrip failed to AddTrackToCassette: %v", err)
		}
	}

	return httpResponse, reqErr //nolint: wrapcheck
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
