package govcr

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// VCRControlPanel holds the parts of a VCR that can be interacted with.
// Client is the HTTP client associated with the VCR.
type VCRControlPanel struct {
	Client *http.Client
}

// Stats returns Stats about the cassette and VCR session.
func (vcr *VCRControlPanel) Stats() Stats {
	vcrT := vcr.Client.Transport.(*vcrTransport)
	return vcrT.Cassette.Stats()
}

const defaultCassettePath = "./govcr-fixtures/"

// VCRConfig holds a set of options for the VCR.
type VCRConfig struct {
	Client            *http.Client
	ExcludeHeaderFunc ExcludeHeaderFunc
	RequestFilterFunc RequestFilterFunc

	// ResponseFilterFunc can be used to modify the header of the response.
	// This is useful when a fingerprint is exchanged and expected to match between request and response.
	ResponseFilterFunc ResponseFilterFunc

	// Filter to run before request is matched against cassettes.
	RequestFilter RequestFilter

	// Filter to run before a response is returned.
	ResponseFilter ResponseFilter

	DisableRecording bool
	Logging          bool
	CassettePath     string
}

// PCB stands for Printed Circuit Board. It is a structure that holds some
// facilities that are passed to the VCR machine to modify its internals.
type pcb struct {
	Transport         http.RoundTripper
	ExcludeHeaderFunc ExcludeHeaderFunc
	RequestFilter     RequestFilter
	ResponseFilter    ResponseFilter
	//RequestFilterFunc  RequestFilterFunc
	//ResponseFilterFunc ResponseFilterFunc
	Logger           *log.Logger
	DisableRecording bool
	CassettePath     string
}

const trackNotFound = -1

func (pcbr *pcb) seekTrack(cassette *cassette, req Request) int {
	for idx := range cassette.Tracks {
		if pcbr.trackMatches(cassette, idx, req) {
			pcbr.Logger.Printf("INFO - Cassette '%s' - Found a matching track for %s %s\n", cassette.Name, req.Method, req.URL.String())
			return idx
		}
	}

	return trackNotFound
}

// Matches checks whether the track is a match for the supplied request.
func (pcbr *pcb) trackMatches(cassette *cassette, trackNumber int, req Request) bool {
	track := cassette.Tracks[trackNumber]

	// apply filter function to track header / body
	filteredTrackRequest := pcbr.RequestFilter(track.Request.Request())

	// apply filter function to request header / body
	filteredReq := pcbr.RequestFilter(req)

	return !track.replayed &&
		track.Request.Method == req.Method &&
		track.Request.URL.String() == req.URL.String() &&
		pcbr.headerResembles(filteredTrackRequest.Header, filteredReq.Header) &&
		pcbr.bodyResembles(filteredTrackRequest.Body, filteredReq.Body)
}

// headerResembles compares HTTP headers for equivalence.
func (pcbr *pcb) headerResembles(header1 http.Header, header2 http.Header) bool {
	for k := range header1 {
		// TODO: a given header may have several values (and in any order)
		if GetFirstValue(header1, k) != GetFirstValue(header2, k) && !pcbr.ExcludeHeaderFunc(k) {
			return false
		}
	}

	// finally assert the number of headers match
	// TODO: perhaps should count how many pcb.ExcludeHeaderFunc() returned true and remove that count from the len to compare?
	return len(header1) == len(header2)
}

// bodyResembles compares HTTP bodies for equivalence.
func (pcbr *pcb) bodyResembles(body1 []byte, body2 []byte) bool {
	return bytes.Equal(body1, body2)
}

func (pcbr *pcb) filterResponse(resp *http.Response, req Request) *http.Response {
	body, err := readResponseBody(resp)
	if err != nil {
		pcbr.Logger.Printf("ERROR - Unable to filter response body so leaving it untouched: %s\n", err.Error())
		return resp
	}

	filtResp := Response{
		req:        req,
		Body:       body,
		Header:     resp.Header,
		StatusCode: resp.StatusCode,
	}
	filtResp = pcbr.ResponseFilter(filtResp)
	resp.Header = filtResp.Header
	resp.Body = toReadCloser(filtResp.Body)
	resp.StatusCode = filtResp.StatusCode

	return resp
}

