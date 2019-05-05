package govcr

import (
	"net/http"
)

// pcb stands for Printed Circuit Board. It is a structure that holds some
// facilities that are passed to the VCR machine to influence its internal
// behaviour.
type pcb struct{}

func (pcbr *pcb) seekTrack(k7 *cassette, httpRequest *http.Request) (*http.Response, error) {
	request := fromHTTPRequest(httpRequest)

	numberOfTracksInCassette := k7.NumberOfTracks()
	for trackNumber := int32(0); trackNumber < numberOfTracksInCassette; trackNumber++ {
		if pcbr.trackMatches(k7, trackNumber, request) {
			return pcbr.replayResponse(k7, trackNumber, httpRequest)
		}
	}
	return nil, nil
}

func (pcbr *pcb) trackMatches(k7 *cassette, trackNumber int32, request *Request) bool {
	track := k7.Track(trackNumber)

	return !track.replayed &&
		DefaultRequestMatcher(request, &track.Request)
}

func (pcbr *pcb) replayResponse(k7 *cassette, trackNumber int32, httpRequest *http.Request) (*http.Response, error) {
	replayedResponse, err := k7.replayResponse(trackNumber)

	var httpResponse *http.Response

	if replayedResponse != nil {
		httpResponse = toHTTPResponse(replayedResponse)
		// See notes on http.Response.Request - Body is nil because it has already been consumed
		httpResponse.Request = cloneHTTPRequest(httpRequest)
		httpResponse.Request.Body = nil
	}

	return httpResponse, err
}
