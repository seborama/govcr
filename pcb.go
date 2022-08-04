package govcr

import (
	"net/http"

	"github.com/seborama/govcr/v6/cassette"
	"github.com/seborama/govcr/v6/cassette/track"
)

// PrintedCircuitBoard is a structure that holds some facilities that are passed to
// the VCR machine to influence its internal behaviour.
type PrintedCircuitBoard struct {
	requestMatcher RequestMatcher

	// These mutators are applied before saving a track to a cassette.
	trackRecordingMutators track.Mutators

	// Replaying happens AFTER the request has been matched. As such, while the track's Request
	// could be mutated, it will have no effect.
	// However, the Request data can be referenced as part of mutating the Response.
	trackReplayingMutators track.Mutators

	// Make live calls only, do not replay from cassette even if a track would exist.
	// Perhaps more useful when used in combination with 'readOnly' to by-pass govcr entirely.
	// TODO: note it probably does not make sense to have Offline true and LiveOnly true
	liveOnly bool

	// Replay tracks from cassette, if present, or make live calls but do not records new tracks.
	readOnly bool

	// Replay tracks from cassette, if present, but do not make live calls.
	// govcr will return a transport error if no track was found.
	// TODO: note it probably does not make sense to have Offline true and LiveOnly true
	offlineMode bool
}

func (pcb *PrintedCircuitBoard) seekTrack(k7 *cassette.Cassette, httpRequest *http.Request) (*track.Track, error) {
	if pcb.liveOnly {
		return nil, nil
	}

	request := track.ToRequest(httpRequest)

	numberOfTracksInCassette := k7.NumberOfTracks()
	for trackNumber := int32(0); trackNumber < numberOfTracksInCassette; trackNumber++ {
		if pcb.trackMatches(k7, trackNumber, request) {
			return pcb.replayTrack(k7, trackNumber)
		}
	}

	return nil, nil
}

func (pcb *PrintedCircuitBoard) trackMatches(k7 *cassette.Cassette, trackNumber int32, httpRequest *track.Request) bool {
	trk := k7.Track(trackNumber)

	// protect the original objects against mutation by the matcher
	httpRequestClone := httpRequest.Clone()
	trackReqClone := trk.Request.Clone()

	return !trk.IsReplayed() && pcb.requestMatcher.Match(httpRequestClone, trackReqClone)
}

func (pcb *PrintedCircuitBoard) replayTrack(k7 *cassette.Cassette, trackNumber int32) (*track.Track, error) {
	return k7.ReplayTrack(trackNumber)
}

func (pcb *PrintedCircuitBoard) mutateTrackRecording(t *track.Track) {
	pcb.trackRecordingMutators.Mutate(t)
}

func (pcb *PrintedCircuitBoard) mutateTrackReplaying(t *track.Track) {
	pcb.trackReplayingMutators.Mutate(t)
}

// SetRequestMatcher sets a new RequestMatcher to the VCR.
func (pcb *PrintedCircuitBoard) SetRequestMatcher(requestMatcher RequestMatcher) {
	pcb.requestMatcher = requestMatcher
}

// SetReadOnlyMode sets the VCR to read-only mode (true) or to normal read-write (false).
func (pcb *PrintedCircuitBoard) SetReadOnlyMode(state bool) {
	pcb.readOnly = state
}

// SetOfflineMode sets the VCR to offline mode (true) or to normal live/replay (false).
func (pcb *PrintedCircuitBoard) SetOfflineMode(state bool) {
	pcb.offlineMode = state
}

// SetLiveOnlyMode sets the VCR to live-only mode (true) or to normal live/replay (false).
func (pcb *PrintedCircuitBoard) SetLiveOnlyMode(state bool) {
	pcb.liveOnly = state
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

// RequestMatcher is an interface that exposes the method to perform request comparison.
// request comparison involves the HTTP request and the track request recorded on cassette.
// TODO: there could be a case to have RequestMatchers (plural) that would work akin to track.Mutators.
//       I.e. they could be chained and conditional.
type RequestMatcher interface {
	Match(httpRequest, trackRequest *track.Request) bool
}