// GetFirstValue is a utility function that extracts the first value of a header key.
// The reason for this function is that some servers require case sensitive headers which
// prevent the use of http.Header.Get() as it expects header keys to be canonicalized.
func GetFirstValue(hdr http.Header, key string) string {
	for k, val := range hdr {
		if strings.ToLower(k) == strings.ToLower(key) {
			if len(val) > 0 {
				return val[0]
			}
			return ""
		}
	}

	return ""
}

// NewVCR creates a new VCR and loads a cassette.
// A RoundTripper can be provided when a custom Transport is needed (for example to provide
// certificates, etc)
func NewVCR(cassetteName string, vcrConfig *VCRConfig) *VCRControlPanel {
	if vcrConfig == nil {
		vcrConfig = &VCRConfig{}
	}

	// set up logging
	logger := log.New(os.Stderr, "", log.LstdFlags)
	if !vcrConfig.Logging {
		out, _ := os.OpenFile(os.DevNull, os.O_WRONLY|os.O_APPEND, 0600)
		logger.SetOutput(out)
	}

	// use a default client if none provided
	if vcrConfig.Client == nil {
		vcrConfig.Client = http.DefaultClient
	}

	// use a default transport if none provided
	if vcrConfig.Client.Transport == nil {
		vcrConfig.Client.Transport = http.DefaultTransport
	}

	// use a default set of FilterFunc's
	if vcrConfig.ExcludeHeaderFunc == nil {
		vcrConfig.ExcludeHeaderFunc = func(key string) bool {
			return false
		}
	}

	if vcrConfig.RequestFilterFunc == nil {
		vcrConfig.RequestFilterFunc = func(header http.Header, body []byte) (*http.Header, *[]byte) {
			return &header, &body
		}
	}

	if vcrConfig.ResponseFilterFunc == nil {
		vcrConfig.ResponseFilterFunc = func(respHdr http.Header, body []byte, reqHdr http.Header) (*http.Header, *[]byte) {
			return &respHdr, &body
		}
	}

	if vcrConfig.RequestFilter == nil {
		vcrConfig.RequestFilter = func(req Request) Request {
			return req
		}
	}

	if vcrConfig.ResponseFilter == nil {
		vcrConfig.ResponseFilter = func(req Response) Response {
			return req
		}
	}

	// load cassette
	cassette, err := loadCassette(cassetteName, vcrConfig.CassettePath)
	if err != nil {
		logger.Fatal(err)
	}

	// create PCB
	pcbr := &pcb{
		// TODO: create appropriate test!
		DisableRecording:  vcrConfig.DisableRecording,
		Transport:         vcrConfig.Client.Transport,
		ExcludeHeaderFunc: vcrConfig.ExcludeHeaderFunc,
		RequestFilter:     vcrConfig.RequestFilterFunc.RequestFilter().Chain(vcrConfig.RequestFilter),
		ResponseFilter:    vcrConfig.ResponseFilterFunc.ResponseFilter().Chain(vcrConfig.ResponseFilter),
		Logger:            logger,
		CassettePath:      vcrConfig.CassettePath,
	}

	// create VCR's HTTP client
	vcrClient := &http.Client{
		Transport: &vcrTransport{
			PCB:      pcbr,
			Cassette: cassette,
		},
	}

	// copy the attributes of the original http.Client
	vcrClient.CheckRedirect = vcrConfig.Client.CheckRedirect
	vcrClient.Jar = vcrConfig.Client.Jar
	vcrClient.Timeout = vcrConfig.Client.Timeout

	// return
	return &VCRControlPanel{
		Client: vcrClient,
	}
}

