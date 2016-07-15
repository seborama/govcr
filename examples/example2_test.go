package examples_test

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/seborama/govcr"
)

// myApp is an application container.
type myApp struct {
	httpClient *http.Client
}

func (app myApp) Get(url string) {
	app.httpClient.Get(url)
}

// Example2 is an example use of govcr.
func Example2() {
	// Create a custom http.Transport.
	tr := http.DefaultTransport.(*http.Transport)
	tr.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true, // just an example, not recommended
	}

	// Create an instance of myApp.
	// It uses the custom Transport created above and a custom Timeout.
	myapp := &myApp{
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   15 * time.Second,
		},
	}

	// Instantiate VCR.
	vcr := govcr.NewVCR("MyCassette2",
		&govcr.VCRConfig{
			Client: myapp.httpClient,
		})

	// Inject VCR's http.Client wrapper.
	// The original transport has been preserved, only just wrapped into VCR's.
	myapp.httpClient = vcr.Client

	myapp.Get("https://example.com/foo")
}
