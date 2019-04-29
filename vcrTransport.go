package govcr

import (
	"log"
	"net/http"

	"github.com/pkg/errors"
)

// vcrTransport is the heart of VCR. It implements
// http.RoundTripper that wraps over the default
// one provided by Go's http package or a custom one
// if provided when calling NewVCR.
type vcrTransport struct {
	pcb       *pcb
	cassette  *cassette
	transport http.RoundTripper
}

func (t *vcrTransport) loadCassette(cassetteName string) error {
	if t.cassette != nil {
		return errors.Errorf("failed to load cassette '%s': another cassette ('%s') is already loaded", cassetteName, t.cassette.name)
	}

	k7, err := loadCassette(cassetteName)
	if err != nil {
		return errors.Wrapf(err, "failed to load contents of cassette '%s'", cassetteName)
	}

	t.cassette = k7

	return nil
}

// RoundTrip is an implementation of http.RoundTripper.
func (t *vcrTransport) RoundTrip(httpRequest *http.Request) (*http.Response, error) {
	// Note: by convention resp should be nil if an error occurs with HTTP
	var httpResponse *http.Response

	httpRequestClone := cloneHTTPRequest(httpRequest)
	if response, err := t.pcb.seekTrack(t.cassette, httpRequestClone); response != nil {
		// TODO: add a test to confirm err is replaying errors correctly (see cassette_test.go / Test_trackReplaysError)
		return response, err
	}

	httpResponse, reqErr := t.transport.RoundTrip(httpRequest)
	response := fromHTTPResponse(httpResponse)

	request := fromHTTPRequest(httpRequestClone)

	if err := recordNewTrackToCassette(t.cassette, request, response, reqErr); err != nil {
		log.Println(err)
	}

	return httpResponse, reqErr
}

func (t *vcrTransport) ejectCassette() {
	t.cassette = nil
}

func (t *vcrTransport) stats() *Stats {
	return t.cassette.Stats()
}

func (t *vcrTransport) NumberOfTracks() int32 {
	return t.cassette.NumberOfTracks()
}
