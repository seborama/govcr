package govcr_test

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"net/http/httptest"

	"github.com/seborama/govcr"
)

const (
	wipeCassette = true
	keepCassette = false
)

func TestPlaybackOrder(t *testing.T) {
	cassetteName := "TestPlaybackOrder"
	clientNum := 1

	// create a test server
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, client %d", clientNum)
		clientNum++
	}))

	fmt.Println("Phase 1 ================================================")

	if err := govcr.DeleteCassette(cassetteName, ""); err != nil {
		t.Fatalf("err from govcr.DeleteCassette(): Expected nil, got %s", err)
	}

	vcr := createVCR(cassetteName, wipeCassette)
	client := vcr.Client

	// run requests
	for i := 1; i <= 10; i++ {
		resp, _ := client.Get(ts.URL)

		// check outcome of the request
		checkResponseForTestPlaybackOrder(t, cassetteName, vcr, resp, i, false)

		if !govcr.CassetteExistsAndValid(cassetteName, "") {
			t.Fatalf("CassetteExists: expected true, got false")
		}

		checkStats(t, vcr.Stats(), 0, i, 0)
	}

	fmt.Println("Phase 2 ================================================")
	clientNum = 1

	// re-run request and expect play back from vcr
	vcr = createVCR(cassetteName, keepCassette)
	client = vcr.Client

	// run requests
	for i := 1; i <= 10; i++ {
		resp, _ := client.Get(ts.URL)

		// check outcome of the request
		checkResponseForTestPlaybackOrder(t, cassetteName, vcr, resp, i, false)

		if !govcr.CassetteExistsAndValid(cassetteName, "") {
			t.Fatalf("CassetteExists: expected true, got false")
		}

		checkStats(t, vcr.Stats(), 10, 0, i)
	}
}

func TestNonUtf8EncodableBinaryBody(t *testing.T) {
	cassetteName := "TestNonUtf8EncodableBinaryBody"
	clientNum := 1

	// create a test server
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := generateBinaryBody(clientNum)
		written, err := w.Write(data)
		if written != len(data) {
			t.Fatalf("** Only %d bytes out of %d were written", written, len(data))
		}
		if err != nil {
			t.Fatalf("err from w.Write(): Expected nil, got %s", err)
		}
		clientNum++
	}))

	fmt.Println("Phase 1 ================================================")

	if err := govcr.DeleteCassette(cassetteName, ""); err != nil {
		t.Fatalf("err from govcr.DeleteCassette(): Expected nil, got %s", err)
	}

	vcr := createVCR(cassetteName, wipeCassette)
	client := vcr.Client

	// run requests
	for i := 1; i <= 10; i++ {
		resp, _ := client.Get(ts.URL)

		// check outcome of the request
		checkResponseForTestPlaybackOrder(t, cassetteName, vcr, resp, i, true)

		if !govcr.CassetteExistsAndValid(cassetteName, "") {
			t.Fatalf("CassetteExists: expected true, got false")
		}

		checkStats(t, vcr.Stats(), 0, i, 0)
	}

	fmt.Println("Phase 2 ================================================")
	clientNum = 1

	// re-run request and expect play back from vcr
	vcr = createVCR(cassetteName, keepCassette)
	client = vcr.Client

	// run requests
	for i := 1; i <= 10; i++ {
		resp, _ := client.Get(ts.URL)

		// check outcome of the request
		checkResponseForTestPlaybackOrder(t, cassetteName, vcr, resp, i, true)

		if !govcr.CassetteExistsAndValid(cassetteName, "") {
			t.Fatalf("CassetteExists: expected true, got false")
		}

		checkStats(t, vcr.Stats(), 10, 0, i)
	}
}

func createVCR(cassetteName string, wipeCassette bool) *govcr.VCRControlPanel {
	// create a custom http.Transport.
	tr := http.DefaultTransport.(*http.Transport)
	tr.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true, // just an example, not recommended
	}

	// create a vcr
	return govcr.NewVCR(cassetteName,
		&govcr.VCRConfig{
			Client: &http.Client{Transport: tr},
		})
}

func checkResponseForTestPlaybackOrder(t *testing.T, cassetteName string, vcr *govcr.VCRControlPanel, resp *http.Response, i int, binary bool) {
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("resp.StatusCode: Expected %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if resp.Body == nil {
		t.Fatalf("resp.Body: Expected non-nil, got nil")
	}

	bodyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("err from ioutil.ReadAll(): Expected nil, got %s", err)
	}
	resp.Body.Close()

	if binary {
		expectedBody := generateBinaryBody(i)
		if bytes.Compare(bodyData, expectedBody) != 0 {
			t.Fatalf("Body: expected '%v', got '%v'", expectedBody, bodyData)
		}
	} else {
		expectedBody := fmt.Sprintf("Hello, client %d", i)
		if string(bodyData) != expectedBody {
			t.Fatalf("Body: expected '%s', got '%s'", expectedBody, string(bodyData))
		}
	}
}

func checkStats(t *testing.T, actualStats govcr.Stats, expectedTracksLoaded, expectedTracksRecorded, expectedTrackPlayed int) {
	if actualStats.TracksLoaded != expectedTracksLoaded {
		t.Fatalf("Expected %d track loaded, got %d", expectedTracksLoaded, actualStats.TracksLoaded)
	}

	if actualStats.TracksRecorded != expectedTracksRecorded {
		t.Fatalf("Expected %d track recorded, got %d", expectedTracksRecorded, actualStats.TracksRecorded)
	}

	if actualStats.TracksPlayed != expectedTrackPlayed {
		t.Fatalf("Expected %d track played, got %d", expectedTrackPlayed, actualStats.TracksPlayed)
	}
}

func generateBinaryBody(sequence int) []byte {
	data := make([]byte, 256, 257)
	for i := range data {
		data[i] = byte(i)
	}
	data = append(data, byte(sequence))
	return data
}
