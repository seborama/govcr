package govcr_test

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/seborama/govcr"
)

const example2CassetteName = "MyCassette2"

// myApp is an application container.
type myApp struct {
	httpClient *http.Client
}

func (app *myApp) Get(url string) (*http.Response, error) {
	return app.httpClient.Get(url)
}

func (app *myApp) Post(url string) (*http.Response, error) {
	// beware: don't use a ReadCloser, only a Reader!
	body := strings.NewReader(`{"Msg": "This is an example request"}`)
	return app.httpClient.Post(url, "application/json", body)
}

func runTestEx2(app *myApp) {
	var samples = []struct {
		f    func(string) (*http.Response, error)
		body string
	}{
		{app.Get, "domain in examples without prior coordination or asking for permission."},
		{app.Post, "404 - Not Found"},
	}

	// Instantiate VCR.
	vcr := govcr.NewVCR(example2CassetteName,
		&govcr.VCRConfig{
			Client: app.httpClient,
		})

	// Inject VCR's http.Client wrapper.
	// The original transport has been preserved, only just wrapped into VCR's.
	app.httpClient = vcr.Client

	for _, td := range samples {
		// Run HTTP call
		resp, _ := td.f("https://example.com/foo")

		// Show results
		fmt.Printf("%d ", resp.StatusCode)
		fmt.Printf("%s ", resp.Header.Get("Content-Type"))

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("%v - ", strings.Contains(string(body), td.body))
	}

	fmt.Printf("%+v\n", vcr.Stats())
}

// Example2 is an example use of govcr.
// It shows the use of a VCR with a custom Client.
// Here, the app executes a GET request.
func Example_number2CustomClientVCR1() {
	// Create a custom http.Transport.
	tr := http.DefaultTransport.(*http.Transport)
	tr.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true, // just an example, not recommended
	}

	// Create an instance of myApp.
	// It uses the custom Transport created above and a custom Timeout.
	app := &myApp{
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   15 * time.Second,
		},
	}

	// Delete cassette to enable live HTTP call
	govcr.DeleteCassette(example2CassetteName)

	// 1st run of the test - will use live HTTP calls
	runTestEx2(app)
	// 2nd run of the test - will use playback
	runTestEx2(app)

	// Output:
	// 404 text/html true - 404 text/html true - {TracksLoaded:0 TracksRecorded:2 TracksPlayed:0}
	// 404 text/html true - 404 text/html true - {TracksLoaded:2 TracksRecorded:0 TracksPlayed:2}
}
