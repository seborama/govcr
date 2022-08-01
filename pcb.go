package govcr

import (
	"net/http"

	"github.com/seborama/govcr/v5/cassette"
	"github.com/seborama/govcr/v5/cassette/track"
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
}

func (pcb *PrintedCircuitBoard) seekTrack(k7 *cassette.Cassette, httpRequest *http.Request) (*track.Track, error) {
	request := track.ToRequest(httpRequest)

	numberOfTracksInCassette := k7.NumberOfTracks()
	for trackNumber := int32(0); trackNumber < numberOfTracksInCassette; trackNumber++ {
		if pcb.trackMatches(k7, trackNumber, request) {
			return pcb.replayTrack(k7, trackNumber)
		}
	}

	return nil, nil
}

func (pcb *PrintedCircuitBoard) trackMatches(k7 *cassette.Cassette, trackNumber int32, request *track.Request) bool {
	trk := k7.Track(trackNumber)

	return !trk.IsReplayed() && pcb.requestMatcher.Match(request, trk.GetRequest())
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

// AddRecordingMutators adds a collection of recording TrackMutator's.
func (pcb *PrintedCircuitBoard) AddRecordingMutators(mutators ...track.Mutator) {
	pcb.trackRecordingMutators = pcb.trackRecordingMutators.Add(mutators...)
}

// AddReplayingMutators adds a collection of replaying TrackMutator's.
// Replaying happens AFTER the request has been matched. As such, while the track's Request
// could be mutated, it will have no effect.
// However, the Request data can be referenced as part of mutating the Response.
func (pcb *PrintedCircuitBoard) AddReplayingMutators(mutators ...track.Mutator) {
	pcb.trackReplayingMutators = pcb.trackReplayingMutators.Add(mutators...)
}

// RequestMatcher is an interface that exposes the method to perform request comparison.
// request comparison involves the HTTP request and the track request recorded on cassette.
// TODO: there could be a case to have RequestMatchers (plural) that would work akin to track.Mutators.
//       I.e. they could be chained and conditional.
type RequestMatcher interface {
	Match(httpRequest, trackRequest *track.Request) bool
}
