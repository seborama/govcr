package govcr

import (
	"bytes"
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

func GetVCR(scenarioName string) *http.Client {
	scenario, err := loadScenarioRecordings(scenarioName)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: offer a control shutdown of the test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseMatched := false

		for _, step := range scenario.Steps {
			log.Printf("DEBUG - step.request.method=%s\n", step.Request.Method)
			log.Printf("DEBUG - r.Method=%s\n", r.Method)
			log.Printf("DEBUG - step.request.url.String()=%s\n", step.Request.URL.String())
			log.Printf("DEBUG - r.URL.Path=%s\n", r.URL.Path)

			if step.Request.Method == r.Method &&
				step.Request.URL.String() == r.URL.String() {
				// TODO: implement status code and headers
				fmt.Fprintln(w, step.Response.Body)

				// mark the step as replayed so it doesn't get re-used
				step.replayed = true
				// mark the response for the request as found
				responseMatched = true

				break
			}
		}

		if !responseMatched {
			// TODO: here would be a good place to make the real HTTP call to RTI and record the response
			log.Printf("INFO - Scenario '%s' - No step found for '%s' '%s' in the steps that remain at this stage (%#v)", scenarioName, r.Method, r.URL.String(), scenario.Steps)

			req, err := http.NewRequest("GET", "http://example.com/foo", nil)
			if err != nil {
				log.Fatal(err)
			}

			resp, err := Record(req, scenarioName)
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

// Record records the response from an HTTP request.
func Record(req *http.Request, filename string) (*httptest.ResponseRecorder, error) {
	w := httptest.NewRecorder()

	err := handler(w, req)
	if err != nil {
		return nil, err
	}

	err = persist(req, w, filename)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// Replay replays the response of an HTTP request.
func Replay(req *http.Request, filename string) (*httptest.ResponseRecorder, error) {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		log.Println(err)
		return nil, err
	}

	// retrieve previous recording from file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// unmarshal
	resp := &response{}
	if err := json.Unmarshal(data, resp); err != nil {
		log.Println(err)
		return nil, err
	}

	w := &httptest.ResponseRecorder{
		Code:      resp.Code,
		HeaderMap: resp.HeaderMap,
		Body:      bytes.NewBufferString(resp.Body),
	}

	return w, nil
}

// readScenarioFromFile reads the scenario file, if present.
func readScenarioFromFile(scenarioName string) (*scenario, error) {
	filename := scenarioName + ".rec"

	// check file existence
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		log.Println(err)
		return nil, err
	}

	// retrieve previous recording from file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// unmarshal
	scenario := &scenario{}
	if err := json.Unmarshal(data, scenario); err != nil {
		log.Println(err)
		return nil, err
	}

	return scenario, nil
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

// persist writes a scenario step to file.
func persist(req *http.Request, w *httptest.ResponseRecorder, filename string) error {
	reqBody := ioutil.NopCloser(req.Body)
	reqBodyData, err := ioutil.ReadAll(reqBody)
	if err != nil {
		log.Println(err)
		return err
	}

	request := request{
		Method:    req.Method,
		URL:       *req.URL,
		HeaderMap: req.Header,
		Body:      string(reqBodyData),
	}

	respBody := ioutil.NopCloser(w.Body)
	respBodyData, err := ioutil.ReadAll(respBody)
	if err != nil {
		log.Println(err)
		return err
	}
	response := response{
		Code:      w.Code,
		HeaderMap: w.Header(),
		Body:      string(respBodyData),
	}

	step := step{
		Request:  request,
		Response: response,
	}

	data, err := json.MarshalIndent(step, "", "  ")
	if err != nil {
		log.Println(err)
		return err
	}

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

// request is a recorded HTTP request.
type request struct {
	Method    string
	URL       url.URL
	HeaderMap http.Header
	Body      string
}

// step is a step (HTTP request + response) in a scenario.
type step struct {
	Request  request
	Response response

	// replayed indicates whether the step has already been processed in the scenario playback.
	replayed bool
}

// scenario is a set of steps.
type scenario struct {
	Name  string
	Steps []step
}

func loadScenarioRecordings(scenarioName string) (*scenario, error) {
	s, err := readScenarioFromFile(scenarioName)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return s, nil
}

// ErrUnknownScenario is an error that occurs when the scenario could not be read from file.
type ErrUnknownScenario string

// NewErrUnknownScenario is a constructor.
func NewErrUnknownScenario(scenarioName string) ErrUnknownScenario {
	return ErrUnknownScenario(fmt.Sprintf("unknown scenario '%s'", scenarioName))
}

func (e ErrUnknownScenario) Error() string {
	return string(e)
}
