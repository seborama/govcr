package govcr

import (
	"net/http"

	"github.com/seborama/govcr/v5/cassette"
)

// Setting defines an optional functional parameter as received by NewVCR()
type Setting func(vcrConfig *VCRSettings)

// WithLongPlay is an optional functional parameter to provide a VCR
// with the Long Play function enabled.
// LongPlay simply compresses the contents of the cassette to make
// it smaller.
// TODO this is not needed if the LongPlay mode is autosensed from the cassette name
//      ie if the name ends with '.gz'
func WithLongPlay() Setting {
	return func(vcrConfig *VCRSettings) {
		vcrConfig.longPlay = true
	}
}

// WithClient is an optional functional parameter to provide a VCR with
// a custom HTTP client.
func WithClient(httpClient *http.Client) Setting {
	return func(vcrConfig *VCRSettings) {
		vcrConfig.client = httpClient
	}
}

// WithCassette is an optional functional parameter to provide a VCR with
// a cassette to load.
func WithCassette(cassetteName string) Setting {
	return func(vcrConfig *VCRSettings) {
		k7 := cassette.LoadCassette(cassetteName)
		vcrConfig.cassette = k7
	}
}

// WithTrackRecordingMutators is an optional functional parameter to provide a VCR with
// a cassette to load.
func WithTrackRecordingMutators(trackRecordingMutators ...TrackMutator) Setting {
	return func(vcrConfig *VCRSettings) {
		vcrConfig.trackRecordingMutators = vcrConfig.trackRecordingMutators.Add(trackRecordingMutators...)
	}
}

// VCRSettings holds a set of options for the VCR.
type VCRSettings struct {
	client                 *http.Client
	cassette               *cassette.Cassette
	longPlay               bool
	trackRecordingMutators TrackMutators
}
