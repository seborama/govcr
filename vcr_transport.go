package govcr

import (
	"net/http"
)

// vcrTransport is the heart of VCR. It provides
// an http.RoundTripper that wraps over the default
// one provided by Go's http package or a custom one
// if specified when calling NewVCR.
type vcrTransport struct {
	PCB      *pcb
	Cassette *cassette
}

// RoundTrip is an implementation of http.RoundTripper.
func (t *vcrTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Note: by convention resp should be nil if an error occurs with HTTP
	var resp *http.Response

	// copy the request before the body is closed by the HTTP server.
	copiedReq, err := copyRequest(req)
	if err != nil {
		t.PCB.Logger.Println(err)
		return nil, err
	}
	copiedReq, _ = t.PCB.TrackFilter(copiedReq, nil, nil)

	// attempt to use a track from the cassette that matches
	// the request if one exists.
	if filteredResp, trackNumber, err := t.PCB.seekTrack(t.Cassette, copiedReq); trackNumber != trackNotFound {
		// Only the played back response is filtered.
		// The live request and response should NOT EVER be changed!

		// TODO: add a test to confirm err is replaying errors correctly (see cassette_test.go / Test_trackReplaysError)
		return filteredResp, err
	}

	t.PCB.Logger.Printf("INFO - Cassette '%s' - Executing request to live server for %s %s\n", t.Cassette.Name, req.Method, req.URL.String())
	resp, err = t.PCB.Transport.RoundTrip(req)

	if !t.PCB.DisableRecording {
		// the VCR is not in read-only mode so
		copiedResp, errResp := copyResponse(resp)
		if errResp != nil {
			t.PCB.Logger.Println(errResp)
			return nil, errResp
		}
		copiedReq, copiedResp = t.PCB.TrackFilter(copiedReq, copiedResp, err)
		t.PCB.Logger.Printf("INFO - Cassette '%s' - Recording new track for %s %s as %s %s\n", t.Cassette.Name, req.Method, req.URL.String(), copiedReq.Method, copiedReq.URL)
		if err := recordNewTrackToCassette(t.Cassette, copiedReq, copiedResp, err); err != nil {
			t.PCB.Logger.Println(err)
		}
	}

	return resp, err
}
