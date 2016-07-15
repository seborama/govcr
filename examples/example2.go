package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/seborama/govcr"
)

const example2CassetteName = "MyCassette2"

// myApp is an application container.
type myApp struct {
	httpClient *http.Client
}

func (app myApp) Get(url string) {
	resp, err := app.httpClient.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(resp)
}

// Example2 is an example use of govcr.
// It shows the use of a VCR with a custom Client.
// Here, the app executes a GET request.
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
	vcr := govcr.NewVCR(example2CassetteName,
		&govcr.VCRConfig{
			Client: myapp.httpClient,
		})

	// Inject VCR's http.Client wrapper.
	// The original transport has been preserved, only just wrapped into VCR's.
	myapp.httpClient = vcr.Client

	myapp.Get("https://example.com/foo")
}