// ExcludeHeaderFunc is a hook function that is used to filter the Header.
//
// Typically this can be used to remove / amend undesirable custom headers from the request.
//
// For instance, if your application sends requests with a timestamp held in a custom header,
// you likely want to exclude it from the comparison to ensure that the request headers are
// considered a match with those saved on the cassette's track.
//
// Parameters:
//  - parameter 1 - Name of header key in the Request
//
// Return value:
// true - exclude header key from comparison
// false - retain header key for comparison
type ExcludeHeaderFunc func(key string) bool

// RequestFilterFunc is a hook function that is used to filter the Request Header / Body.
//
// Typically this can be used to remove / amend undesirable header / body elements from the request.
//
// For instance, if your application sends requests with a timestamp held in a part of
// the header / body, you likely want to remove it or force a static timestamp via
// RequestFilterFunc to ensure that the request body matches those saved on the cassette's track.
//
// It is important to note that this differs from ExcludeHeaderFunc in that the former does not
// modify the header (it only returns a bool) whereas this function can be used to modify the header.
//
// Parameters:
//  - parameter 1 - Copy of http.Header of the Request
//  - parameter 2 - Copy of string of the Request's Body
//
// Return values:
//  - value 1 - Request's amended header
//  - value 2 - Request's amended body
// Deprecated: Use RequestFilter instead.
type RequestFilterFunc func(http.Header, []byte) (*http.Header, *[]byte)

// RequestFilter returns the RequestFilterFunc as a RequestFilter.
func (r RequestFilterFunc) RequestFilter() RequestFilter {
	return func(req Request) Request {
		header, body := r(req.Header, req.Body)
		if header != nil {
			req.Header = *header
		}
		if body != nil {
			req.Body = *body
		}
		return req
	}
}

// Typically this can be used to remove / amend undesirable header / body elements from the request.
//
// For instance, if your application sends requests with a timestamp held in a part of
// the header / body, you likely want to remove it or force a static timestamp via
// RequestFilterFunc to ensure that the request body matches those saved on the cassette's track.
//
// It is important to note that this differs from ExcludeHeaderFunc in that the former does not
// modify the header (it only returns a bool) whereas this function can be used to modify the header.
//
// Return the request with any modified values.
type RequestFilter func(req Request) Request

// Request provides the request parameters.
// The returned the amended values.
type Request struct {
	Header http.Header
	Body   []byte
	Method string
	URL    url.URL
}

// WithMethod will return a new filter that will only apply 'r'
// if the method of the request matches.
// Original filter is unmodified.
func (r RequestFilter) WithMethod(method string) RequestFilter {
	return func(req Request) Request {
		if req.Method != method {
			return req
		}
		return r(req)
	}
}

// WithPath will return a request filter that will only apply 'r'
// if the url string of the request matches the supplied regex.
// Original filter is unmodified.
func (r RequestFilter) WithPath(pathRegEx string) RequestFilter {
	if pathRegEx == "" {
		pathRegEx = "*"
	}
	re := regexp.MustCompile(pathRegEx)
	return func(req Request) Request {
		if !re.MatchString(req.URL.String()) {
			return req
		}
		return r(req)
	}
}

// AddHeaderValue will add a header to the request.
func (r RequestFilter) AddHeaderValue(key, value string) RequestFilter {
	return func(req Request) Request {
		req = r(req)
		req.Header.Add(key, value)
		return req
	}
}

// DeleteHeaderKeys will delete one or more header keys on the request.
func (r RequestFilter) DeleteHeaderKeys(keys ...string) RequestFilter {
	return func(req Request) Request {
		req = r(req)
		for _, key := range keys {
			req.Header.Del(key)
		}
		return req
	}
}

// Chain one or more filters after the current one and return as single filter.
func (r RequestFilter) Chain(filters ...RequestFilter) RequestFilter {
	return func(req Request) Request {
		req = r(req)
		for _, fn := range filters {
			if fn == nil {
				continue
			}
			req = fn(req)
		}
		return req
	}
}

// ResponseFilter is a hook function that is used to filter the Response Header / Body.
//
// It works similarly to RequestFilterFunc but applies to the Response and also receives a
// copy of the Request context (if you need to pick info from it to override the response).
//
// Return the modified response.
type ResponseFilter func(resp Response) Response

