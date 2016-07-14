package govcr

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
)

type header http.Header

// request is a recorded HTTP request.
type request struct {
	Method string
	URL    *url.URL
	Header header
	Body   string
}

// Resembles compares HTTP headers for equivalence.
// TODO: BUG should not compare ALL headers (particularly those with timestamps, etc).
func (h *header) Resembles(actual http.Header) bool {
	if len(*h) != len(actual) {
		log.Printf("DEBUG - Resembles - Headers length mismatch: %d vs %d\n", len(*h), len(actual))
		log.Printf("DEBUG - Resembles - Headers: %#v vs %#v\n", *h, actual)
		return false
	}

	for k, v1 := range *h {
		log.Printf("DEBUG - Resembles - Comparing keys: %#v vs %#v\n", actual.Get(k), v1)
		for _, v2 := range v1 {
			if actual.Get(k) != v2 {
				log.Printf("DEBUG - Resembles - Comparing values: %#v vs %#v\n", actual.Get(k), v2)
				return false
			}
		}
	}

	return true
}

// response is a recorded HTTP response.
type response struct {
	Status     string
	StatusCode int
	Proto      string
	ProtoMajor int
	ProtoMinor int

	Header           http.Header
	Body             string
	ContentLength    int64
	TransferEncoding []string
	Trailer          http.Header
	TLS              *tls.ConnectionState
}

// track is a recording (HTTP request + response) in a cassette.
type track struct {
	Request  request
	Response response
	ErrType  string
	ErrMsg   string

	// replayed indicates whether the track has already been processed in the cassette playback.
	replayed bool
}

func (t *track) response(req *http.Request) *http.Response {
	var (
		err  error
		resp = &http.Response{}
	)

	// create a ReadCloser to supply to resp
	bodyReadCloser := toReadCloser(t.Response.Body)

	// create error object
	switch t.ErrType {
	case "*net.OpError":
		err = &net.OpError{
			Op:     "govcr",
			Net:    "govcr",
			Source: nil,
			Addr:   nil,
			Err:    errors.New(t.ErrType + ": " + t.ErrMsg),
		}
	case "":
		err = nil

	default:
		err = errors.New(t.ErrType + ": " + t.ErrMsg)
	}

	if err != nil {
		// No need to parse the response.
		// By convention, when an HTTP error occurred, the response should be empty
		// (or Go's http package will show a warning message at runtime).
		return resp
	}

	// re-create the response object from track record
	tls := t.Response.TLS

	resp.Status = t.Response.Status
	resp.StatusCode = t.Response.StatusCode
	resp.Proto = t.Response.Proto
	resp.ProtoMajor = t.Response.ProtoMajor
	resp.ProtoMinor = t.Response.ProtoMinor

	resp.Header = t.Response.Header
	resp.Body = bodyReadCloser
	resp.ContentLength = t.Response.ContentLength
	resp.TransferEncoding = t.Response.TransferEncoding
	resp.Trailer = t.Response.Trailer

	// See notes on http.Response.Request - Body is nil because it has already been consumed
	resp.Request = copyRequestWithoutBody(req)

	resp.TLS = tls

	return resp
}

// Matches checks whether the track is a match for the supplied request.
func (t *track) Matches(req *http.Request) bool {
	if req == nil {
		return false
	}

	// get body data safely
	bodyData, err := readRequestBody(req)
	if err != nil {
		log.Println(err)
		return false
	}

	return !t.replayed &&
		t.Request.Method == req.Method &&
		t.Request.URL.String() == req.URL.String() &&
		t.Request.Header.Resembles(req.Header) &&
		t.Request.Body == bodyData
}

// newTrack creates a new track from an HTTP request and response.
func newTrack(req *http.Request, resp *http.Response, reqErr error) (*track, error) {
	var (
		k7Request  request
		k7Response response
	)

	// build request object
	if req != nil {
		bodyData, err := readRequestBody(req)
		if err != nil {
			log.Println(err)
			// continue nonetheless
		}

		k7Request = request{
			Method: req.Method,
			URL:    req.URL,
			Header: header(req.Header),
			Body:   bodyData,
		}
	}

	// build response object
	if resp != nil {
		bodyData, err := readResponseBody(resp)
		if err != nil {
			log.Println(err)
			// continue nonetheless
		}

		k7Response = response{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
			Proto:      resp.Proto,
			ProtoMajor: resp.ProtoMajor,
			ProtoMinor: resp.ProtoMinor,

			Header:           resp.Header,
			Body:             bodyData,
			ContentLength:    resp.ContentLength,
			TransferEncoding: resp.TransferEncoding,
			Trailer:          resp.Trailer,
			TLS:              resp.TLS,
		}
	}

	// build track object
	var reqErrType, reqErrMsg string
	if reqErr != nil {
		reqErrType = fmt.Sprintf("%T", reqErr)
		reqErrMsg = reqErr.Error()
	}

	track := &track{
		Request:  k7Request,
		Response: k7Response,
		ErrType:  reqErrType,
		ErrMsg:   reqErrMsg,
	}

	return track, nil
}

// Stats holds information about the cassette and
// VCR runtime.
type Stats struct {
	// TracksLoaded is the number of tracks that were loaded from the cassette.
	TracksLoaded int

	// TracksRecorded is the number of new tracks recorded by VCR.
	TracksRecorded int

	// TracksPlayed is the number of tracks played back straight from the cassette.
	// I.e. tracks that were already present on the cassette and were played back.
	TracksPlayed int
}

