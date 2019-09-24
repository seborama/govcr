package govcr_test

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/seborama/govcr"
)

const example8CassetteName = "MyCassette8"

func runTestEx8() {
	// Create vcr and make http call

	cfg := govcr.VCRConfig{
		Logging: true,
		SkipErrorCodes: true,
	}
	
	vcr := govcr.NewVCR(example8CassetteName, &cfg)
	resp, _ := vcr.Client.Get("http://www.example.com/foo")

	// Show results
	fmt.Printf("%d ", resp.StatusCode)
	fmt.Printf("%s ", resp.Header.Get("Content-Type"))
	fmt.Printf("%+v\n", vcr.Stats())
}

// Example_number8SkipErrorCodes is an example of the use of the SkipErrorCodes functionality.
// It shows how to use govcr by adding SkipErrorCodes to skip saving tracks if an error is returned ( status code: 404, 403 etc.. )

func Example_number8SkipErrorCodes() {
	// Delete cassette to enable live HTTP call
	govcr.DeleteCassette(example8CassetteName, "")

	// 1st run of the test - will use live HTTP call - Will not save the track due to 404.
	runTestEx8()
	// 2nd run of the test - will use live HTTP call as well - Will not save the track again due to 404.
	runTestEx8()

	// Output:
	// 404 text/html; charset=UTF-8 true {TracksLoaded:0 TracksRecorded:0 TracksPlayed:0}
	// 404 text/html; charset=UTF-8 true {TracksLoaded:0 TracksRecorded:0 TracksPlayed:0}
}