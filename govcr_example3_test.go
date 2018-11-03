package govcr_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/seborama/govcr"
)

const example3CassetteName = "MyCassette3"

func runTestEx3() {
	var samples = []struct {
		method string
		body   string
	}{
		{"GET", "domain in examples without prior coordination or asking for permission."},
		{"POST", "404 - Not Found"},
		{"PUT", ""},
		{"DELETE", ""},
	}

	// Create vcr
	vcr := govcr.NewVCR(example3CassetteName,
		&govcr.VCRConfig{
			RequestFilters: govcr.RequestFilters{
				govcr.RequestDeleteHeaderKeys("X-Custom-My-Date"),
			},
		})

	for _, td := range samples {
		// Create a request with our custom header
		req, _ := http.NewRequest(td.method, "http://www.example.com/foo", nil)
		req.Header.Add("X-Custom-My-Date", time.Now().String())

		// Make http call
		resp, _ := vcr.Client.Do(req)

		// Show results
		fmt.Printf("%d ", resp.StatusCode)
		fmt.Printf("%s ", resp.Header.Get("Content-Type"))

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("%v ", strings.Contains(string(body), td.body))
	}

	fmt.Printf("%+v\n", vcr.Stats())
}

// Example_simpleVCR is an example use of govcr.
// It shows how to use govcr in the simplest case when the default
// http.Client suffices.
func Example_number3HeaderExclusionVCR() {
	// Delete cassette to enable live HTTP call
	govcr.DeleteCassette(example3CassetteName, "")

	// 1st run of the test - will use live HTTP calls
	runTestEx3()
	// 2nd run of the test - will use playback
	runTestEx3()

	// Output:
	// 404 text/html; charset=UTF-8 true 404 text/html; charset=UTF-8 true 404 text/html; charset=UTF-8 true 404 text/html; charset=UTF-8 true {TracksLoaded:0 TracksRecorded:4 TracksPlayed:0}
	// 404 text/html; charset=UTF-8 true 404 text/html; charset=UTF-8 true 404 text/html; charset=UTF-8 true 404 text/html; charset=UTF-8 true {TracksLoaded:4 TracksRecorded:0 TracksPlayed:4}
}