// ResponseContext provides the response parameters.
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

// WithMethod will return a Response filter that will only apply 'r'
// if the method of the response matches.
// Original filter is unmodified.
func (r ResponseFilter) WithMethod(method string) ResponseFilter {
	return func(resp Response) Response {
		if resp.req.Method != method {
			return resp
		}
		return r(resp)
	}
}

// WithPath will return a Response filter that will only apply 'r'
// if the url string of the Response matches the supplied regex.
// Original filter is unmodified.
func (r ResponseFilter) WithPath(pathRegEx string) ResponseFilter {
	if pathRegEx == "" {
		pathRegEx = "*"
	}
	re := regexp.MustCompile(pathRegEx)
	return func(resp Response) Response {
		if !re.MatchString(resp.req.URL.String()) {
			return resp
		}
		return r(resp)
	}
}

// WithStatus will return a Response filter that will only apply 'r'  if the response status matches.
// Original filter is unmodified.
func (r ResponseFilter) WithStatus(status int) ResponseFilter {
	return func(resp Response) Response {
		if resp.StatusCode != status {
			return resp
		}
		return r(resp)
	}
}

// AddHeaderValue will add a header to the response.
func (r ResponseFilter) AddHeaderValue(key, value string) ResponseFilter {
	return func(resp Response) Response {
		resp = r(resp)
		resp.Header.Add(key, value)
		return resp
	}
}

// DeleteHeader will delete a header on the response.
func (r ResponseFilter) DeleteHeaderKeys(keys ...string) ResponseFilter {
	return func(resp Response) Response {
		resp = r(resp)
		for _, key := range keys {
			resp.Header.Del(key)
		}
		return resp
	}
}

// ChangeBody will allows to change the body.
func (r ResponseFilter) ChangeBody(fn func(b []byte) []byte) ResponseFilter {
	return func(resp Response) Response {
		resp = r(resp)
		resp.Body = fn(resp.Body)
		return resp
	}
}

// Chain one or more filters after the current one and return as single filter.
func (r ResponseFilter) Chain(filters ...ResponseFilter) ResponseFilter {
	return func(resp Response) Response {
		resp = r(resp)
		for _, fn := range filters {
			if fn == nil {
				continue
			}
			resp = fn(resp)
		}
		return resp
	}
}

// ResponseFilterFunc is a hook function that is used to filter the Response Header / Body.
//
// It works similarly to RequestFilterFunc but applies to the Response and also receives a
// copy of the Request's header (if you need to pick info from it to override the response).
//
// Parameters:
//  - parameter 1 - Copy of http.Header of the Response
//  - parameter 2 - Copy of string of the Response's Body
//  - parameter 3 - Copy of http.Header of the Request
//
// Return values:
//  - value 1 - Response's amended header
//  - value 2 - Response's amended body
// Deprecated: Use ResponseFilterFunc instead.
type ResponseFilterFunc func(http.Header, []byte, http.Header) (*http.Header, *[]byte)

// ResponseFilter returns the ResponseFilterFunc as a ResponseFilter.
func (r ResponseFilterFunc) ResponseFilter() ResponseFilter {
	return func(resp Response) Response {
		header, body := r(resp.req.Header, resp.Body, resp.Header)
		if header != nil {
			resp.Header = *header
		}
		if body != nil {
			resp.Body = *body
		}
		return resp
	}
}

// vcrTransport is the heart of VCR. It provides
// an http.RoundTripper that wraps over the default
// one provided by Go's http package or a custom one
// if specified when calling NewVCR.
type vcrTransport struct {
	PCB      *pcb
	Cassette *cassette
}

