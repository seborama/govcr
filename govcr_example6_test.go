package govcr_test

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
func runTestEx6(rng *rand.Rand) {
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

	// Add filters
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

	orderID := fmt.Sprint(rng.Uint64())
	vcr := govcr.NewVCR(example6CassetteName, &cfg)

	// create a request with our custom header and a random url part.
	req, err := http.NewRequest("POST", "http://www.example.com/order/"+orderID, nil)
	if err != nil {
		fmt.Println(err)
	}
	runExample6Request(req, vcr)

	// create a request with our custom header and a random url part.
	req, err = http.NewRequest("GET", "http://www.example.com/order/"+orderID, nil)
	if err != nil {
		fmt.Println(err)
	}
	runExample6Request(req, vcr)

}

func runExample6Request(req *http.Request, vcr *govcr.VCRControlPanel) {
	req.Header.Add("X-Transaction-Id", time.Now().String())
	// run the request
	resp, err := vcr.Client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	// verify outcome
	if req.Header.Get("X-Transaction-Id") != resp.Header.Get("X-Transaction-Id") {
		fmt.Println("Header transaction Id verification FAILED - this would be the live request!")
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

// Example_simpleVCR is an example use of govcr.
// It shows how to use govcr in the simplest case when the default
// http.Client suffices.
func Example_number6ConditionalRewrites() {
	// Delete cassette to enable live HTTP call
	govcr.DeleteCassette(example6CassetteName, "")

	// We need a predictable RNG
	rng := rand.New(rand.NewSource(6))

	// 1st run of the test - will use live HTTP calls
	runTestEx6(rng)
	// 2nd run of the test - will use playback
	runTestEx6(rng)

	// Output:
	//Header transaction Id verification FAILED - this would be the live request!
	//Status code: 404  (should be 404 on real and 202 on replay)
	//method-was-get:  (should never be true on GET)
	//method-was-post:  (should be true on replay on POST)
	//get-url:  (actual url of the request, not of the track)
	//{TracksLoaded:0 TracksRecorded:1 TracksPlayed:0}
	//Header transaction Id verification FAILED - this would be the live request!
	//Status code: 404  (should be 404 on real and 202 on replay)
	//method-was-get:  (should never be true on GET)
	//method-was-post:  (should be true on replay on POST)
	//get-url:  (actual url of the request, not of the track)
	//{TracksLoaded:0 TracksRecorded:2 TracksPlayed:0}
	//Header transaction Id verification passed - this would be the replayed track!
	//Status code: 202  (should be 404 on real and 202 on replay)
	//method-was-get:  (should never be true on GET)
	//method-was-post: true (should be true on replay on POST)
	//get-url:  (actual url of the request, not of the track)
	//{TracksLoaded:2 TracksRecorded:0 TracksPlayed:1}
	//Header transaction Id verification passed - this would be the replayed track!
	//Status code: 202  (should be 404 on real and 202 on replay)
	//method-was-get: true (should never be true on GET)
	//method-was-post:  (should be true on replay on POST)
	//get-url: http://www.example.com/order/7790075977082629872 (actual url of the request, not of the track)
	//{TracksLoaded:2 TracksRecorded:0 TracksPlayed:2}
}
