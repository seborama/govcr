package govcr

import (
	"net/http"

	"github.com/seborama/govcr/cassette"
	"github.com/seborama/govcr/cassette/track"
)

// pcb stands for Printed Circuit Board. It is a structure that holds some
// facilities that are passed to the VCR machine to influence its internal
// behaviour.
type pcb struct {
	requestMatcher         RequestMatcher
	trackRecordingMutators TrackMutators
}

func (pcbr *pcb) seekTrack(k7 *cassette.Cassette, httpRequest *http.Request) (*http.Response, error) {
	request := track.FromHTTPRequest(httpRequest)

	numberOfTracksInCassette := k7.NumberOfTracks()
	for trackNumber := int32(0); trackNumber < numberOfTracksInCassette; trackNumber++ {
		if pcbr.trackMatches(k7, trackNumber, request) {
			return pcbr.replayResponse(k7, trackNumber, httpRequest)
		}
	}
	return nil, nil
}

func (pcbr *pcb) trackMatches(k7 *cassette.Cassette, trackNumber int32, request *track.Request) bool {
	t := k7.Track(trackNumber)

	return !t.IsReplayed() &&
		pcbr.requestMatcher.Match(request, t.GetRequest())
}

func (pcbr *pcb) replayResponse(k7 *cassette.Cassette, trackNumber int32, httpRequest *http.Request) (*http.Response, error) {
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

func (pcbr *pcb) mutateTrack(t *track.Track) {
	pcbr.trackRecordingMutators.Mutate(t)
}

// AddRecordingMutators adds a collection of recording TrackMutator's.
func (pcbr *pcb) AddRecordingMutators(mutators ...TrackMutator) {
	pcbr.trackRecordingMutators = pcbr.trackRecordingMutators.Add(mutators...)
}

// RequestMatcher is an interface that exposes the method to perform request comparison.
// request comparison involves the HTTP request and the track request recorded on cassette.
type RequestMatcher interface {
	Match(httpRequest *track.Request, trackRequest *track.Request) bool
}
