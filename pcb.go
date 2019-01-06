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
	Logger           *log.Logger
	DisableRecording bool
	CassettePath     string
}

const trackNotFound = -1

func (pcbr *pcb) seekTrack(cassette *cassette, req Request) int {
	for idx := range cassette.Tracks {
		if pcbr.trackMatches(cassette, idx, req) {
			pcbr.Logger.Printf("INFO - Cassette '%s' - Found a matching track for %s %s\n", cassette.Name, req.Method, req.URL.String())
			return idx
		}
	}

	return trackNotFound
}

// Matches checks whether the track is a match for the supplied request.
func (pcbr *pcb) trackMatches(cassette *cassette, trackNumber int, req Request) bool {
	track := cassette.Tracks[trackNumber]

	// apply filter function to track header / body
	filteredTrackRequest := pcbr.RequestFilter(track.Request.Request())

	// apply filter function to request header / body
	filteredReq := pcbr.RequestFilter(req)

	return !track.replayed &&
		filteredTrackRequest.Method == req.Method &&
		filteredTrackRequest.URL.String() == req.URL.String() &&
		pcbr.headerResembles(filteredTrackRequest.Header, filteredReq.Header) &&
		pcbr.bodyResembles(filteredTrackRequest.Body, filteredReq.Body)
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

func (pcbr *pcb) filterResponse(resp *http.Response, req Request) *http.Response {
	body, err := readResponseBody(resp)
	if err != nil {
		pcbr.Logger.Printf("ERROR - Unable to filter response body so leaving it untouched: %s\n", err.Error())
		return resp
	}

	filtResp := Response{
		req:        req,
		Body:       body,
		Header:     cloneHeader(resp.Header),
		StatusCode: resp.StatusCode,
	}
	if pcbr.ResponseFilter != nil {
		filtResp = pcbr.ResponseFilter(filtResp)
	}
	resp.Header = filtResp.Header
	resp.Body = toReadCloser(filtResp.Body)
	resp.StatusCode = filtResp.StatusCode
	resp.Status = http.StatusText(resp.StatusCode)

	return resp
}
