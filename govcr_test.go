package govcr_test

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"testing"

	"net/http/httptest"

	"github.com/seborama/govcr"
)

// TODO: re-write with table test to include more HTTP verbs and payloads
func TestRecordClientGetRequest(t *testing.T) {
	cassetteName := "TestRecordClientGetRequest"

	fmt.Println("Phase 1 ================================================")

	// wipe cassette clear
	if err := govcr.DeleteCassette(cassetteName); err != nil {
		t.Fatalf("err from govcr.DeleteCassette(): Expected nil, got %s", err)
	}

	// create a vcr
	vcr := govcr.NewVCR(cassetteName, nil)
	client := vcr.Client

	// run request
	resp, err := client.Get("http://example.com/foo")
	if err != nil {
		t.Fatalf("err from c.Get(): Expected nil, got %s", err)
	}

	// check outcome of the request
	checkResponseForTestRecordClientGetRequest(t, cassetteName, vcr, resp)

	if vcr.Stats().TracksLoaded != 0 {
		t.Fatalf("Expected 0 track loaded, got %d", vcr.Stats().TracksLoaded)
	}

	if vcr.Stats().TracksRecorded != 1 {
		t.Fatalf("Expected 1 track recorded, got %d", vcr.Stats().TracksRecorded)
	}

	if vcr.Stats().TracksPlayed != 0 {
		t.Fatalf("Expected 0 track played, got %d", vcr.Stats().TracksPlayed)
	}

	fmt.Println("Phase 2 ================================================")

	// re-run request and expect play back from vcr
	vcr = govcr.NewVCR(cassetteName, nil)
	client = vcr.Client

	resp, err = client.Get("http://example.com/foo")
	if err != nil {
		t.Fatalf("err from c.Get(): Expected nil, got %s", err)
	}

	// check outcome of the request
	checkResponseForTestRecordClientGetRequest(t, cassetteName, vcr, resp)

	if vcr.Stats().TracksLoaded != 1 {
		t.Fatalf("Expected 1 track loaded, got %d", vcr.Stats().TracksLoaded)
	}

	if vcr.Stats().TracksRecorded != 0 {
		t.Fatalf("Expected 0 track recorded, got %d", vcr.Stats().TracksRecorded)
	}

	if vcr.Stats().TracksPlayed != 1 {
		t.Fatalf("Expected 1 track played, got %d", vcr.Stats().TracksPlayed)
	}
}

func TestPlaybackOrder(t *testing.T) {
	cassetteName := "TestPlaybackOrder"
	clientNum := 1

	// create a test server
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, client %d", clientNum)
		clientNum++
	}))

	fmt.Println("Phase 1 ================================================")

	// wipe cassette clear
	if err := govcr.DeleteCassette(cassetteName); err != nil {
		t.Fatalf("err from govcr.DeleteCassette(): Expected nil, got %s", err)
	}

	// create a custom http.Transport.
	tr := http.DefaultTransport.(*http.Transport)
	tr.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true, // just an example, not recommended
	}

	// create a vcr
	vcr := govcr.NewVCR(cassetteName,
		&govcr.PCB{
			Transport: tr,
		})
	client := vcr.Client

	// run requests
	for i := 1; i <= 10; i++ {
		log.Printf("i=%d\n", i)
		resp, err := client.Get(ts.URL)
		if err != nil {
			t.Fatalf("err from c.Get(): Expected nil, got %s", err)
		}

		// check outcome of the request
		checkResponseForTestPlaybackOrder(t, cassetteName, vcr, resp, i)

		if !govcr.CassetteExists(cassetteName) {
			t.Fatalf("CassetteExists: expected true, got false")
		}

		if vcr.Stats().TracksLoaded != 0 {
			t.Fatalf("Expected 0 track loaded, got %d", vcr.Stats().TracksLoaded)
		}

		if vcr.Stats().TracksRecorded != i {
			t.Fatalf("Expected %d track(s) recorded, got %d", i, vcr.Stats().TracksRecorded)
		}

		if vcr.Stats().TracksPlayed != 0 {
			t.Fatalf("Expected 0 track played, got %d", vcr.Stats().TracksPlayed)
		}
	}

	// TODO: add a test to confirm that all track are marked as replyed

	fmt.Println("Phase 2 ================================================")
	clientNum = 1

	// re-run request and expect play back from vcr
	vcr = govcr.NewVCR(cassetteName,
		&govcr.PCB{
			Transport: tr,
		})
	client = vcr.Client

	// run requests
	for i := 1; i <= 10; i++ {
		resp, err := client.Get(ts.URL)
		if err != nil {
			t.Fatalf("err from c.Get(): Expected nil, got %s", err)
		}

		// check outcome of the request
		checkResponseForTestPlaybackOrder(t, cassetteName, vcr, resp, i)

		if vcr.Stats().TracksLoaded != 10 {
			t.Fatalf("Expected 10 tracks loaded, got %d", vcr.Stats().TracksLoaded)
		}

		if vcr.Stats().TracksRecorded != 0 {
			t.Fatalf("Expected 0 track recorded, got %d", vcr.Stats().TracksRecorded)
		}

		if vcr.Stats().TracksPlayed != i {
			t.Fatalf("Expected %d track(s) played, got %d", i, vcr.Stats().TracksPlayed)
		}
	}
}

func checkResponseForTestRecordClientGetRequest(t *testing.T, cassetteName string, vcr *govcr.VCRControlPannel, resp *http.Response) {
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("resp.StatusCode: Expected %d, got %d", http.StatusNotFound, resp.StatusCode)
	}

	if resp.Body == nil {
		t.Fatalf("resp.Body: Expected non-nil, got nil")
	}

	bodyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("err from ioutil.ReadAll(): Expected nil, got %s", err)
	}
	resp.Body.Close()

	if !strings.Contains(string(bodyData), "Example Domain") {
		t.Fatalf("Body does not contain the expected string")
	}

	if !govcr.CassetteExists(cassetteName) {
		t.Fatalf("CassetteExists: expected true, got false")
	}

	// TODO: add a test to confirm that all track are marked as replyed
}

func checkResponseForTestPlaybackOrder(t *testing.T, cassetteName string, vcr *govcr.VCRControlPannel, resp *http.Response, i int) {
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

	expectedBody := fmt.Sprintf("Hello, client %d", i)
	if string(bodyData) != expectedBody {
		t.Fatalf("Body: expected '%s', got '%s'", expectedBody, string(bodyData))
	}

	// TODO: add a test to confirm that all track are marked as replyed
}
