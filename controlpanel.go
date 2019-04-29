package govcr

import "net/http"

// ControlPanel holds the parts of a VCR that can be interacted with.
type ControlPanel struct {
	// client is the HTTP client associated with the VCR.
	client *http.Client
}

// Stats returns Stats about the cassette and VCR session.
func (controlPanel *ControlPanel) Stats() *Stats {
	return controlPanel.vcrTransport().stats()
}

// LoadCassette into the VCR.
func (controlPanel *ControlPanel) LoadCassette(cassetteName string) error {
	return controlPanel.vcrTransport().loadCassette(cassetteName)
}

// Player returns the http.Client that contains the VCR.
func (controlPanel *ControlPanel) Player() *http.Client {
	return controlPanel.client
}

// EjectCassette from the VCR.
func (controlPanel *ControlPanel) EjectCassette() {
	controlPanel.vcrTransport().ejectCassette()
}

func (controlPanel *ControlPanel) vcrTransport() *vcrTransport {
	return controlPanel.client.Transport.(*vcrTransport)
}

// NumberOfTracks returns the number of tracks contained in the cassette.
func (controlPanel *ControlPanel) NumberOfTracks() int32 {
	return controlPanel.vcrTransport().NumberOfTracks()
}
