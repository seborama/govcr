package govcr

import (
	"net/http"
	"net/url"

	"github.com/seborama/govcr/v13/cassette/track"
)

// RequestMatcher is a function that performs request comparison.
// request comparison involves the HTTP request and the track request recorded on cassette.
type RequestMatcher func(httpRequest, trackRequest *track.Request) bool

// RequestMatchers is a collection of RequestMatcher's.
type RequestMatchers []RequestMatcher

// Add a set of RequestMatcher's to this RequestMatchers collection.
func (rm RequestMatchers) Add(reqMatchers ...RequestMatcher) RequestMatchers {
	return append(rm, reqMatchers...)
}

// Match returns true if all RequestMatcher's in RequestMatchers return true, thereby indicating that
// the trackRequest matches the httpRequest.
// When no matchers are supplied, Match returns false.
func (rm RequestMatchers) Match(httpRequest, trackRequest *track.Request) bool {
	for _, matcher := range rm {
		if !matcher(httpRequest, trackRequest) {
			return false
		}
	}

	return len(rm) != 0
}

// NewStrictRequestMatchers creates a new default sets of RequestMatcher's.
func NewStrictRequestMatchers() RequestMatchers {
	return RequestMatchers{
		DefaultHeaderMatcher,
		DefaultMethodMatcher,
		DefaultURLMatcher,
		DefaultBodyMatcher,
		DefaultTrailerMatcher,
	}
}

// NewMethodURLRequestMatchers creates a new default set of RequestMatcher's based on Method and URL.
func NewMethodURLRequestMatchers() RequestMatchers {
	return RequestMatchers{
		DefaultMethodMatcher,
		DefaultURLMatcher,
	}
}

// DefaultHeaderMatcher is the default implementation of HeaderMatcher.
func DefaultHeaderMatcher(httpRequest, trackRequest *track.Request) bool {
	return areHTTPHeadersEqual(httpRequest.Header, trackRequest.Header)
}

// DefaultMethodMatcher is the default implementation of MethodMatcher.
func DefaultMethodMatcher(httpRequest, trackRequest *track.Request) bool {
	return httpRequest.Method == trackRequest.Method
}

// DefaultURLMatcher is the default implementation of URLMatcher.
// nolint: gocyclo,gocognit
func DefaultURLMatcher(httpRequest, trackRequest *track.Request) bool {
	httpURL := httpRequest.URL
	if httpURL == nil {
		httpURL = &url.URL{}
	}

	trackURL := trackRequest.URL
	if trackURL == nil {
		trackURL = &url.URL{}
	}

	return httpURL.Scheme == trackURL.Scheme &&
		httpURL.Opaque == trackURL.Opaque &&
		httpURL.User.String() == trackURL.User.String() &&
		httpURL.Host == trackURL.Host &&
		httpURL.Path == trackURL.Path &&
		httpURL.RawPath == trackURL.RawPath &&
		httpURL.ForceQuery == trackURL.ForceQuery &&
		httpURL.RawQuery == trackURL.RawQuery &&
		httpURL.Fragment == trackURL.Fragment
}

// DefaultBodyMatcher is the default implementation of BodyMatcher.
func DefaultBodyMatcher(httpRequest, trackRequest *track.Request) bool {
	return string(httpRequest.Body) == string(trackRequest.Body)
}

// DefaultTrailerMatcher is the default implementation of TrailerMatcher.
func DefaultTrailerMatcher(httpRequest, trackRequest *track.Request) bool {
	return areHTTPHeadersEqual(httpRequest.Trailer, trackRequest.Trailer)
}

// nolint: gocyclo,gocognit
func areHTTPHeadersEqual(httpHeaders1, httpHeaders2 http.Header) bool {
	if len(httpHeaders1) != len(httpHeaders2) {
		return false
	}

	for httpHeaderKey, httpHeaderValues := range httpHeaders1 {
		trackHeaderValues, ok := httpHeaders2[httpHeaderKey]
		if !ok || len(httpHeaderValues) != len(trackHeaderValues) {
			return false
		}

		// "postal" sorting algo
		m := make(map[string]int)

		for _, httpHeaderValue := range httpHeaderValues {
			m[httpHeaderValue]++ // put mail in inbox
		}

		for _, trackHeaderValue := range trackHeaderValues {
			m[trackHeaderValue]-- // pop mail from inbox
		}

		for _, count := range m {
			if count != 0 {
				return false
			}
		}
	}

	return true
}
