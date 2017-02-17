package main

import (
	"fmt"
	"strings"
	"time"

	"net/http"

	"github.com/seborama/govcr"
)

const example5CassetteName = "MyCassette5"

// Example5 is an example use of govcr.
// Supposing a fictional application where the request contains a custom header
// 'X-Transaction-Id' which must be matched in the response from the server.
// When replaying, the request will have a different Transaction Id than that which was recorded.
// Hence the protocol (of this fictional example) is broken.
// To circumvent that, we inject the new request's X-Transaction-Id into the recorded response.
// Without the ResponseFilterFunc, the X-Transaction-Id in the header would not match that
// of the recorded response and our fictional application would reject the response on validation!
func Example5() {
	vcr := govcr.NewVCR(example5CassetteName,
		&govcr.VCRConfig{
			ExcludeHeaderFunc: func(key string) bool {
				// ignore the X-Transaction-Id since it changes per-request
				return strings.ToLower(key) == "x-transaction-id"
			},
			ResponseFilterFunc: func(respHeader http.Header, respBody []byte, reqHeader http.Header) (*http.Header, *[]byte) {
				// overwrite X-Transaction-Id in the Response with that from the Request
				respHeader.Set("X-Transaction-Id", reqHeader.Get("X-Transaction-Id"))

				return &respHeader, &respBody
			},
			Logging: true,
		})

	// create a request with our custom header
	req, err := http.NewRequest("POST", "http://example.com/foo5", nil)
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Add("X-Transaction-Id", time.Now().String())

	// run the request
	resp, err := vcr.Client.Do(req)
	if err != nil {
		fmt.Println(err)
	}

	// verify outcome
	if req.Header.Get("X-Transaction-Id") != resp.Header.Get("X-Transaction-Id") {
		fmt.Println("Header transaction Id verification failed - this would be the live request!")
	} else {
		fmt.Println("Header transaction Id verification passed - this would be the replayed track!")
	}

	fmt.Printf("%+v\n", vcr.Stats())
}
