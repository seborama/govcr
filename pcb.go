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

func (pcb *PrintedCircuitBoard) seekTrack(k7 *cassette.Cassette, httpRequest *http.Request) (*http.Response, error) {
	request := track.FromHTTPRequest(httpRequest)

	numberOfTracksInCassette := k7.NumberOfTracks()
	for trackNumber := int32(0); trackNumber < numberOfTracksInCassette; trackNumber++ {
		if pcb.trackMatches(k7, trackNumber, request) {
			return pcb.replayResponse(k7, trackNumber, httpRequest)
		}
	}
	return nil, nil
}

func (pcb *PrintedCircuitBoard) trackMatches(k7 *cassette.Cassette, trackNumber int32, request *track.Request) bool {
	t := k7.Track(trackNumber)

	return !t.IsReplayed() &&
		pcb.requestMatcher.Match(request, t.GetRequest())
}

func (pcb *PrintedCircuitBoard) replayResponse(k7 *cassette.Cassette, trackNumber int32, httpRequest *http.Request) (*http.Response, error) {
	replayedResponse, err := k7.ReplayResponse(trackNumber)

	var httpResponse *http.Response

	if replayedResponse != nil {
		httpResponse = track.ToHTTPResponse(replayedResponse)
		// See notes on http.response.request - Body is nil because it has already been consumed
		httpResponse.Request = track.CloneHTTPRequest(httpRequest)
		httpResponse.Request.Body = nil
	}

	return httpResponse, err
}

func (pcb *PrintedCircuitBoard) mutateTrackRecording(t *track.Track) {
	pcb.trackRecordingMutators.Mutate(t)
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
	Match(httpRequest *track.Request, trackRequest *track.Request) bool
}
