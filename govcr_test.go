package govcr_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/seborama/govcr"
)

// TODO: re-write with table test to include more HTTP verbs and payloads
func TestRecordClientGetRequest(t *testing.T) {
	cassetteName := "TestRecordClientGetRequest"

	// wipe cassette clear
	if err := govcr.DeleteCassette(cassetteName); err != nil && !os.IsNotExist(err) {
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
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("resp.StatusCode: Expected %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if resp.Body == nil {
		t.Fatalf("resp.Body: Expected non-nil, got nil")
	}

	body := ioutil.NopCloser(resp.Body)
	bodyData, err := ioutil.ReadAll(body)
	if err != nil {
		t.Fatalf("err from ioutil.ReadAll(): Expected nil, got %s", err)
	}

	if !strings.Contains(string(bodyData), "Example Domain") {
		t.Fatalf("Body does not contain the expected string")
	}

	if !govcr.CassetteExists(cassetteName) {
		t.Fatalf("CassetteExists: expected true, got false")
	}

	if vcr.Stats().TracksLoaded != 0 {
		t.Fatalf("Expected 0 track loaded, got %d", vcr.Stats().TracksRecorded)
	}

	if vcr.Stats().TracksRecorded != 1 {
		t.Fatalf("Expected 1 track recorded, got %d", vcr.Stats().TracksRecorded)
	}

	if vcr.Stats().TracksPlayed != 0 {
		t.Fatalf("Expected 0 track played, got %d", vcr.Stats().TracksRecorded)
	}

	// TODO: add a test to confirm that all track are marked as replyed

	// re-run request and expect play back from vcr
	vcr = govcr.NewVCR(cassetteName, nil)
	client = vcr.Client

	resp, err = client.Get("http://example.com/foo")
	if err != nil {
		t.Fatalf("err from c.Get(): Expected nil, got %s", err)
	}

	if vcr.Stats().TracksLoaded != 1 {
		t.Fatalf("Expected 1 track loaded, got %d", vcr.Stats().TracksRecorded)
	}

	if vcr.Stats().TracksRecorded != 0 {
		t.Fatalf("Expected 0 track recorded, got %d", vcr.Stats().TracksRecorded)
	}

	if vcr.Stats().TracksPlayed != 1 {
		t.Fatalf("Expected 1 track played, got %d", vcr.Stats().TracksRecorded)
	}
}
