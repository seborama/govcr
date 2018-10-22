package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"net/http"

	"github.com/seborama/govcr"
)

const example6CassetteName = "MyCassette6"

// Example5 is an example use of govcr.
// Supposing a fictional application where the request contains a custom header
// 'X-Transaction-Id' which must be matched in the response from the server.
// When replaying, the request will have a different Transaction Id than that which was recorded.
// Hence the protocol (of this fictional example) is broken.
// To circumvent that, we inject the new request's X-Transaction-Id into the recorded response.
// Without the ResponseFilterFunc, the X-Transaction-Id in the header would not match that
// of the recorded response and our fictional application would reject the response on validation!
func Example6() {
	cfg := govcr.VCRConfig{
		ExcludeHeaderFunc: func(key string) bool {
			// ignore the X-Transaction-Id since it changes per-request
			return strings.ToLower(key) == "x-transaction-id"
		},
		Logging: true,
	}

	// RequestFilter will neutralize a value in the URL.
	cfg.RequestFilter = func(req govcr.Request) govcr.Request {
		// Replace path with a predictable one.
		req.URL.Path = "/foo6/1234"
		return req
	}

	// Only execute POST and path contains www.example.com/order/
	cfg.RequestFilter.OnMethod(http.MethodPost).OnPath(`example\.com\/order\/`)

	cfg.ResponseFilter = cfg.ResponseFilter.TransferHeaderKeys("X-Transaction-Id").
		// Chain independent functions.
		Chain(
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

			// Add request URL to header.
			govcr.ResponseFilter(func(resp govcr.Response) govcr.Response {
				url := resp.Request().URL
				resp.Header.Add("get-url", url.String())
				return resp
			}),
		)

	vcr := govcr.NewVCR(example6CassetteName, &cfg)

	// create a request with our custom header and a random url part.
	req, err := http.NewRequest("POST", "http://www.example.com/order/"+fmt.Sprint(rand.Int63()), nil)
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

	// print outcome.
	fmt.Println("Status code:", resp.StatusCode, " (should be 404 on real and 202 on replay)")
	fmt.Println("method-was-get:", resp.Header.Get("method-was-get"), "(should never be true)")
	fmt.Println("method-was-post:", resp.Header.Get("method-was-post"), "(should be true on replay)")
	fmt.Println("get-url:", resp.Header.Get("get-url"), "(should be http://www.example.com/order/1234 on replay)")

	fmt.Printf("%+v\n", vcr.Stats())
}
