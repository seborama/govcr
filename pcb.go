package govcr

import (
	"bytes"
	"log"
	"net/http"
)

// pcb stands for Printed Circuit Board. It is a structure that holds some
// facilities that are passed to the VCR machine to modify its internals.
type pcb struct {
	Transport        http.RoundTripper
	RequestFilter    RequestFilter
	ResponseFilter   ResponseFilter
	TrackFilter      TrackFilter
	Logger           *log.Logger
	DisableRecording bool
	CassettePath     string
}

const trackNotFound = -1

func (pcbr *pcb) seekTrack(cassette *cassette, req *http.Request) (*http.Response, int32, error) {
	filteredRequest, err := newRequest(req, pcbr.Logger)
	if err != nil {
		return nil, trackNotFound, nil
	}
	// See warning in govcr.Request definition comment.
	filteredRequest = pcbr.RequestFilter(filteredRequest)

	numberOfTracksInCassette := cassette.NumberOfTracks()
	for trackNumber := int32(0); trackNumber < numberOfTracksInCassette; trackNumber++ {
		if pcbr.trackMatches(cassette, trackNumber, filteredRequest) {
			pcbr.Logger.Printf("INFO - Cassette '%s' - Found a matching track for %s %s\n", cassette.Name, req.Method, req.URL.String())

			// See warning in govcr.Request definition comment.
			request, err := newRequest(req, pcbr.Logger)
			if err != nil {
				return nil, trackNotFound, nil
			}
			replayedResponse, err := cassette.replayResponse(trackNumber, req)
			filteredResp := pcbr.filterResponse(replayedResponse, request)

			return filteredResp, trackNumber, err
		}
	}

	return nil, trackNotFound, nil
}

// Matches checks whether the track is a match for the supplied request.
func (pcbr *pcb) trackMatches(cassette *cassette, trackNumber int32, request Request) bool {
	track := cassette.Track(trackNumber)

	// apply filter function to track header / body
	filteredTrackRequest := pcbr.RequestFilter(track.Request.Request())

	return !track.replayed &&
		filteredTrackRequest.Method == request.Method &&
		filteredTrackRequest.URL.String() == request.URL.String() &&
		pcbr.headerResembles(filteredTrackRequest.Header, request.Header) &&
		pcbr.bodyResembles(filteredTrackRequest.Body, request.Body)
}

// headerResembles compares HTTP headers for equivalence.
func (pcbr *pcb) headerResembles(header1 http.Header, header2 http.Header) bool {
	for k := range header1 {
		// TODO: a given header may have several values (and in any order)
		if GetFirstValue(header1, k) != GetFirstValue(header2, k) {
			return false
		}
	}

	// finally assert the number of headers match
	return len(header1) == len(header2)
}

// bodyResembles compares HTTP bodies for equivalence.
func (pcbr *pcb) bodyResembles(body1 []byte, body2 []byte) bool {
	return bytes.Equal(body1, body2)
}

// filterResponse applies the PCB ResponseFilter filter functions to the Response.
func (pcbr *pcb) filterResponse(resp *http.Response, req Request) *http.Response {
	body, err := readResponseBody(resp)
	if err != nil {
		pcbr.Logger.Printf("ERROR - Unable to filter response body so leaving it untouched: %s\n", err.Error())
		return resp
	}

	filtResp := Response{
		req:        copyGovcrRequest(&req), // See warning in govcr.Request definition comment.
		Body:       body,
		Header:     cloneHeader(resp.Header),
		StatusCode: resp.StatusCode,
	}
	if pcbr.ResponseFilter != nil {
		// See warning in govcr.Request definition comment, for req.Request.
		filtResp = pcbr.ResponseFilter(filtResp)
	}
	resp.Header = filtResp.Header
	resp.Body = toReadCloser(filtResp.Body)
	resp.StatusCode = filtResp.StatusCode
	resp.Status = http.StatusText(resp.StatusCode)

	return resp
}
