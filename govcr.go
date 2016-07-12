package govcr

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// StopVCRFunc stops the VCR server.
// Typically, this is called via a defer statement after calling StartVCR().
type StopVCRFunc func()

// VCR is the structure returned from StartVCR.
// It contains useful information for users of govcr.
type VCR struct {
	Client      *http.Client
	StopVCRFunc StopVCRFunc
}

// StartVCR brings up the proxy server and returns a VCR object helper
// for programmers to use govcr.
func StartVCR(cassetteName string) VCR {
	cassette, err := loadCassette(cassetteName)
	if err != nil {
		log.Fatal(err)
	}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("DEBUG - StartVCR - r url=: %s\n", r.URL.String())
		responseMatched := false

		for _, track := range cassette.Tracks {
			if track.Request.Method == r.Method &&
				track.Request.URL.String() == r.URL.String() {
				log.Printf("INFO - Cassette '%s' - Replaying roundtrip from track '%s' '%s'", cassetteName, r.Method, r.URL.String())

				// TODO: implement status code and headers
				fmt.Fprintln(w, track.Response.Body)

				// mark the track as replayed so it doesn't get replayed
				track.replayed = true
				// mark the response for the request as found
				responseMatched = true

				break
			}
		}

		if !responseMatched {
			// TODO: here would be a good place to make the real HTTP call to RTI and record the response
			log.Printf("INFO - Cassette '%s' - No track found for '%s' '%s' in the tracks that remain at this stage (%#v). Recording a new track from live HTTP", cassetteName, r.Method, r.URL.String(), cassette.Tracks)

			bodyData, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Fatal(err)
			}

			body := strings.NewReader(string(bodyData))

			log.Printf("DEBUG - r.URL.String()=%s\n", r.URL.String())
			req, err := http.NewRequest(r.Method, r.URL.String(), body)
			if err != nil {
				log.Fatal(err)
			}

			log.Println("DEBUG 0")
			resp, err := recordNewTrackToCassette(cassette, req)
			if err != nil {
				log.Fatal(err)
			}

			// TODO: implement status code and headers
			w.Write(resp.Body.Bytes())
		}
	}))

	log.Print("")
	if ts.Config.TLSConfig == nil {
		ts.Config.TLSConfig = &tls.Config{}
	}
	ts.Config.TLSConfig.InsecureSkipVerify = true

	c := &http.Client{
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				url, err := url.Parse(ts.URL)
				log.Printf("DEBUG - Proxy - req url=: %s\n", req.URL.String())
				log.Printf("DEBUG - Proxy - url=: %s\n", url.String())
				return url, err
			},
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	return VCR{
		Client:      c,
		StopVCRFunc: func() { ts.Close() },
	}
}

// recordNewTrackToCassette saves a new track to a cassette.
func recordNewTrackToCassette(cassette *cassette, req *http.Request) (*httptest.ResponseRecorder, error) {
	w := httptest.NewRecorder()

	// execute HTTP request
	log.Println("DEBUG 1")
	err := httpHandler(w, req)
	if err != nil {
		return nil, err
	}

	// create track
	log.Println("DEBUG 2")
	track, err := newTrack(req, w)
	if err != nil {
		return nil, err
	}

	// add track to cassette
	log.Println("DEBUG 3")
	cassette.addTrack(track)

	// save cassette
	log.Println("DEBUG 4")
	err = cassette.save()
	if err != nil {
		return nil, err
	}
	log.Println("DEBUG 5")

	return w, nil
}

// handler executes the request and saves the data
// to the supplied ResponseWriter.
func httpHandler(w http.ResponseWriter, r *http.Request) error {
	log.Println("DEBUG A")
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		log.Println(err)
		return err
	}

	// record the headers
	log.Println("DEBUG B")
	for k, v1 := range resp.Header {
		for _, v2 := range v1 {
			w.Header().Add(k, v2)
		}
	}

	// record the body
	log.Println("DEBUG C")
	body := ioutil.NopCloser(resp.Body)
	bodyData, err := ioutil.ReadAll(body)
	if err != nil {
		log.Println(err)
		return err
	}
	w.Write([]byte(bodyData))
	log.Println("DEBUG D")

	return nil
}

// request is a recorded HTTP request.
type request struct {
	Method    string
	URL       url.URL
	HeaderMap http.Header
	Body      string
}

// response is a recorded HTTP response.
type response struct {
	Code      int
	HeaderMap http.Header
	Body      string
}

// track is a recording (HTTP request + response) in a cassette.
type track struct {
	Request  request
	Response response

	// replayed indicates whether the track has already been processed in the cassette playback.
	replayed bool
}

// newTrack creates a new track from an HTTP request and response.
func newTrack(req *http.Request, w *httptest.ResponseRecorder) (*track, error) {
	var reqBodyData []byte
	var respBodyData []byte

	// build request object
	if req.Body != nil {
		var err error

		// See Golang source code for req.Body:
		//   The Server will close the request body. The ServeHTTP
		//   Handler does not need to.
		reqBodyData, err = ioutil.ReadAll(req.Body)
		if err != nil {
			log.Println(err)
			return nil, err
		}
	}
	request := request{
		Method:    req.Method,
		URL:       *req.URL,
		HeaderMap: req.Header,
		Body:      string(reqBodyData),
	}

	// build response object
	if w.Body != nil {
		respBodyData = w.Body.Bytes()
	}
	response := response{
		Code:      w.Code,
		HeaderMap: w.Header(),
		Body:      string(respBodyData),
	}

	// build track object
	track := &track{
		Request:  request,
		Response: response,
	}

	return track, nil
}

// cassette contains a set of tracks.
type cassette struct {
	Name   string
	Tracks []track
}

// DeleteCassette removes the cassette file from disk.
func DeleteCassette(cassetteName string) error {
	filename := cassetteNameToFile(cassetteName)

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

// cassetteNameToFile returns the filename associated to the cassette.
func cassetteNameToFile(cassetteName string) string {
	if cassetteName == "" {
		return ""
	}

	return "./govcr-fixtures/" + cassetteName + ".cassette"
}

// saveCassette writes a cassette to file.
func (k7 *cassette) save() error {
	// marshal
	data, err := json.MarshalIndent(k7, "", "  ")
	if err != nil {
		log.Println(err)
		return err
	}

	// write cassette to file
	filename := cassetteNameToFile(k7.Name)
	path := filepath.Dir(filename)
	if err := os.MkdirAll(path, 0750); err != nil {
		log.Println(err)
		return err
	}

	if err := ioutil.WriteFile(filename, data, 0640); err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// addTrack adds a track to a cassette.
func (k7 *cassette) addTrack(track *track) {
	k7.Tracks = append(k7.Tracks, *track)
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

	return k7, nil
}

// readCassetteFromFile reads the cassette file, if present.
func readCassetteFromFile(cassetteName string) (*cassette, error) {
	filename := cassetteNameToFile(cassetteName)

	// check file existence
	_, err := os.Stat(filename)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// retrieve cassette from file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// unmarshal
	cassette := &cassette{}
	if err := json.Unmarshal(data, cassette); err != nil {
		log.Println(err)
		return nil, err
	}

	return cassette, nil
}

// ErrUnknownCassette is an error that occurs when the cassette could not be read from file.
type ErrUnknownCassette string

// NewErrUnknownCassette is a constructor.
func NewErrUnknownCassette(cassetteName string) ErrUnknownCassette {
	return ErrUnknownCassette(fmt.Sprintf("unknown cassette '%s'", cassetteName))
}

func (e ErrUnknownCassette) Error() string {
	return string(e)
}
