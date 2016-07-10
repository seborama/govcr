package govcr_test

import (
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	"github.com/seborama/govcr"
)

func TestRecordClientGetRequest(t *testing.T) {
	client := govcr.StartVCR("TestRecordClientGetRequest")

	resp, err := client.Get("http://example.com/foo")
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
