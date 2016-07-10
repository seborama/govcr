package govcr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
)

// response is a recorded HTTP response.
type response struct {
	Code      int
	HeaderMap http.Header
	Body      string
}

func StartVCR(cassetteName string) *http.Client {
	cassette, err := loadCassette(cassetteName)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: offer a control shutdown of the test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseMatched := false

		for _, track := range cassette.Tracks {
			log.Printf("DEBUG - track.request.method=%s\n", track.Request.Method)
			log.Printf("DEBUG - r.Method=%s\n", r.Method)
			log.Printf("DEBUG - track.request.url.String()=%s\n", track.Request.URL.String())
			log.Printf("DEBUG - r.URL.Path=%s\n", r.URL.Path)

			if track.Request.Method == r.Method &&
				track.Request.URL.String() == r.URL.String() {
				// TODO: implement status code and headers
				fmt.Fprintln(w, track.Response.Body)

				// mark the track as replayed so it doesn't get re-used
				track.replayed = true
				// mark the response for the request as found
				responseMatched = true

				break
			}
		}

		if !responseMatched {
			// TODO: here would be a good place to make the real HTTP call to RTI and record the response
			log.Printf("INFO - Cassette '%s' - No track found for '%s' '%s' in the tracks that remain at this stage (%#v)", cassetteName, r.Method, r.URL.String(), cassette.Tracks)

			req, err := http.NewRequest("GET", "http://example.com/foo", nil)
			if err != nil {
				log.Fatal(err)
			}

			resp, err := recordTrackToCassette(cassette, req)
			if err != nil {
				log.Fatal(err)
			}

			// TODO: implement status code and headers
			w.Write(resp.Body.Bytes())
		}
	}))

	c := &http.Client{
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				url, err := url.Parse(ts.URL)
				return url, err
			},
		},
	}

	return c
}

// recordTrackToCassette saves a new track to a cassette.
func recordTrackToCassette(cassette *cassette, req *http.Request) (*httptest.ResponseRecorder, error) {
	w := httptest.NewRecorder()

	// execute HTTP request
	err := handler(w, req)
	if err != nil {
		return nil, err
	}

	// create track
	track, err := newTrack(req, w)
	if err != nil {
		return nil, err
	}

	// add track to cassette
	cassette.addTrack(track)

	// save cassette
	err = cassette.save()
	if err != nil {
		return nil, err
	}

	return w, nil
}

// readCassetteFromFile reads the cassette file, if present.
func readCassetteFromFile(cassetteName string) (*cassette, error) {
	filename := "/tmp/govcr/fixtures/" + cassetteName + ".cassette"

	// check file existence
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		log.Println(err)
		return nil, err
	} else if err != nil {
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

// handler executes the request and saves the data
// to the supplied ResponseWriter.
func handler(w http.ResponseWriter, r *http.Request) error {
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		log.Println(err)
		return err
	}

	// record the headers
	for k, v1 := range resp.Header {
		for _, v2 := range v1 {
			w.Header().Add(k, v2)
		}
	}

	// record the body
	body := ioutil.NopCloser(resp.Body)
	bodyData, err := ioutil.ReadAll(body)
	if err != nil {
		log.Println(err)
		return err
	}
	w.Write([]byte(bodyData))

	return nil
}

// newTrack creates a new track from an HTTP request and response.
func newTrack(req *http.Request, w *httptest.ResponseRecorder) (*track, error) {
	log.Printf("DEBUG - req=%#v\n", req)
	log.Printf("DEBUG - w=%#v\n", w)
	var reqBodyData []byte
	var respBodyData []byte

	// build request object
	if req.Body != nil {
		var err error

		reqBody := ioutil.NopCloser(req.Body)
		reqBodyData, err = ioutil.ReadAll(reqBody)
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
		var err error

		respBody := ioutil.NopCloser(w.Body)
		respBodyData, err = ioutil.ReadAll(respBody)
		if err != nil {
			log.Println(err)
			return nil, err
		}
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

// request is a recorded HTTP request.
type request struct {
	Method    string
	URL       url.URL
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

// cassette contains a set of tracks.
type cassette struct {
	Name   string
	Tracks []track
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
	filename := "/tmp/govcr/fixtures/" + k7.Name + ".cassette"
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

// ErrUnknownCassette is an error that occurs when the cassette could not be read from file.
type ErrUnknownCassette string

// NewErrUnknownCassette is a constructor.
func NewErrUnknownCassette(cassetteName string) ErrUnknownCassette {
	return ErrUnknownCassette(fmt.Sprintf("unknown cassette '%s'", cassetteName))
}

func (e ErrUnknownCassette) Error() string {
	return string(e)
}
