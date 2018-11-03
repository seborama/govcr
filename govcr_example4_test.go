package govcr_test

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/seborama/govcr"
)

const example4CassetteName = "MyCassette4"

func runTestEx4() {
	// Create vcr and make http call
	vcr := govcr.NewVCR(example4CassetteName, nil)
	resp, _ := vcr.Client.Get("http://www.example.com/foo")

	// Show results
	fmt.Printf("%d ", resp.StatusCode)
	fmt.Printf("%s ", resp.Header.Get("Content-Type"))

	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Printf("%v ", strings.Contains(string(body), "domain in examples without prior coordination or asking for permission."))

	fmt.Printf("%+v\n", vcr.Stats())
}

// Example_simpleVCR is an example use of govcr.
// It shows a simple use of a Long Play cassette (i.e. compressed).
func Example_number4SimpleVCR() {
	// Delete cassette to enable live HTTP call
	govcr.DeleteCassette(example4CassetteName, "")

	// 1st run of the test - will use live HTTP calls
	runTestEx4()
	// 2nd run of the test - will use playback
	runTestEx4()

	// Output:
	// 404 text/html; charset=UTF-8 true {TracksLoaded:0 TracksRecorded:1 TracksPlayed:0}
	// 404 text/html; charset=UTF-8 true {TracksLoaded:1 TracksRecorded:0 TracksPlayed:1}
}
