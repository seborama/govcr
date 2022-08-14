package govcr

import (
	"net/http"
	"net/url"

	"github.com/seborama/govcr/v8/cassette/track"
)

// RequestMatcherFunc is a function that performs request comparison.
type RequestMatcherFunc func(httpRequest, trackRequest *track.Request) bool

// HeaderMatcher is a function that performs header comparison.
type HeaderMatcher func(httpHeaders, trackHeaders http.Header) bool

// MethodMatcher is a function that performs method name comparison.
type MethodMatcher func(httpMethod, trackMethod string) bool

// URLMatcher is a function that performs URL comparison.
type URLMatcher func(httpURL, trackURL *url.URL) bool

// BodyMatcher is a function that performs body comparison.
type BodyMatcher func(httpBody, trackBody []byte) bool

// TrailerMatcher is a function that performs trailer comparison.
type TrailerMatcher func(httpTrailers, trackTrailers http.Header) bool

// DefaultRequestMatcher is a default implementation of RequestMatcher.
type DefaultRequestMatcher struct {
	matchers []RequestMatcherFunc
}

// Match is the default implementation of RequestMatcher.
func (rm *DefaultRequestMatcher) Match(httpRequest, trackRequest *track.Request) bool {
	for _, matcher := range rm.matchers {
		if !matcher(httpRequest, trackRequest) {
			return false
		}
	}

	return true
}

// DefaultRequestMatcherOptions defines a signature to provide options
// when creating a new DefaultRequestMatcher.
type DefaultRequestMatcherOptions func(*DefaultRequestMatcher)

// WithRequestMatcherFunc is an option that can be used when creating
// a new DefaultRequestMatcherOptions to add a RequestMatcherFunc to it.
func WithRequestMatcherFunc(m RequestMatcherFunc) DefaultRequestMatcherOptions {
	return func(rm *DefaultRequestMatcher) {
		rm.matchers = append(rm.matchers, m)
	}
}

// NewBlankRequestMatcher creates a new default implementation of RequestMatcher.
// By default, it will always match any and all requests to a cassette track.
// You should pass specific RequestMatcherFunc as options to customise its behaviour.
// You can also use one of the predefined matchers such as those provided by
// NewStrictRequestMatcher() or NewMethodURLRequestMatcher().
func NewBlankRequestMatcher(options ...DefaultRequestMatcherOptions) *DefaultRequestMatcher {
	drm := DefaultRequestMatcher{}

	for _, option := range options {
		option(&drm)
	}

	return &drm
}

// NewStrictRequestMatcher creates a new default implementation of RequestMatcher.
func NewStrictRequestMatcher() *DefaultRequestMatcher {
	drm := DefaultRequestMatcher{
		matchers: []RequestMatcherFunc{
			DefaultHeaderMatcher,
			DefaultMethodMatcher,
			DefaultURLMatcher,
			DefaultBodyMatcher,
			DefaultTrailerMatcher,
		},
	}

	return &drm
}

// NewMethodURLRequestMatcher creates a new implementation of RequestMatcher based on Method and URL.
func NewMethodURLRequestMatcher() *DefaultRequestMatcher {
	drm := DefaultRequestMatcher{
		matchers: []RequestMatcherFunc{
			DefaultMethodMatcher,
			DefaultURLMatcher,
		},
	}

	return &drm
}

// DefaultHeaderMatcher is the default implementation of HeaderMatcher.
// Because this function is meant to be called from RequestMatcher.Match(),
// it doesn't check for either argument to be nil. Match() takes care of it.
func DefaultHeaderMatcher(httpRequest, trackRequest *track.Request) bool {
	return areHTTPHeadersEqual(httpRequest.Header, trackRequest.Header)
}

// DefaultMethodMatcher is the default implementation of MethodMatcher.
// Because this function is meant to be called from DefaultRequestMatcher.Match(),
// it doesn't check for either argument to be nil. Match() takes care of it.
func DefaultMethodMatcher(httpRequest, trackRequest *track.Request) bool {
	return httpRequest.Method == trackRequest.Method
}

// DefaultURLMatcher is the default implementation of URLMatcher.
// Because this function is meant to be called from DefaultRequestMatcher.Match(),
// it doesn't check for either argument to be nil. Match() takes care of it.
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
// Because this function is meant to be called from DefaultRequestMatcher.Match(),
// it doesn't check for either argument to be nil. Match() takes care of it.
func DefaultBodyMatcher(httpRequest, trackRequest *track.Request) bool {
	return string(httpRequest.Body) == string(trackRequest.Body)
}

// DefaultTrailerMatcher is the default implementation of TrailerMatcher.
// Because this function is meant to be called from DefaultRequestMatcher.Match(),
// it doesn't check for either argument to be nil. Match() takes care of it.
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
