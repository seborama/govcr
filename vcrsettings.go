package govcr

import (
	"net/http"

	"github.com/seborama/govcr/v5/cassette"
	"github.com/seborama/govcr/v5/cassette/track"
)

// Setting defines an optional functional parameter as received by NewVCR()
type Setting func(vcrConfig *VCRSettings)

// WithLongPlay is an optional functional parameter to provide a VCR
// with the Long Play function enabled.
// LongPlay simply compresses the contents of the cassette to make
// it smaller.
// TODO this is not needed if the LongPlay mode is auto-sensed from the cassette name
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

// WithRequestMatcher is an optional functional parameter to provide a VCR with
// a RequestMatcher applied when matching an HTTP/S request to an existing track
// on a cassette.
func WithRequestMatcher(matcher RequestMatcher) Setting {
	return func(vcrConfig *VCRSettings) {
		vcrConfig.requestMatcher = matcher
	}
}

// WithTrackRecordingMutators is an optional functional parameter to provide a VCR with
// a set of track mutators applied when recording a track to a cassette.
func WithTrackRecordingMutators(trackRecordingMutators ...track.Mutator) Setting {
	return func(vcrConfig *VCRSettings) {
		vcrConfig.trackRecordingMutators = vcrConfig.trackRecordingMutators.Add(trackRecordingMutators...)
	}
}

// WithTrackReplayingMutators is an optional functional parameter to provide a VCR with
// a set of track mutators applied when replaying a track to a cassette.
// Replaying happens AFTER the request has been matched. As such, while the track's Request could be
// mutated, it will have no effect.
// However, the Request data can be referenced as part of mutating the Response.
func WithTrackReplayingMutators(trackReplayingMutators ...track.Mutator) Setting {
	return func(vcrConfig *VCRSettings) {
		vcrConfig.trackReplayingMutators = vcrConfig.trackReplayingMutators.Add(trackReplayingMutators...)
	}
}

// VCRSettings holds a set of options for the VCR.
type VCRSettings struct {
	client                 *http.Client
	cassette               *cassette.Cassette
	longPlay               bool
	requestMatcher         RequestMatcher
	trackRecordingMutators track.Mutators
	trackReplayingMutators track.Mutators
}
