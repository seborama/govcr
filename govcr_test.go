package govcr_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/seborama/govcr"
)

// TODO: re-write table test to include more HTTP verbs and payloads
func TestRecordClientGetRequest(t *testing.T) {
	cassetteName := "TestRecordClientGetRequest"

	if err := govcr.DeleteCassette(cassetteName); err != nil && !os.IsNotExist(err) {
		t.Fatalf("err from govcr.DeleteCassette(): Expected nil, got %s", err)
	}

	vcr := govcr.StartVCR(cassetteName)
	defer vcr.StopVCRFunc()

	client := vcr.Client

	resp, err := client.Get("http://example.com/foo")
	if err != nil {
		t.Fatalf("err from c.Get(): Expected nil, got %s", err)
	}

	if resp.StatusCode != http.StatusOK {
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
		t.Fatalf("Body contains string: Expected true, got false")
	}

	if !govcr.CassetteExists(cassetteName) {
		t.Fatalf("CassetteExists: expected true, got false")
	}
}
