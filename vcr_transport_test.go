package govcr

import (
	"log"
	"net/http"
	"os"
	"reflect"
	"testing"
)

type mockRoundTripper struct{}

func (t *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Request:    req,
		StatusCode: http.StatusMovedPermanently,
	}, nil
}

func Test_vcrTransport_RoundTrip_doesNotChangeLiveReqOrLiveResp(t *testing.T) {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	out, err := os.OpenFile(os.DevNull, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Errorf("unable to initialise logger - error = %v", err)
		return
	}
	defer func() { out.Close() }()
	logger.SetOutput(out)

	mutateReq := RequestFilter(func(req Request) Request {
		req.Method = "INVALID"
		req.URL.Host = "host.changed.internal"
		return req
	})
	requestFilters := RequestFilters{}
	requestFilters.Add(mutateReq)

	mutateResp := ResponseFilter(func(resp Response) Response {
		resp.StatusCode = -9999
		return resp
	})
	responseFilters := ResponseFilters{}
	responseFilters.Add(mutateResp)

	mrt := &mockRoundTripper{}
	transport := &vcrTransport{
		PCB: &pcb{
			DisableRecording: true,
			Transport:        mrt,
			RequestFilter:    requestFilters.combined(),
			ResponseFilter:   responseFilters.combined(),
			Logger:           logger,
			CassettePath:     "",
		},
		Cassette: &cassette{},
	}

	req, err := http.NewRequest("GET", "https://example.com/path?query", toReadCloser([]byte("Lorem ipsum dolor sit amet")))
	if err != nil {
		t.Errorf("req http.NewRequest() error = %v", err)
		return
	}

	wantReq, err := http.NewRequest("GET", "https://example.com/path?query", toReadCloser([]byte("Lorem ipsum dolor sit amet")))
	if err != nil {
		t.Errorf("wantReq http.NewRequest() error = %v", err)
		return
	}

	gotResp, err := transport.RoundTrip(req)
	if err != nil {
		t.Errorf("vcrTransport.RoundTrip() error = %v", err)
		return
	}
	wantResp := http.Response{
		Request:    wantReq,
		StatusCode: http.StatusMovedPermanently,
	}

	if !reflect.DeepEqual(req, wantReq) {
		t.Errorf("vcrTransport.RoundTrip() Request has been modified = %+v, want %+v", req, wantReq)
	}

	if !reflect.DeepEqual(gotResp, &wantResp) {
		t.Errorf("vcrTransport.RoundTrip() Response has been modified = %+v, want %+v", gotResp, wantResp)
	}
}
