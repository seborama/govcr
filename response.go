package govcr

import (
	"net/http"
	"regexp"
)

// ResponseFilter is a hook function that is used to filter the Response Header / Body.
//
// It works similarly to RequestFilterFunc but applies to the Response and also receives a
// copy of the Request context (if you need to pick info from it to override the response).
//
// Return the modified response.
type ResponseFilter func(resp Response) Response

// ResponseFilters is a slice of ResponseFilter
type ResponseFilters []ResponseFilter

// Response provides the response parameters.
// When returned from a ResponseFilter these values will be returned instead.
type Response struct {
	req Request

	// The content returned in the response.
	Body       []byte
	Header     http.Header
	StatusCode int
}

// Request returns the request.
// This is the request after RequestFilters have been applied.
func (r Response) Request() Request {
	// Copied to avoid modifications.
	return r.req
}

// ResponseAddHeaderValue will add/overwrite a header to the response when it is returned from vcr playback.
func ResponseAddHeaderValue(key, value string) ResponseFilter {
	return func(resp Response) Response {
		resp.Header.Add(key, value)
		return resp
	}
}

// ResponseDeleteHeaderKeys will delete one or more headers on the response when returned from vcr playback.
func ResponseDeleteHeaderKeys(keys ...string) ResponseFilter {
	return func(resp Response) Response {
		for _, key := range keys {
			resp.Header.Del(key)
		}
		return resp
	}
}

// ResponseTransferHeaderKeys will transfer one or more header from the Request to the Response.
func ResponseTransferHeaderKeys(keys ...string) ResponseFilter {
	return func(resp Response) Response {
		for _, key := range keys {
			resp.Header.Add(key, resp.req.Header.Get(key))
		}
		return resp
	}
}

// ResponseChangeBody will allows to change the body.
// Supply a function that does input to output transformation.
func ResponseChangeBody(fn func(b []byte) []byte) ResponseFilter {
	return func(resp Response) Response {
		resp.Body = fn(resp.Body)
		return resp
	}
}

// OnMethod will return a Response filter that will only apply 'r'
// if the method of the response matches.
// Original filter is unmodified.
func (r ResponseFilter) OnMethod(method string) ResponseFilter {
	return func(resp Response) Response {
		if resp.req.Method != method {
			return resp
		}
		return r(resp)
	}
}

// OnPath will return a Response filter that will only apply 'r'
// if the url string of the Response matches the supplied regex.
// Original filter is unmodified.
func (r ResponseFilter) OnPath(pathRegEx string) ResponseFilter {
	if pathRegEx == "" {
		pathRegEx = ".*"
	}
	re := regexp.MustCompile(pathRegEx)
	return func(resp Response) Response {
		if !re.MatchString(resp.req.URL.String()) {
			return resp
		}
		return r(resp)
	}
}

// OnStatus will return a Response filter that will only apply 'r'  if the response status matches.
// Original filter is unmodified.
func (r ResponseFilter) OnStatus(status int) ResponseFilter {
	return func(resp Response) Response {
		if resp.StatusCode != status {
			return resp
		}
		return r(resp)
	}
}

// Add one or more filters at the end of the filter chain.
func (r *ResponseFilters) Add(filters ...ResponseFilter) {
	v := *r
	v = append(v, filters...)
	*r = v
}

// Prepend one or more filters before the current ones.
func (r *ResponseFilters) Prepend(filters ...ResponseFilter) {
	src := *r
	dst := make(ResponseFilters, 0, len(filters)+len(src))
	dst = append(dst, filters...)
	*r = append(dst, src...)
}

// combined returns the filters as a single filter.
func (r ResponseFilters) combined() ResponseFilter {
	return func(req Response) Response {
		for _, filter := range r {
			req = filter(req)
		}
		return req
	}
}
