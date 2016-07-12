package govcr

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

// VCR holds the internal parts of a VCR.
// Client is the HTTP client associated to VCR.
type VCR struct {
	Client *http.Client
}

// Stats returns Stats about the cassette and VCR session.
func (vcr *VCR) Stats() Stats {
	vcrT := vcr.Client.Transport.(*vcrTransport)
	return vcrT.Cassette.Stats()
}

// NewVCR creates a new VCR and loads a cassette.
// A RoundTripper can be provided when a custom
// Transport is needed (such as one to provide
// certificates, etc)
func NewVCR(cassetteName string, rt http.RoundTripper) *VCR {
	// use a default transport if none provided
	if rt == nil {
		rt = http.DefaultTransport
	}

	// load cassette
	cassette, err := loadCassette(cassetteName)
	if err != nil {
		log.Fatal(err)
	}

	// return
	return &VCR{
		Client: &http.Client{
			// TODO: BUG should also copy all other Client attributes such as Timeout, Jar, etc
			Transport: &vcrTransport{
				Transport: rt,
				Cassette:  cassette,
			},
		},
	}
}

// vcrTransport is the heart of VCR. It provides
// an http.RoundTripper that wraps over the default
// one provided by Go's http package or a custom one
// if specified when calling NewVCR.
type vcrTransport struct {
	Transport http.RoundTripper
	Cassette  *cassette
}

// RoundTrip is an implementation of http.RoundTripper.
func (t *vcrTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		resp = &http.Response{}
		err  error
	)

	responseMatched := false

	// attempt to use a track from the cassette that matches
	// the request if one exists.
	for _, track := range t.Cassette.Tracks {
		// TODO: matching requests should be more specific (incluing the body of the request)
		if !track.replayed &&
			track.Request.Method == req.Method &&
			track.Request.URL.String() == req.URL.String() {
			log.Printf("INFO - Cassette '%s' - Replaying roundtrip from track '%s' '%s'", t.Cassette.Name, req.Method, req.URL.String())

			// create a ReadCloser to supply to resp
			bodyReadCloser := ioutil.NopCloser(strings.NewReader(track.Response.Body))

			// create error object
			if track.ErrType != "" {
				err = errors.New(track.ErrType + ": " + track.ErrMsg)
			}

			// re-create the response object from track record
			if err == nil {
				tls := track.Response.TLS

				resp.Status = track.Response.Status
				resp.StatusCode = track.Response.StatusCode
				resp.Proto = track.Response.Proto
				resp.ProtoMajor = track.Response.ProtoMajor
				resp.ProtoMinor = track.Response.ProtoMinor

				resp.Header = track.Response.Header
				resp.Body = bodyReadCloser
				resp.ContentLength = track.Response.ContentLength
				resp.TransferEncoding = track.Response.TransferEncoding
				resp.Trailer = track.Response.Trailer
				resp.Request = req
				resp.TLS = tls
			}

			// mark the track as replayed so it doesn't get re-used
			track.replayed = true
			t.Cassette.stats.TracksPlayed++

			// mark the response for the request as found
			responseMatched = true

			break
		}
	}

	if !responseMatched {
		// no recorded track was found so execute the
		// request live and record into a new track on
		// the cassette
		log.Printf("INFO - Cassette '%s' - No track found for '%s' '%s' in the tracks that remain at this stage (%#v). Recording a new track from live server", t.Cassette.Name, req.Method, req.URL.String(), t.Cassette.Tracks)

		resp, err = t.Transport.RoundTrip(req)
		recordNewTrackToCassette(t.Cassette, req, resp, err)
	}

	return resp, err
}