// RoundTrip is an implementation of http.RoundTripper.
func (t *vcrTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		// Note: by convention resp should be nil if an error occurs with HTTP
		resp *http.Response

		requestMatched bool
		copiedReq      *http.Request
	)

	// copy the request before the body is closed by the HTTP server.
	copiedReq, err := copyRequest(req)
	if err != nil {
		t.PCB.Logger.Println(err)
		return nil, err
	}

	// get body data safely
	bodyData, err := readRequestBody(req)
	if err != nil {
		t.PCB.Logger.Println(err)
		return nil, err
	}

	request := Request{
		Header: req.Header,
		Body:   bodyData,
		Method: req.Method,
	}
	if req.URL != nil {
		request.URL = *req.URL
	}


	// attempt to use a track from the cassette that matches
	// the request if one exists.
	if trackNumber := t.PCB.seekTrack(t.Cassette, request); trackNumber != trackNotFound {
		// only the played back response is filtered. Never the live response!
		resp = t.PCB.filterResponse(t.Cassette.replayResponse(trackNumber, copiedReq), request)
		requestMatched = true
	}

	if !requestMatched {
		// no recorded track was found so execute the request live
		t.PCB.Logger.Printf("INFO - Cassette '%s' - Executing request to live server for %s %s\n", t.Cassette.Name, req.Method, req.URL.String())

		resp, err = t.PCB.Transport.RoundTrip(req)

		if !t.PCB.DisableRecording {
			// the VCR is not in read-only mode so
			// record the HTTP traffic into a new track on the cassette
			t.PCB.Logger.Printf("INFO - Cassette '%s' - Recording new track for %s %s\n", t.Cassette.Name, req.Method, req.URL.String())
			if err := recordNewTrackToCassette(t.Cassette, copiedReq, resp, err); err != nil {
				t.PCB.Logger.Println(err)
			}
		}
	}

	return resp, err
}

// copyRequest makes a copy an HTTP request.
// It ensures that the original request Body stream is restored to its original state
// and can be read from again.
// TODO: should perform a deep copy of the TLS property as with URL
func copyRequest(req *http.Request) (*http.Request, error) {
	if req == nil {
		return nil, nil
	}

	// get a deep copy without body considerations
	copiedReq := copyRequestWithoutBody(req)

	// deal with the Body
	bodyCopy, err := readRequestBody(req)
	if err != nil {
		return nil, err
	}

	// restore Body stream state
	req.Body = toReadCloser(bodyCopy)
	copiedReq.Body = toReadCloser(bodyCopy)

	return copiedReq, nil
}

// copyRequestWithoutBody makes a copy an HTTP request but not the Body (set to nil).
// TODO: should perform a deep copy of the TLS property as with URL
func copyRequestWithoutBody(req *http.Request) *http.Request {
	if req == nil {
		return nil
	}

	// get a shallow copy
	copiedReq := *req

	// remove the channel reference
	copiedReq.Cancel = nil

	// deal with the body
	copiedReq.Body = nil

	// deal with the URL (BEWARE obj == &*obj in Go, with obj being a pointer)
	if req.URL != nil {
		url := *req.URL
		if req.URL.User != nil {
			userInfo := *req.URL.User
			url.User = &userInfo
		}
		copiedReq.URL = &url
	}

	return &copiedReq
}

// readRequestBody reads the Body data stream and restores its states.
// It ensures the stream is restored to its original state and can be read from again.
func readRequestBody(req *http.Request) ([]byte, error) {
	if req == nil || req.Body == nil {
		return nil, nil
	}

	// dump the data
	bodyWriter := bytes.NewBuffer(nil)

	_, err := io.Copy(bodyWriter, req.Body)
	if err != nil {
		return nil, err
	}

	bodyData := bodyWriter.Bytes()

	// restore original state of the Body source stream
	req.Body.Close()
	req.Body = toReadCloser(bodyData)

	return bodyData, nil
}

// readResponseBody reads the Body data stream and restores its states.
// It ensures the stream is restored to its original state and can be read from again.
func readResponseBody(resp *http.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, nil
	}

	// dump the data
	bodyWriter := bytes.NewBuffer(nil)

	_, err := io.Copy(bodyWriter, resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	bodyData := bodyWriter.Bytes()

	// restore original state of the Body source stream
	resp.Body = toReadCloser(bodyData)

	return bodyData, nil
}

func toReadCloser(body []byte) io.ReadCloser {
	return ioutil.NopCloser(bytes.NewReader(body))
}