// cassette contains a set of tracks.
type cassette struct {
	Name   string
	Tracks []track

	// stats is unexported since it doesn't need serialising
	stats Stats
}

const trackNotFound = -1

func (k7 *cassette) seekTrack(req *http.Request) int {
	for idx, track := range k7.Tracks {
		if track.Matches(req) {
			log.Printf("INFO - Cassette '%s' - Found a track matching the request '%s' '%s'", k7.Name, req.Method, req.URL.String())
			return idx
		}
	}

	return trackNotFound
}

func (k7 *cassette) replayResponse(trackNumber int, req *http.Request) *http.Response {
	if trackNumber == trackNotFound || trackNumber >= len(k7.Tracks) {
		return nil
	}
	track := &k7.Tracks[trackNumber]

	// mark the track as replayed so it doesn't get re-used
	track.replayed = true

	// update Stats
	k7.stats.TracksPlayed++

	return track.response(req)
}

// DeleteCassette removes the cassette file from disk.
func DeleteCassette(cassetteName string) error {
	if !CassetteExists(cassetteName) {
		return nil
	}

	filename := cassetteNameToFilename(cassetteName)

	err := os.Remove(filename)
	if err != nil {
		log.Println(err)
	}

	return err
}

// CassetteExists verifies a cassette exists and is seemingly valid.
func CassetteExists(cassetteName string) bool {
	_, err := readCassetteFromFile(cassetteName)
	if err != nil {
		log.Println(err)
		return false
	}

	return true
}

// cassetteNameToFilename returns the filename associated to the cassette.
func cassetteNameToFilename(cassetteName string) string {
	if cassetteName == "" {
		return ""
	}

	return "./govcr-fixtures/" + cassetteName + ".cassette"
}

// saveCassette writes a cassette to file.
func (k7 *cassette) save() error {
	// marshal
	data, err := json.Marshal(k7)
	if err != nil {
		log.Println(err)
		return err
	}

	// transform properties known to fail on Unmarshal
	data = transformInterfacesInJSON(data)

	// beautify JSON (now that the JSON text has been transformed)
	var iData bytes.Buffer

	err = json.Indent(&iData, data, "", "  ")
	if err != nil {
		log.Println(err)
		return err
	}

	// write cassette to file
	filename := cassetteNameToFilename(k7.Name)
	path := filepath.Dir(filename)
	if err := os.MkdirAll(path, 0750); err != nil {
		log.Println(err)
		return err
	}

	if err := ioutil.WriteFile(filename, iData.Bytes(), 0640); err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// transformInterfacesInJSON looks for known properties in the JSON that are defined as interface{}
// in their original Go structure and don't UnMarshal correctly.
//
// Example x509.Certificate.PublicKey:
// When the type is rsa.PublicKey, Unmarshal attempts to map property "N" to a float64 because it is a number.
// However, it really is a big.Int which does not fit float64 and makes Unmarshal fail.
//
// This is not an ideal solution but it works. In the future, we could consider adding a property that
// records the original type and re-creates it post Unmarshal.
func transformInterfacesInJSON(jsonString []byte) []byte {
	regex, err := regexp.Compile(`("PublicKey":{"N":)([0-9]+),`)
	if err != nil {
		log.Fatalln(err)
	}

	return []byte(regex.ReplaceAllString(string(jsonString), `$1"$2",`))
}

// addTrack adds a track to a cassette.
func (k7 *cassette) addTrack(track *track) {
	k7.Tracks = append(k7.Tracks, *track)
	k7.stats.TracksRecorded++
}

// Stats returns the cassette's Stats.
func (k7 *cassette) Stats() Stats {
	return k7.stats
}

func loadCassette(cassetteName string) (*cassette, error) {
	k7, err := readCassetteFromFile(cassetteName)
	if err != nil && !os.IsNotExist(err) {
		log.Println(err)
		return nil, err
	}

	// provide an empty cassette as a minimum
	if k7 == nil {
		log.Println("WARNING - loadCassette - No cassette. Creating a blank one")
		k7 = &cassette{Name: cassetteName}
	}

	// initial stats
	k7.stats.TracksLoaded = len(k7.Tracks)

	return k7, nil
}

// readCassetteFromFile reads the cassette file, if present.
func readCassetteFromFile(cassetteName string) (*cassette, error) {
	filename := cassetteNameToFilename(cassetteName)

	// retrieve cassette from file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// unmarshal
	cassette := &cassette{}
	// NOTE: Properties which are of type 'interface{}' are not handled very well
	if err := json.Unmarshal(data, cassette); err != nil {
		log.Println(err)
		return nil, err
	}

	return cassette, nil
}

// recordNewTrackToCassette saves a new track to a cassette.
func recordNewTrackToCassette(cassette *cassette, req *http.Request, resp *http.Response, httpErr error) error {
	// create track
	track, err := newTrack(req, resp, httpErr)
	if err != nil {
		return err
	}

	// mark track as replayed since it's coming from a live request!
	track.replayed = true

	// add track to cassette
	cassette.addTrack(track)

	// save cassette
	return cassette.save()
}
