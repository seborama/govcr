package govcr

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
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
	Client *http.Client

	// Filter to run before request is matched against cassettes.
	RequestFilters RequestFilters

	// Filter to run before a response is returned.
	ResponseFilters ResponseFilters

	// Filter to run before storing a request/response pair in a track
	TrackFilters TrackFilters

	// LongPlay will compress data on cassettes.
	LongPlay         bool
	DisableRecording bool
	Logging          bool
	CassettePath     string

	// RemoveTLS will remove TLS from the Response when recording.
	// TLS information is rarely needed and takes up a lot of space.
	RemoveTLS bool
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

	// load cassette
	cassette, err := loadCassette(cassetteName, vcrConfig.CassettePath)
	if err != nil {
		logger.Fatal(err)
	}
	cassette.removeTLS = vcrConfig.RemoveTLS

	// create PCB
	pcbr := &pcb{
		// TODO: create appropriate test!
		DisableRecording: vcrConfig.DisableRecording,
		Transport:        vcrConfig.Client.Transport,
		RequestFilter:    vcrConfig.RequestFilters.combined(),
		ResponseFilter:   vcrConfig.ResponseFilters.combined(),
		TrackFilter:      vcrConfig.TrackFilters.combined(),
		Logger:           logger,
		CassettePath:     vcrConfig.CassettePath,
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

func newRequest(req *http.Request, logger *log.Logger) (Request, error) {
	bodyData, err := readRequestBody(req)
	if err != nil {
		logger.Println(err)
		return Request{}, err
	}

	request := Request{
		Header: cloneHeader(req.Header),
		Body:   bodyData,
		Method: req.Method,
	}

	if req.URL != nil {
		request.URL = *copyURL(req.URL)
	}

	return request, nil
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

	// deal with the URL
	if req.URL != nil {
		copiedReq.URL = copyURL(req.URL)
	}
	copiedReq.Header = cloneHeader(req.Header)

	return &copiedReq
}

func copyURL(url *url.URL) *url.URL {
	// shallow copy
	copiedURL := *url

	if url.User != nil {
		// BEWARE: obj == &*obj in Go, with obj being a pointer
		userInfo := *url.User
		copiedURL.User = &userInfo
	}

	return &copiedURL
}

// cloneHeader return return a deep copy of the header.
func cloneHeader(h http.Header) http.Header {
	if h == nil {
		return nil
	}

	copied := make(http.Header, len(h))
	for k, v := range h {
		copied[k] = append([]string{}, v...)
	}
	return copied
}

// readRequestBody reads the Body data stream and restores its states.
// It ensures the stream is restored to its original state and can be read from again.
// TODO - readRequestBody and readResponseBody are so similar - perhaps create a new interface Bodyer and extend http.Request and http.Response to implement it. This would allow to merge readRequestBody and readResponseBody
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

// copyResponse makes a copy an HTTP response.
// It ensures that the original response Body stream is restored to its original state
// and can be read from again.
// TODO: should perform a deep copy of the TLS property as with URL
func copyResponse(resp *http.Response) (*http.Response, error) {
	if resp == nil {
		return nil, nil
	}

	// get a shallow copy
	copiedResp := *resp

	copiedResp.Header = cloneHeader(resp.Header)

	// deal with the Body
	bodyCopy, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}

	// restore Body stream state
	resp.Body = toReadCloser(bodyCopy)
	copiedResp.Body = toReadCloser(bodyCopy)

	return &copiedResp, nil
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
