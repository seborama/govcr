package govcr

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
)

// request is a recorded HTTP request.
type request struct {
	Method    string
	URL       *url.URL
	HeaderMap http.Header
	Body      string
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

// newTrack creates a new track from an HTTP request and response.
func newTrack(req *http.Request, resp *http.Response, reqErr error) (*track, error) {
	var (
		k7Request  request
		k7Response response
	)

	// build request object
	if req != nil {
		var (
			data []byte
			err  error
		)

		if req.Body != nil {
			body := ioutil.NopCloser(req.Body)
			data, err = ioutil.ReadAll(body)
			if err != nil {
				log.Println(err)
				// continue nonetheless
			}

			// reset the Body on req
			req.Body = ioutil.NopCloser(bytes.NewReader(data))
		}

		k7Request = request{
			Method:    req.Method,
			URL:       req.URL,
			HeaderMap: req.Header,
			Body:      string(data),
		}
	}

	// build response object
	if resp != nil {
		var (
			data []byte
			err  error
		)

		if resp.Body != nil {
			body := ioutil.NopCloser(resp.Body)
			data, err = ioutil.ReadAll(body)
			if err != nil {
				log.Println(err)
				// continue nonetheless
			}

			// reset the Body on resp
			resp.Body = ioutil.NopCloser(bytes.NewReader(data))
		}

		k7Response = response{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
			Proto:      resp.Proto,
			ProtoMajor: resp.ProtoMajor,
			ProtoMinor: resp.ProtoMinor,

			Header:           resp.Header,
			Body:             string(data),
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

// cassette contains a set of tracks.
type cassette struct {
	Name   string
	Tracks []track
	stats  Stats
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

// DeleteCassette removes the cassette file from disk.
func DeleteCassette(cassetteName string) error {
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
	transformInterfacesInJSON(data)

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
	// TODO: BUG http.Response.TLS.PeerCertificates[xxx].PublicKey is an interface{} - See http://stackoverflow.com/questions/28254102/how-to-unmarshal-json-into-interface-in-golang?rq=1
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
