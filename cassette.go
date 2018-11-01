package govcr

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// request is a recorded HTTP request.
type request struct {
	Method string
	URL    *url.URL
	Header http.Header
	Body   []byte
}

// Request transforms internal request to a filter request.
func (r request) Request() Request {
	res := Request{
		Header: r.Header,
		Body:   r.Body,
		Method: r.Method,
	}
	if r.URL != nil {
		res.URL = *r.URL
	}
	return res
}

// response is a recorded HTTP response.
type response struct {
	Status     string
	StatusCode int
	Proto      string
	ProtoMajor int
	ProtoMinor int

	Header           http.Header
	Body             []byte
	ContentLength    int64
	TransferEncoding []string
	Trailer          http.Header
	TLS              *tls.ConnectionState
}

// Response returns the internal response to a filter response.
func (r response) Response(req Request) Response {
	return Response{
		req:        req,
		Body:       r.Body,
		Header:     r.Header,
		StatusCode: r.StatusCode,
	}
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
			return nil, err
		}

		k7Request = request{
			Method: req.Method,
			URL:    req.URL,
			Header: req.Header,
			Body:   bodyData,
		}
	}

	// build response object
	if resp != nil {
		bodyData, err := readResponseBody(resp)
		if err != nil {
			return nil, err
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
	Name, Path string
	Tracks     []track

	// stats is not exported since it doesn't need serialising
	stats Stats
}

func (k7 *cassette) isLongPlay() bool {
	return strings.HasSuffix(k7.Name, ".gz")
}

func (k7 *cassette) replayResponse(trackNumber int, req *http.Request) *http.Response {
	if trackNumber == trackNotFound || trackNumber >= len(k7.Tracks) {
		return nil
	}
	track := &k7.Tracks[trackNumber]

	// mark the track as replayed so it doesn't get re-used
	track.replayed = true

	return track.response(req)
}

// saveCassette writes a cassette to file.
func (k7 *cassette) save() error {
	data, err := json.Marshal(k7)
	if err != nil {
		return err
	}

	tData, err := transformInterfacesInJSON(data)
	if err != nil {
		return err
	}

	// beautify JSON (now that the JSON text has been transformed)
	var iData bytes.Buffer
	if err := json.Indent(&iData, tData, "", "  "); err != nil {
		return err
	}

	filename := cassetteNameToFilename(k7.Name, k7.Path)
	path := filepath.Dir(filename)
	if err := os.MkdirAll(path, 0750); err != nil {
		return err
	}

	gData, err := k7.gzipFilter(iData)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, gData, 0640)
}

// gzipFilter compresses the cassette data in gzip format if the cassette
// name ends with '.gz', otherwise data is left as is (i.e. de-compressed)
func (k7 *cassette) gzipFilter(data bytes.Buffer) ([]byte, error) {
	if k7.isLongPlay() {
		return compress(data.Bytes())
	}
	return data.Bytes(), nil
}

// gunzipFilter de-compresses the cassette data in gzip format if the cassette
// name ends with '.gz', otherwise data is left as is (i.e. de-compressed)
func (k7 *cassette) gunzipFilter(data []byte) ([]byte, error) {
	if k7.isLongPlay() {
		return decompress(data)
	}
	return data, nil
}

// addTrack adds a track to a cassette.
func (k7 *cassette) addTrack(track *track) {
	k7.Tracks = append(k7.Tracks, *track)
}

// Stats returns the cassette's Stats.
func (k7 *cassette) Stats() Stats {
	k7.stats.TracksRecorded = k7.numberOfTracks() - k7.stats.TracksLoaded
	k7.stats.TracksPlayed = k7.tracksPlayed() - k7.stats.TracksRecorded

	return k7.stats
}

func (k7 *cassette) tracksPlayed() int {
	replayed := 0

	for _, t := range k7.Tracks {
		if t.replayed {
			replayed++
		}
	}

	return replayed
}

func (k7 *cassette) numberOfTracks() int {
	return len(k7.Tracks)
}

// DeleteCassette removes the cassette file from disk.
func DeleteCassette(cassetteName, cassettePath string) error {
	filename := cassetteNameToFilename(cassetteName, cassettePath)

	err := os.Remove(filename)
	if os.IsNotExist(err) {
		// the file does not exist so this is not an error since we wanted it gone!
		err = nil
	}

	return err
}

// CassetteExistsAndValid verifies a cassette file exists and is seemingly valid.
func CassetteExistsAndValid(cassetteName, cassettePath string) bool {
	_, err := readCassetteFromFile(cassetteName, cassettePath)
	return err == nil
}

// cassetteNameToFilename returns the filename associated to the cassette.
func cassetteNameToFilename(cassetteName, cassettePath string) string {
	if cassetteName == "" {
		return ""
	}

	if cassettePath == "" {
		cassettePath = defaultCassettePath
	}

	fPath, err := filepath.Abs(filepath.Join(cassettePath, cassetteName+".cassette"))
	if err != nil {
		log.Fatal(err)
	}

	return fPath
}

// transformInterfacesInJSON looks for known properties in the JSON that are defined as interface{}
// in their original Go structure and don't Unmarshal correctly.
//
// Example x509.Certificate.PublicKey:
// When the type is rsa.PublicKey, Unmarshal attempts to map property "N" to a float64 because it is a number.
// However, it really is a big.Int which does not fit float64 and makes Unmarshal fail.
//
// This is not an ideal solution but it works. In the future, we could consider adding a property that
// records the original type and re-creates it post Unmarshal.
func transformInterfacesInJSON(jsonString []byte) ([]byte, error) {
	// TODO: precompile this regexp perhaps via a receiver
	regex, err := regexp.Compile(`("PublicKey":{"N":)([0-9]+),`)
	if err != nil {
		return []byte{}, err
	}

	return []byte(regex.ReplaceAllString(string(jsonString), `$1"$2",`)), nil
}

func loadCassette(cassetteName, cassettePath string) (*cassette, error) {
	k7, err := readCassetteFromFile(cassetteName, cassettePath)
	if err != nil {
		return nil, err
	}

	// provide an empty cassette as a minimum
	if k7 == nil {
		k7 = &cassette{Name: cassetteName, Path: cassettePath}
	}

	// initial stats
	k7.stats.TracksLoaded = len(k7.Tracks)

	return k7, nil
}

// readCassetteFromFile reads the cassette file, if present.
func readCassetteFromFile(cassetteName, cassettePath string) (*cassette, error) {
	filename := cassetteNameToFilename(cassetteName, cassettePath)

	k7 := &cassette{
		Name: cassetteName,
		Path: cassettePath,
	}

	data, err := ioutil.ReadFile(filename)
	if os.IsNotExist(err) {
		return k7, nil
	} else if err != nil {
		return nil, err
	}

	cData, err := k7.gunzipFilter(data)
	if err != nil {
		return nil, err
	}

	// NOTE: Properties which are of type 'interface{}' are not handled very well
	if err := json.Unmarshal(cData, k7); err != nil {
		return nil, err
	}

	return k7, nil
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

// compress data and returns the result
func compress(data []byte) ([]byte, error) {
	var out bytes.Buffer

	w := gzip.NewWriter(&out)
	if _, err := io.Copy(w, bytes.NewBuffer(data)); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// decompress data and returns the result
func decompress(data []byte) ([]byte, error) {
	fmt.Println("DEBUG - NOT YET WRITTEN!!!!")
	return data, nil
}
