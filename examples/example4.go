package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/seborama/govcr"
)

const example4CassetteName = "MyCassette4"

// Example4 is an example use of govcr.
// The request contains a customer header 'X-Custom-My-Date' which varies with every request.
// This example shows how to exclude a particular header from the request to facilitate
// matching a previous recording.
// Without the ExcludeHeaderFunc, the headers would not match and hence the playback would not
// happen!
func Example4() {
	vcr := govcr.NewVCR(example4CassetteName,
		&govcr.VCRConfig{
			RequestFilters: govcr.RequestFilters{
				govcr.RequestDeleteHeaderKeys("X-Transaction-Id"),
			},
			Logging: true,
		})

	// create a request with our custom header
	req, err := http.NewRequest("POST", "http://www.example.com/foo", nil)
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Add("X-Custom-My-Date", time.Now().String())

	// run the request
	vcr.Client.Do(req)
	fmt.Printf("%+v\n", vcr.Stats())
}
