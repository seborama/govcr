package main

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"time"

	"strings"

	"github.com/seborama/govcr"
)

func (app myApp) Post(url string) {
	body := ioutil.NopCloser(strings.NewReader(`{"Msg": "This is an example request"}`))
	app.httpClient.Post(url, "application/json", body)
}

// Example3 is an example use of govcr.
func Example3() {
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
	vcr := govcr.NewVCR("MyCassette3",
		&govcr.VCRConfig{
			Client: myapp.httpClient,
		})

	// Inject VCR's http.Client wrapper.
	// The original transport has been preserved, only just wrapped into VCR's.
	myapp.httpClient = vcr.Client

	myapp.Post("https://example.com/foo")
}
