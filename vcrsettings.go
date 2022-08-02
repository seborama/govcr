package govcr

import (
	"net/http"

	"github.com/seborama/govcr/v6/cassette"
	"github.com/seborama/govcr/v6/cassette/track"
)

// Setting defines an optional functional parameter as received by NewVCR().
type Setting func(vcrSettings *VCRSettings)

// WithClient is an optional functional parameter to provide a VCR with
// a custom HTTP client.
func WithClient(httpClient *http.Client) Setting {
	return func(vcrSettings *VCRSettings) {
		vcrSettings.client = httpClient
	}
}

// WithCassette is an optional functional parameter to provide a VCR with
// a cassette to load.
func WithCassette(cassetteName string) Setting {
	return func(vcrSettings *VCRSettings) {
		k7 := cassette.LoadCassette(cassetteName)
		vcrSettings.cassette = k7
	}
}

// WithRequestMatcher is an optional functional parameter to provide a VCR with
// a RequestMatcher applied when matching an HTTP/S request to an existing track
// on a cassette.
func WithRequestMatcher(matcher RequestMatcher) Setting {
	return func(vcrSettings *VCRSettings) {
		vcrSettings.requestMatcher = matcher
	}
}

// WithTrackRecordingMutators is an optional functional parameter to provide a VCR with
// a set of track mutators applied when recording a track to a cassette.
func WithTrackRecordingMutators(trackRecordingMutators ...track.Mutator) Setting {
	return func(vcrSettings *VCRSettings) {
		vcrSettings.trackRecordingMutators = vcrSettings.trackRecordingMutators.Add(trackRecordingMutators...)
	}
}

// WithTrackReplayingMutators is an optional functional parameter to provide a VCR with
// a set of track mutators applied when replaying a track to a cassette.
// Replaying happens AFTER the request has been matched. As such, while the track's Request could be
// mutated, it will have no effect.
// However, the Request data can be referenced as part of mutating the Response.
func WithTrackReplayingMutators(trackReplayingMutators ...track.Mutator) Setting {
	return func(vcrSettings *VCRSettings) {
		vcrSettings.trackReplayingMutators = vcrSettings.trackReplayingMutators.Add(trackReplayingMutators...)
	}
}

// VCRSettings holds a set of options for the VCR.
type VCRSettings struct {
	client                 *http.Client
	cassette               *cassette.Cassette
	requestMatcher         RequestMatcher
	trackRecordingMutators track.Mutators
	trackReplayingMutators track.Mutators
}
