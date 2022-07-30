package examples_test

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/seborama/govcr/v5"
)

const exampleCassetteName2 = "temp-fixtures/TestExample2.cassette.json"

// imaginary app
type myApp struct {
	httpClient *http.Client
}

func (app myApp) Get(url string) {
	app.httpClient.Get(url)
}

// TestExample2 is an example use of govcr.
func TestExample2(t *testing.T) {
	// Create a custom http.Transport for our app.
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

	// Instantiate VCR.
	vcr := govcr.NewVCR(
		govcr.WithCassette(exampleCassetteName2),
		govcr.WithClient(app.httpClient),
	)

	// Inject VCR's http.Client wrapper.
	// The original transport has been preserved, only just wrapped into VCR's.
	app.httpClient = vcr.HTTPClient()

	// Run request and display stats.
	app.Get("https://example.com/foo")
	t.Logf("%+v\n", vcr.Stats())
}
