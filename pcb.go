package govcr

import (
	"net/http"

	"github.com/seborama/govcr/v13/cassette"
	"github.com/seborama/govcr/v13/cassette/track"
)

// HTTPMode defines govcr's mode for HTTP requests.
// See specific modes for further details.
type HTTPMode int

const (
	// HTTPModeNormal replays from cassette if a match exists or execute live request.
	HTTPModeNormal HTTPMode = iota

	// HTTPModeLiveOnly executes live calls for all requests, ignores cassette.
	HTTPModeLiveOnly

	// HTTPModeOffline, plays back from cassette or if no match, return a transport error.
	HTTPModeOffline
)

// PrintedCircuitBoard is a structure that holds some facilities that are passed to
// the VCR machine to influence its internal behaviour.
type PrintedCircuitBoard struct {
	requestMatchers RequestMatchers

	// These mutators are applied before saving a track to a cassette.
	trackRecordingMutators track.Mutators

	// Replaying happens AFTER the request has been matched. As such, while the track's Request
	// could be mutated, it will have no effect.
	// However, the Request data can be referenced as part of mutating the Response.
	trackReplayingMutators track.Mutators

	// httpMode govcr's mode for HTTP request - see httpMode for details.
	httpMode HTTPMode

	// Replay tracks from cassette, if present, or make live calls but do not records new tracks.
	readOnly bool
}

func (pcb *PrintedCircuitBoard) SeekTrack(k7 *cassette.Cassette, httpRequest *http.Request) (*track.Track, error) {
	if pcb.httpMode == HTTPModeLiveOnly {
		return nil, nil
	}

	request := track.ToRequest(httpRequest)

	numberOfTracksInCassette := k7.NumberOfTracks()
	for trackNumber := int32(0); trackNumber < numberOfTracksInCassette; trackNumber++ {
		if pcb.trackMatches(k7, trackNumber, request) {
			currentReq := track.ToRequest(httpRequest)
			return pcb.replayTrack(k7, trackNumber, currentReq)
		}
	}

	return nil, nil
}

func (pcb *PrintedCircuitBoard) trackMatches(k7 *cassette.Cassette, trackNumber int32, httpRequest *track.Request) bool {
	trk := k7.Track(trackNumber)

	// protect the original objects against mutation by the matcher
	httpRequestClone := httpRequest.Clone()
	trackReqClone := trk.Request.Clone()

	return !trk.IsReplayed() && pcb.requestMatchers.Match(httpRequestClone, trackReqClone)
}

func (pcb *PrintedCircuitBoard) replayTrack(k7 *cassette.Cassette, trackNumber int32, httpRequest *track.Request) (*track.Track, error) {
	trk, err := k7.ReplayTrack(trackNumber)
	if err != nil {
		return nil, err
	}

	// protect the original objects against mutation by the matcher
	httpRequestClone := httpRequest.Clone()

	// inject current request into Response.Request
	if trk.Response != nil {
		trk.Response.Request = httpRequestClone
	}

	return trk, nil
}

func (pcb *PrintedCircuitBoard) mutateTrackRecording(t *track.Track) {
	pcb.trackRecordingMutators.Mutate(t)
}

func (pcb *PrintedCircuitBoard) mutateTrackReplaying(t *track.Track) {
	pcb.trackReplayingMutators.Mutate(t)
}

// SetRequestMatchers sets a collection of RequestMatcher's.
func (pcb *PrintedCircuitBoard) SetRequestMatchers(requestMatchers ...RequestMatcher) {
	pcb.requestMatchers = requestMatchers
}

// AddRequestMatchers adds a collection of RequestMatcher's.
func (pcb *PrintedCircuitBoard) AddRequestMatchers(requestMatchers ...RequestMatcher) {
	pcb.requestMatchers = pcb.requestMatchers.Add(requestMatchers...)
}

// SetReadOnlyMode sets the VCR to read-only mode (true) or to normal read-write (false).
func (pcb *PrintedCircuitBoard) SetReadOnlyMode(state bool) {
	pcb.readOnly = state
}

// SetNormalMode sets the VCR to normal HTTP mode.
func (pcb *PrintedCircuitBoard) SetNormalMode() {
	pcb.httpMode = HTTPModeNormal
}

// SetOfflineMode sets the VCR to offline mode.
func (pcb *PrintedCircuitBoard) SetOfflineMode() {
	pcb.httpMode = HTTPModeOffline
}

// SetLiveOnlyMode sets the VCR to live-only mode.
func (pcb *PrintedCircuitBoard) SetLiveOnlyMode() {
	pcb.httpMode = HTTPModeLiveOnly
}

// AddRecordingMutators adds a collection of recording TrackMutator's.
func (pcb *PrintedCircuitBoard) AddRecordingMutators(mutators ...track.Mutator) {
	pcb.trackRecordingMutators = pcb.trackRecordingMutators.Add(mutators...)
}

// SetRecordingMutators replaces the set of recording Track Mutator's in the VCR.
func (pcb *PrintedCircuitBoard) SetRecordingMutators(trackMutators ...track.Mutator) {
	pcb.trackRecordingMutators = trackMutators
}

// ClearRecordingMutators clears the set of recording Track Mutator's from the VCR.
func (pcb *PrintedCircuitBoard) ClearRecordingMutators() {
	pcb.trackRecordingMutators = nil
}

// AddReplayingMutators adds a collection of replaying TrackMutator's.
// Replaying happens AFTER the request has been matched. As such, while the track's Request
// could be mutated, it will have no effect.
// However, the Request data can be referenced as part of mutating the Response.
func (pcb *PrintedCircuitBoard) AddReplayingMutators(mutators ...track.Mutator) {
	pcb.trackReplayingMutators = pcb.trackReplayingMutators.Add(mutators...)
}

// SetReplayingMutators replaces the set of replaying Track Mutator's in the VCR.
func (pcb *PrintedCircuitBoard) SetReplayingMutators(trackMutators ...track.Mutator) {
	pcb.trackReplayingMutators = trackMutators
}

// ClearReplayingMutators clears the set of replaying Track Mutator's from the VCR.
func (pcb *PrintedCircuitBoard) ClearReplayingMutators() {
	pcb.trackReplayingMutators = nil
}
