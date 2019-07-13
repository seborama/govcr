package govcr

import (
	"net/http"
	"net/url"
)

// RequestMatcherFunc is a function that performs request comparison.
type RequestMatcherFunc func(httpRequest *Request, trackRequest *Request) bool

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

// DefaultRequestMatcherOptions defines a signature to provide options
// when creating a new DefaultRequestMatcher.
type DefaultRequestMatcherOptions func(*DefaultRequestMatcher)

// WithRequestMatcher is an option that can be used when creating a new
// DefaultRequestMatcherOptions to add a request matcher to it.
func WithRequestMatcher(m RequestMatcherFunc) DefaultRequestMatcherOptions {
	return func(rm *DefaultRequestMatcher) {
		rm.matchers = append(rm.matchers, m)
	}
}

// NewDefaultRequestMatcher creates a new default implementation of RequestMatcher.
func NewDefaultRequestMatcher(options ...DefaultRequestMatcherOptions) RequestMatcher {
	drm := DefaultRequestMatcher{
		matchers: []RequestMatcherFunc{
			DefaultHeaderMatcher,
			DefaultMethodMatcher,
			DefaultURLMatcher,
			DefaultBodyMatcher,
			DefaultTrailerMatcher,
		},
	}

	for _, option := range options {
		option(&drm)
	}

	return &drm
}

// Match is the default implementation of RequestMatcher.
func (rm *DefaultRequestMatcher) Match(httpRequest *Request, trackRequest *Request) bool {
	for _, matcher := range rm.matchers {
		if !matcher(httpRequest, trackRequest) {
			return false
		}
	}
	return true
}

// DefaultHeaderMatcher is the default implementation of HeaderMatcher.
// Because this function is meant to be called from RequestMatcher.Match(),
// it doesn't check for either argument to be nil. Match() takes care of it.
func DefaultHeaderMatcher(httpRequest *Request, trackRequest *Request) bool {
	return areHTTPHeadersEqual(httpRequest.Header, trackRequest.Header)
}

// DefaultMethodMatcher is the default implementation of MethodMatcher.
// Because this function is meant to be called from DefaultRequestMatcher.Match(),
// it doesn't check for either argument to be nil. Match() takes care of it.
func DefaultMethodMatcher(httpRequest *Request, trackRequest *Request) bool {
	return httpRequest.Method == trackRequest.Method
}

// DefaultURLMatcher is the default implementation of URLMatcher.
// Because this function is meant to be called from DefaultRequestMatcher.Match(),
// it doesn't check for either argument to be nil. Match() takes care of it.
func DefaultURLMatcher(httpRequest *Request, trackRequest *Request) bool {
	httpURL := httpRequest.URL
	trackURL := trackRequest.URL
	if httpURL == nil {
		httpURL = &url.URL{}
	}
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
func DefaultBodyMatcher(httpRequest *Request, trackRequest *Request) bool {
	return string(httpRequest.Body) == string(trackRequest.Body)
}

// DefaultTrailerMatcher is the default implementation of TrailerMatcher.
// Because this function is meant to be called from DefaultRequestMatcher.Match(),
// it doesn't check for either argument to be nil. Match() takes care of it.
func DefaultTrailerMatcher(httpRequest *Request, trackRequest *Request) bool {
	return areHTTPHeadersEqual(httpRequest.Trailer, trackRequest.Trailer)
}

func areHTTPHeadersEqual(httpHeaders1, httpHeaders2 http.Header) bool {
	if len(httpHeaders1) != len(httpHeaders2) {
		return false
	}

	for httpHeaderKey, httpHeaderValues := range httpHeaders1 {
		trackHeaderValues, ok := httpHeaders2[httpHeaderKey]
		if !ok || len(httpHeaderValues) != len(trackHeaderValues) {
			return false
		}

		m := make(map[string]int)
		for _, httpHeaderValue := range httpHeaderValues {
			m[httpHeaderValue]++
		}
		for _, trackHeaderValue := range trackHeaderValues {
			m[trackHeaderValue]--
		}
		for _, count := range m {
			if count != 0 {
				return false
			}
		}
	}

	return true
}