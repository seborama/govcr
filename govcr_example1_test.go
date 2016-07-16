package govcr_test

import (
	"fmt"
	"io/ioutil"

	"strings"

	"github.com/seborama/govcr"
)

const example1CassetteName = "MyCassette1"

func runTestEx1() {
	// Create vcr and make http call
	vcr := govcr.NewVCR(example1CassetteName, nil)
	resp, _ := vcr.Client.Get("http://example.com/foo")

	// Show results
	fmt.Printf("%d ", resp.StatusCode)
	fmt.Printf("%s ", resp.Header.Get("Content-Type"))

	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Printf("%v ", strings.Contains(string(body), "domain in examples without prior coordination or asking for permission."))

	fmt.Printf("%+v\n", vcr.Stats())
}

// Example_simpleVCR is an example use of govcr.
// It shows how to use govcr in the simplest case when the default
// http.Client suffices.
func Example_Numer1simpleVCR() {
	// Delete cassette to enable live HTTP call
	govcr.DeleteCassette(example1CassetteName)

	// 1st run of the test - will use live HTTP calls
	runTestEx1()
	// 2nd run of the test - will use playback
	runTestEx1()

	// Output:
	// 404 text/html true {TracksLoaded:0 TracksRecorded:1 TracksPlayed:0}
	// 404 text/html true {TracksLoaded:1 TracksRecorded:0 TracksPlayed:1}
}
