package govcr

import (
	"net/http"
	"net/url"
)

type HeaderMatcher func(headers *http.Header) bool
type MethodMatcher func(method string) bool
type URLMatcher func(url *url.URL) bool
type BodyMatcher func(body string) bool
type TrailerMatcher func(trailers *http.Header) bool

func DefaultRequestMatcher(httpRequest *request, trackRequest *request) bool {
	return DefaultHeaderMatcher(httpRequest.Header, trackRequest.Header) &&
		DefaultMethodMatcher(httpRequest.Method, trackRequest.Method) &&
		DefaultURLMatcher(httpRequest.URL, trackRequest.URL) &&
		DefaultBodyMatcher(httpRequest.Body, trackRequest.Body) &&
		DefaultTrailerMatcher(httpRequest.Trailer, trackRequest.Trailer)
}

func DefaultHeaderMatcher(httpHeaders, trackHeaders http.Header) bool {
	return areHTTPHeadersEqual(httpHeaders, trackHeaders)
}

func DefaultMethodMatcher(httpMethod, trackMethod string) bool {
	return httpMethod == trackMethod
}

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

func DefaultBodyMatcher(httpBody, trackBody []byte) bool {
	return string(httpBody) == string(trackBody)
}

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
