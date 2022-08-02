package govcr

import (
	"net/http"

	"github.com/seborama/govcr/v6/cassette/track"
	"github.com/seborama/govcr/v6/stats"
)

// ControlPanel holds the parts of a VCR that can be interacted with.
type ControlPanel struct {
	// client is the HTTP client associated with the VCR.
	client *http.Client
}

// Stats returns Stats about the cassette and VCR session.
func (controlPanel *ControlPanel) Stats() *stats.Stats {
	return controlPanel.vcrTransport().stats()
}

// LoadCassette into the VCR.
// Note: cassette.LoadCassette panics if the cassette exists but fails to load.
func (controlPanel *ControlPanel) LoadCassette(cassetteName string) error {
	return controlPanel.vcrTransport().loadCassette(cassetteName)
}

// SetRequestMatcher sets a new RequestMatcher to the VCR.
func (controlPanel *ControlPanel) SetRequestMatcher(requestMatcher RequestMatcher) {
	controlPanel.vcrTransport().SetRequestMatcher(requestMatcher)
}

// AddRecordingMutators adds a set of recording Track Mutator's to the VCR.
func (controlPanel *ControlPanel) AddRecordingMutators(trackMutators ...track.Mutator) {
	controlPanel.vcrTransport().AddRecordingMutators(trackMutators...)
}

// AddReplayingMutators adds a set of replaying Track Mutator's to the VCR.
// Replaying happens AFTER the request has been matched. As such, while the track's Request
// could be mutated, it will have no effect.
// However, the Request data can be referenced as part of mutating the Response.
func (controlPanel *ControlPanel) AddReplayingMutators(trackMutators ...track.Mutator) {
	controlPanel.vcrTransport().AddReplayingMutators(trackMutators...)
}

// HTTPClient returns the http.Client that contains the VCR.
func (controlPanel *ControlPanel) HTTPClient() *http.Client {
	return controlPanel.client
}

// EjectCassette from the VCR.
func (controlPanel *ControlPanel) EjectCassette() {
	controlPanel.vcrTransport().ejectCassette()
}

// NumberOfTracks returns the number of tracks contained in the cassette.
func (controlPanel *ControlPanel) NumberOfTracks() int32 {
	return controlPanel.vcrTransport().NumberOfTracks()
}

func (controlPanel *ControlPanel) vcrTransport() *vcrTransport {
	return controlPanel.client.Transport.(*vcrTransport)
}
