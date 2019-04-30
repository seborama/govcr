package govcr

import (
	"net/http"
	"net/url"
)

// RequestMatcher is a function that performs request comparison.
type RequestMatcher func(httpRequest *request, trackRequest *request) bool

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

// DefaultRequestMatcher is the default implementation of RequestMatcher.
func DefaultRequestMatcher(httpRequest *request, trackRequest *request) bool {
	return DefaultHeaderMatcher(httpRequest.Header, trackRequest.Header) &&
		DefaultMethodMatcher(httpRequest.Method, trackRequest.Method) &&
		DefaultURLMatcher(httpRequest.URL, trackRequest.URL) &&
		DefaultBodyMatcher(httpRequest.Body, trackRequest.Body) &&
		DefaultTrailerMatcher(httpRequest.Trailer, trackRequest.Trailer)
}

// DefaultHeaderMatcher is the default implementation of HeaderMatcher.
func DefaultHeaderMatcher(httpHeaders, trackHeaders http.Header) bool {
	return areHTTPHeadersEqual(httpHeaders, trackHeaders)
}

// DefaultMethodMatcher is the default implementation of MethodMatcher.
func DefaultMethodMatcher(httpMethod, trackMethod string) bool {
	return httpMethod == trackMethod
}

// DefaultURLMatcher is the default implementation of URLMatcher.
func DefaultURLMatcher(httpURL, trackURL *url.URL) bool {
	if (httpURL == nil && trackURL != nil) ||
		(httpURL != nil && trackURL == nil) {
		return false
	} else if httpURL == nil {
		return true
	}

	if (httpURL.User == nil && trackURL.User != nil) ||
		(httpURL.User != nil && trackURL.User == nil) {
		return false
	} else if httpURL.User != nil &&
		httpURL.User.String() != trackURL.User.String() {
		return false
	}

	return httpURL.Scheme == trackURL.Scheme &&
		httpURL.Opaque == trackURL.Opaque &&
		httpURL.Host == trackURL.Host &&
		httpURL.Path == trackURL.Path &&
		httpURL.RawPath == trackURL.RawPath &&
		httpURL.ForceQuery == trackURL.ForceQuery &&
		httpURL.RawQuery == trackURL.RawQuery &&
		httpURL.Fragment == trackURL.Fragment
}

// DefaultBodyMatcher is the default implementation of BodyMatcher.
func DefaultBodyMatcher(httpBody, trackBody []byte) bool {
	return string(httpBody) == string(trackBody)
}

// DefaultTrailerMatcher is the default implementation of TrailerMatcher.
func DefaultTrailerMatcher(httpTrailers, trackTrailers http.Header) bool {
	return areHTTPHeadersEqual(httpTrailers, trackTrailers)
}

func areHTTPHeadersEqual(httpHeaders1, httpHeaders2 http.Header) bool {
	if (httpHeaders1 == nil && httpHeaders2 != nil) ||
		(httpHeaders1 != nil && httpHeaders2 == nil) {
		return false
	} else if httpHeaders1 == nil {
		return true
	}

	if len(httpHeaders1) != len(httpHeaders2) {
		return false
	}

	for httpHeaderKey, httpHeaderValues := range httpHeaders1 {
		trackHeaderValues, ok := httpHeaders2[httpHeaderKey]
		if !ok {
			return false
		}
		if len(httpHeaderValues) != len(trackHeaderValues) {
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
