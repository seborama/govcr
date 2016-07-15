package govcr

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
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

// VCRConfig holds a set of options for the VCR.
// TODO: update README.md with details of DisableRecording and FilterFunc's
type VCRConfig struct {
	Client                *http.Client
	DisableRecording      bool
	ExcludeHeaderFunc     ExcludeHeaderFunc
	RequestBodyFilterFunc BodyFilterFunc
}

// PCB stands for Printed Circuit Board. It is a structure that holds some
// facilities that are passed to the VCR machine to modify its internals.
type pcb struct {
	DisableRecording      bool
	Transport             http.RoundTripper
	ExcludeHeaderFunc     ExcludeHeaderFunc
	RequestBodyFilterFunc BodyFilterFunc
}

const trackNotFound = -1

func (pcbr *pcb) seekTrack(cassette *cassette, req *http.Request) int {
	for idx := range cassette.Tracks {
		if pcbr.trackMatches(cassette, idx, req) {
			log.Printf("INFO - Cassette '%s' - Found a matching track for %s %s\n", cassette.Name, req.Method, req.URL.String())
			return idx
		}
	}

	return trackNotFound
}

// Matches checks whether the track is a match for the supplied request.
func (pcbr *pcb) trackMatches(cassette *cassette, trackNumber int, req *http.Request) bool {
	if req == nil {
		return false
	}

	// get body data safely
	bodyData, err := readRequestBody(req)
	if err != nil {
		log.Println(err)
		return false
	}

	track := cassette.Tracks[trackNumber]

	return !track.replayed &&
		track.Request.Method == req.Method &&
		track.Request.URL.String() == req.URL.String() &&
		pcbr.headerResembles(track.Request.Header, req.Header) &&
		pcbr.bodyResembles(track.Request.Body, bodyData)
}

// Resembles compares HTTP headers for equivalence.
func (pcbr *pcb) headerResembles(header1 http.Header, header2 http.Header) bool {
	for k, v1 := range header1 {
		for _, v2 := range v1 {
			if header2.Get(k) != v2 && !pcbr.ExcludeHeaderFunc(k) {
				return false
			}
		}
	}

	// finally assert the number of headers match
	// TODO: perhaps should count how manypcb.ExcludeHeaderFunc() returned true and removes that count from the len to compare?
	if len(header1) != len(header2) {
		return false
	}

	return true
}

// Resembles compares HTTP bodies for equivalence.
func (pcbr *pcb) bodyResembles(body1 string, body2 string) bool {
	return *pcbr.RequestBodyFilterFunc(body1) == *pcbr.RequestBodyFilterFunc(body2)
}

// NewVCR creates a new VCR and loads a cassette.
// A RoundTripper can be provided when a custom Transport is needed (for example to provide
// certificates, etc)
func NewVCR(cassetteName string, vcrConfig *VCRConfig) *VCRControlPanel {
	if vcrConfig == nil {
		vcrConfig = &VCRConfig{}
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
			return true
		}
	}

	if vcrConfig.RequestBodyFilterFunc == nil {
		vcrConfig.RequestBodyFilterFunc = func(body string) *string {
			return &body
		}
	}

	// load cassette
	cassette, err := loadCassette(cassetteName)
	if err != nil {
		log.Fatal(err)
	}

	// create PCB
	pcbr := &pcb{
		// TODO: create appropriate test!
		DisableRecording:      vcrConfig.DisableRecording,
		Transport:             vcrConfig.Client.Transport,
		ExcludeHeaderFunc:     vcrConfig.ExcludeHeaderFunc,
		RequestBodyFilterFunc: vcrConfig.RequestBodyFilterFunc,
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
type ExcludeHeaderFunc func(key string) bool

// BodyFilterFunc is a hook function that is used to filter the Body.
//
// Typically this can be used to remove / amend undesirable body elements from the request.
//
// For instance, if your application sends requests with a timestamp held in a part of the body,
// you likely want to remove it or force a static timestamp via BodyFilterFunc to
// ensure that the request body matches those saved on the cassette's track.
type BodyFilterFunc func(string) *string

// vcrTransport is the heart of VCR. It provides
// an http.RoundTripper that wraps over the default
// one provided by Go's http package or a custom one
// if specified when calling NewVCR.
type vcrTransport struct {
	PCB               *pcb
	Cassette          *cassette
	ExcludeHeaderFunc ExcludeHeaderFunc
	RequestBodyFilter BodyFilterFunc
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
		log.Println(err)
		return nil, err
	}

	// attempt to use a track from the cassette that matches
	// the request if one exists.
	if trackNumber := t.PCB.seekTrack(t.Cassette, copiedReq); trackNumber != trackNotFound {
		resp = t.Cassette.replayResponse(trackNumber, copiedReq)
		requestMatched = true
	}

	if !requestMatched {
		// no recorded track was found so execute the request live
		log.Printf("INFO - Cassette '%s' - Executing request from live server for %s %s\n", t.Cassette.Name, req.Method, req.URL.String())

		resp, err = t.PCB.Transport.RoundTrip(req)

		if !t.PCB.DisableRecording {
			// the VCR is not in read-only mode so
			// record the HTTP traffic into a new track on the cassette
			log.Printf("INFO - Cassette '%s' - Recording new track for %s %s\n", t.Cassette.Name, req.Method, req.URL.String())
			recordNewTrackToCassette(t.Cassette, copiedReq, resp, err)
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
		log.Println(err)
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
func readRequestBody(req *http.Request) (string, error) {
	if req == nil || req.Body == nil {
		return "", nil
	}

	// dump the data
	bodyWriter := bytes.NewBuffer(nil)

	_, err := io.Copy(bodyWriter, req.Body)
	if err != nil {
		log.Println(err)
		return "", err
	}

	bodyData := bodyWriter.String()

	// restore original state of the Body source stream
	req.Body.Close()
	req.Body = toReadCloser(bodyData)

	return bodyData, nil
}

// readResponseBody reads the Body data stream and restores its states.
// It ensures the stream is restored to its original state and can be read from again.
func readResponseBody(resp *http.Response) (string, error) {
	if resp == nil || resp.Body == nil {
		return "", nil
	}

	// dump the data
	bodyWriter := bytes.NewBuffer(nil)

	_, err := io.Copy(bodyWriter, resp.Body)
	if err != nil {
		log.Println(err)
		return "", err
	}
	resp.Body.Close()

	bodyData := bodyWriter.String()

	// restore original state of the Body source stream
	resp.Body = toReadCloser(bodyData)

	return bodyData, nil
}

func toReadCloser(body string) io.ReadCloser {
	return ioutil.NopCloser(bytes.NewReader([]byte(body)))
}
