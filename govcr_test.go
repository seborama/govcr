package govcr_test

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/seborama/govcr"
)

func TestRecordGetRequest(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com/foo", nil)
	if err != nil {
		log.Fatal(err)
	}

	filename := "/tmp/govcr/fixtures/TestRecordGetRequest.rec"
	w, err := govcr.Record(req, filename)
	if err != nil {
		t.Fatalf("err from govcr.Record(): Expected nil, got %s", err)
	}

	if w.Code != http.StatusOK {
		t.Fatalf("w.Code: Expected %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.Len() < 1 {
		t.Fatalf("Body length: Expected >= 1, got %d", w.Body.Len())
	}

	fi, err := os.Stat(filename)
	if os.IsNotExist(err) {
		t.Fatalf("'" + filename + "' does not exist")
	}
	if fi.Size() < 2 {
		t.Fatalf("fi.Size(): Expected >= 2, got %d", fi.Size())
	}
}

func TestReplayGetRequest(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com/foo", nil)
	if err != nil {
		log.Fatal(err)
	}

	filename := "/tmp/govcr/fixtures/TestRecordGetRequest.rec"
	w, err := govcr.Replay(req, filename)
	if err != nil {
		t.Fatalf("err from govcr.Record(): Expected nil, got %s", err)
	}

	if w.Code != http.StatusOK {
		t.Fatalf("w.Code: Expected %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.Len() < 1 {
		t.Fatalf("Body length: Expected >= 1, got %d", w.Body.Len())
	}
}

func TestRecordClientGetRequest(t *testing.T) {
	c := govcr.GetVCR("TestRecordClientGetRequest")

	resp, err := c.Get("http://example.com/foo")
	if err != nil {
		t.Fatalf("err from c.Get(): Expected nil, got %s", err)
	}

	body := ioutil.NopCloser(resp.Body)
	bodyData, err := ioutil.ReadAll(body)
	if err != nil {
		t.Fatalf("err from ioutil.ReadAll(): Expected nil, got %s", err)
	}
	log.Printf("DEBUG - bodyData=%s\n", bodyData)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("w.Code: Expected %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if l := len(bodyData); l < 1 {
		t.Fatalf("Body length: Expected >= 1, got %d", l)
	}
}
