package govcr

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

// VCRControlPannel holds the parts of a VCR that can be interacted with.
// Client is the HTTP client associated with the VCR.
type VCRControlPannel struct {
	Client *http.Client
}

// Stats returns Stats about the cassette and VCR session.
func (vcr *VCRControlPannel) Stats() Stats {
	vcrT := vcr.Client.Transport.(*vcrTransport)
	return vcrT.Cassette.Stats()
}

// PCB stands for Printer Circuit Board. It is a structure that holds some
// facilities that are passed to the VCR machine to modify its internals.
type PCB struct {
	Transport        http.RoundTripper
	HeaderFilterFunc *HeaderFilterFunc
}

// NewVCR creates a new VCR and loads a cassette.
// A RoundTripper can be provided when a custom
// Transport is needed (such as one to provide
// certificates, etc)
func NewVCR(cassetteName string, pcb *PCB) *VCRControlPannel {
	if pcb == nil {
		pcb = &PCB{}
	}

	// use a default transport if none provided
	if pcb.Transport == nil {
		pcb.Transport = http.DefaultTransport
	}

	// load cassette
	cassette, err := loadCassette(cassetteName)
	if err != nil {
		log.Fatal(err)
	}

	// return
	return &VCRControlPannel{
		Client: &http.Client{
			// TODO: BUG should also copy all other Client attributes such as Timeout, Jar, etc
			Transport: &vcrTransport{
				PCB:      pcb,
				Cassette: cassette,
			},
		},
	}
}

// HeaderFilterFunc is a hook function that is used to filter the Header.
//
// Typically this can be used to remove / amend undesirable custom headers from the request.
//
// For instance, if your application sends requests with a timestamp held in a custom header,
// you likely want to remove it or force a static timestamp via HeaderFilterFunc to
// ensure that the request headers match those saved on the cassette's track.
type HeaderFilterFunc func(*http.Header) *http.Header

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
	PCB      *PCB
	Cassette *cassette

	// TODO: this is currently ignored. Implement!
	RequestHeaderFilter *HeaderFilterFunc

	// TODO: this is currently ignored. Implement!
	RequestBodyFilter *BodyFilterFunc
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
	if trackNumber := t.Cassette.seekTrack(copiedReq); trackNumber != trackNotFound {
		resp = t.Cassette.replayResponse(trackNumber, copiedReq)
		requestMatched = true
	}

	if !requestMatched {
		// no recorded track was found so execute the
		// request live and record into a new track on
		// the cassette
		log.Printf("INFO - Cassette '%s' - No track found for '%s' '%s' in the tracks that remain at this stage in the cassette. Recording a new track from live server", t.Cassette.Name, req.Method, req.URL.String())

		resp, err = t.PCB.Transport.RoundTrip(req)
		recordNewTrackToCassette(t.Cassette, copiedReq, resp, err)
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
