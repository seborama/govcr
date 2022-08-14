package govcr

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/seborama/govcr/v7/cassette"
	"github.com/seborama/govcr/v7/cassette/track"
	"github.com/seborama/govcr/v7/stats"
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
// Note: by convention resp should be nil if an error occurs with HTTP.
func (t *vcrTransport) RoundTrip(httpRequest *http.Request) (*http.Response, error) {
	httpRequestClone := track.CloneHTTPRequest(httpRequest)

	// search for a matching track on cassette if liveOnly mode is not selected
	trk, err := t.pcb.SeekTrack(t.cassette, httpRequestClone)
	if err != nil {
		return nil, errors.Wrap(err, "govcr failed to read matching track from cassette")
	}

	if trk != nil {
		t.pcb.mutateTrackReplaying(trk)

		httpResponse := trk.ToHTTPResponse()
		httpError := trk.ToErr()

		return httpResponse, httpError //nolint: wrapcheck
	}

	if t.pcb.httpMode == HTTPModeOffline {
		return nil, errors.New("no track matched on cassette and offline mode is active")
	}

	httpResponse, reqErr := t.transport.RoundTrip(httpRequest)
	if !t.pcb.readOnly {
		trkResponse := track.ToResponse(httpResponse)
		trkRequest := track.ToRequest(httpRequestClone)
		newTrack := track.NewTrack(trkRequest, trkResponse, reqErr)

		t.pcb.mutateTrackRecording(newTrack)

		if err = cassette.AddTrackToCassette(t.cassette, newTrack); err != nil {
			return nil, errors.Wrap(err, "govcr failed to add track to cassette")
		}
	}

	return httpResponse, errors.WithStack(reqErr)
}

// NumberOfTracks returns the number of tracks contained in the cassette.
func (t *vcrTransport) NumberOfTracks() int32 {
	return t.cassette.NumberOfTracks()
}

func (t *vcrTransport) loadCassette(cassetteName string, opts ...cassette.Option) error {
	if t.cassette != nil {
		return errors.Errorf("failed to load cassette '%s': another cassette ('%s') is already loaded", cassetteName, t.cassette.Name())
	}

	k7 := cassette.LoadCassette(cassetteName, opts...)
	t.cassette = k7

	return nil
}

func (t *vcrTransport) ejectCassette() {
	t.cassette = nil
}

// SetRequestMatcher sets a new RequestMatcher to the VCR.
func (t *vcrTransport) SetRequestMatcher(requestMatcher RequestMatcher) {
	t.pcb.SetRequestMatcher(requestMatcher)
}

// SetReadOnlyMode sets the VCR to read-only mode (true) or to normal read-write (false).
func (t *vcrTransport) SetReadOnlyMode(state bool) {
	t.pcb.SetReadOnlyMode(state)
}

// SetNormalMode sets the VCR to normal HTTP mode.
func (t *vcrTransport) SetNormalMode() {
	t.pcb.SetNormalMode()
}

// SetOfflineMode sets the VCR to offline mode.
func (t *vcrTransport) SetOfflineMode() {
	t.pcb.SetOfflineMode()
}

// SetLiveOnlyMode sets the VCR to live-only mode.
func (t *vcrTransport) SetLiveOnlyMode() {
	t.pcb.SetLiveOnlyMode()
}

// AddRecordingMutators adds a set of recording Track Mutator's to the VCR.
func (t *vcrTransport) AddRecordingMutators(mutators ...track.Mutator) {
	t.pcb.AddRecordingMutators(mutators...)
}

// SetRecordingMutators replaces the set of recording Track Mutator's in the VCR.
func (t *vcrTransport) SetRecordingMutators(trackMutators ...track.Mutator) {
	t.pcb.SetRecordingMutators(trackMutators...)
}

// ClearRecordingMutators clears the set of recording Track Mutator's from the VCR.
func (t *vcrTransport) ClearRecordingMutators() {
	t.pcb.ClearRecordingMutators()
}

// AddReplayingMutators adds a set of replaying Track Mutator's to the VCR.
// Replaying happens AFTER the request has been matched. As such, while the track's Request
// could be mutated, it will have no effect.
// However, the Request data can be referenced as part of mutating the Response.
func (t *vcrTransport) AddReplayingMutators(mutators ...track.Mutator) {
	t.pcb.AddReplayingMutators(mutators...)
}

// SetReplayingMutators replaces the set of replaying Track Mutator's in the VCR.
func (t *vcrTransport) SetReplayingMutators(trackMutators ...track.Mutator) {
	t.pcb.SetReplayingMutators(trackMutators...)
}

// ClearReplayingMutators clears the set of replaying Track Mutator's from the VCR.
func (t *vcrTransport) ClearReplayingMutators() {
	t.pcb.ClearReplayingMutators()
}

func (t *vcrTransport) stats() *stats.Stats {
	return t.cassette.Stats()
}
