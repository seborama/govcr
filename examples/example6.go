package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/seborama/govcr"
)

const example6CassetteName = "MyCassette6"

// Example6 is an example use of govcr.
// This will show how to do conditional rewrites.
// For example, your request has a "/order/{random}" path
// and we want to rewrite it to /order/1234 so we can match it later.
// We change the response status code.
// We add headers based on request method.
func Example6() {
	cfg := govcr.VCRConfig{
		Logging: true,
	}

	// The filter will neutralize a value in the URL.
	// In this case we rewrite /order/{random} to /order/1234
	replacePath := govcr.RequestFilter(func(req govcr.Request) govcr.Request {
		// Replace path with a predictable one.
		req.URL.Path = "/order/1234"
		return req
	})
	// Only execute when we match path.
	replacePath = replacePath.OnPath(`example\.com\/order\/`)

	// Add to request filters.
	cfg.RequestFilters.Add(replacePath)
	cfg.RequestFilters.Add(govcr.RequestDeleteHeaderKeys("X-Transaction-Id"))

	// Add filters to
	cfg.ResponseFilters.Add(
		// Always transfer 'X-Transaction-Id' as in example 5.
		govcr.ResponseTransferHeaderKeys("X-Transaction-Id"),

		// Change status 404 to 202.
		func(resp govcr.Response) govcr.Response {
			if resp.StatusCode == http.StatusNotFound {
				resp.StatusCode = http.StatusAccepted
			}
			return resp
		},

		// Add header if method was "GET"
		govcr.ResponseFilter(func(resp govcr.Response) govcr.Response {
			resp.Header.Add("method-was-get", "true")
			return resp
		}).OnMethod(http.MethodGet),

		// Add header if method was "POST"
		govcr.ResponseFilter(func(resp govcr.Response) govcr.Response {
			resp.Header.Add("method-was-post", "true")
			return resp
		}).OnMethod(http.MethodPost),

		// Add actual request URL to header.
		govcr.ResponseFilter(func(resp govcr.Response) govcr.Response {
			url := resp.Request().URL
			resp.Header.Add("get-url", url.String())
			return resp
		}).OnMethod(http.MethodGet),
	)

	orderID := fmt.Sprint(rand.Int63())
	vcr := govcr.NewVCR(example6CassetteName, &cfg)

	// create a request with our custom header and a random url part.
	req, err := http.NewRequest("POST", "http://www.example.com/order/"+orderID, nil)
	if err != nil {
		fmt.Println(err)
	}
	runRequest(req, err, vcr)

	// create a request with our custom header and a random url part.
	req, err = http.NewRequest("GET", "http://www.example.com/order/"+orderID, nil)
	if err != nil {
		fmt.Println(err)
	}
	runRequest(req, err, vcr)

}

func runRequest(req *http.Request, err error, vcr *govcr.VCRControlPanel) {
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

	// print outcome.
	fmt.Println("Status code:", resp.StatusCode, " (should be 404 on real and 202 on replay)")
	fmt.Println("method-was-get:", resp.Header.Get("method-was-get"), "(should never be true on GET)")
	fmt.Println("method-was-post:", resp.Header.Get("method-was-post"), "(should be true on replay on POST)")
	fmt.Println("get-url:", resp.Header.Get("get-url"), "(actual url of the request, not of the track)")
	fmt.Printf("%+v\n", vcr.Stats())
}
