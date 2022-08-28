package govcr

import (
	"net/http"

	"github.com/seborama/govcr/v13/cassette"
	"github.com/seborama/govcr/v13/cassette/track"
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

// WithRequestMatchers is an optional functional parameter to provide a VCR with a
// set of RequestMatcher's applied when matching an HTTP/S request to an existing
// track on a cassette.
func WithRequestMatchers(reqMatchers ...RequestMatcher) Setting {
	return func(vcrSettings *VCRSettings) {
		vcrSettings.requestMatchers = vcrSettings.requestMatchers.Add(reqMatchers...)
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

// WithLiveOnlyMode sets the VCR to make live calls only, do not replay from cassette even
// if a track would exist.
// Perhaps more useful when used in combination with 'readOnly' to by-pass govcr entirely.
func WithLiveOnlyMode() Setting {
	return func(vcrSettings *VCRSettings) {
		vcrSettings.httpMode = HTTPModeLiveOnly
	}
}

// WithReadOnlyMode sets the VCR to replay tracks from cassette, if present, or make live
// calls but do not records new tracks.
func WithReadOnlyMode() Setting {
	return func(vcrSettings *VCRSettings) {
		vcrSettings.readOnly = true
	}
}

// WithOfflineMode sets the VCR to replay tracks from cassette, if present, but do not make
// live calls.
// govcr will return a transport error if no track was found.
func WithOfflineMode() Setting {
	return func(vcrSettings *VCRSettings) {
		vcrSettings.httpMode = HTTPModeOffline
	}
}

// VCRSettings holds a set of options for the VCR.
type VCRSettings struct {
	client                 *http.Client
	cassette               *cassette.Cassette
	requestMatchers        RequestMatchers
	trackRecordingMutators track.Mutators
	trackReplayingMutators track.Mutators
	httpMode               HTTPMode
	readOnly               bool
}
