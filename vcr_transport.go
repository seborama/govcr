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

	if err := t.recordNewTrackToCassette(copiedReq, resp, err); err != nil {
		t.PCB.Logger.Println(err)
	}

	return resp, err
}

// recordNewTrackToCassette saves a new track to the cassette in the vcr.
func (t *vcrTransport) recordNewTrackToCassette(req *http.Request, resp *http.Response, httpErr error) error {
	// create track
	track, err := newTrack(req, resp, httpErr)
	if err != nil {
		return err
	}

	// mark track as replayed since it's coming from a live request!
	track.replayed = true

	// Apply save filter.
	if t.PCB.SaveFilter != nil {
		org := track.Response.Response(track.Request.Request())
		filtered := t.PCB.SaveFilter(org)
		track.Response = filtered.applyRecorded(track.Response)
		filtered.apply(resp)
	}

	if t.PCB.DisableRecording {
		// the VCR is not in read-only mode so return before saving.
		return nil
	}

	// add track to cassette
	t.PCB.Logger.Printf("INFO - Cassette '%s' - Recording new track for %s %s as %s %s\n", t.Cassette.Name, req.Method, req.URL.String(), req.Method, req.URL)
	t.Cassette.addTrack(track)

	// save cassette
	return t.Cassette.save()
}
