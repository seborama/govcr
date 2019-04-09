package govcr

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"reflect"
	"testing"
)

type mockRoundTripper struct {
	addTLS bool
}

func (t *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	tls := &tls.ConnectionState{}
	if !t.addTLS {
		tls = nil
	}
	return &http.Response{
		Request:    req,
		StatusCode: http.StatusMovedPermanently,
		TLS:        tls,
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

	mrt := &mockRoundTripper{addTLS: true}
	transport := &vcrTransport{
		PCB: &pcb{
			DisableRecording: true,
			Transport:        mrt,
			RequestFilter:    requestFilters.combined(),
			ResponseFilter:   responseFilters.combined(),
			SaveFilter:       ResponseSetTLS(nil),
			Logger:           logger,
			CassettePath:     "",
		},
		Cassette: newCassette("", ""),
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
		TLS:        nil,
		Status:     http.StatusText(http.StatusMovedPermanently),
	}

	if !reflect.DeepEqual(req, wantReq) {
		t.Errorf("vcrTransport.RoundTrip() Request has been modified = %+v, want %+v", req, wantReq)
	}

	if r1, r2 := *gotResp.Request, *wantResp.Request; !reflect.DeepEqual(r1, r2) {
		t.Errorf("vcrTransport.RoundTrip() Response request has been modified = %+v, want %+v", r1, r2)
	}

	// These are compared above.
	gotResp.Request, wantResp.Request = nil, nil
	if !reflect.DeepEqual(gotResp, &wantResp) {
		t.Errorf("vcrTransport.RoundTrip() Response has been modified = %+v, want %+v", gotResp, wantResp)
	}
}

func Test_vcrTransport_RemoveTLS(t *testing.T) {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	type fields struct {
		removeTLS bool
	}
	type args struct {
		track track
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "with tls, keep",
			fields: fields{
				removeTLS: false,
			},
			args: args{
				track: track{
					Response: response{
						TLS: &tls.ConnectionState{},
					},
				},
			},
		},
		{
			name: "with tls, remove",
			fields: fields{
				removeTLS: true,
			},
			args: args{
				track: track{
					Response: response{
						TLS: &tls.ConnectionState{},
					},
				},
			},
		},
		{
			name: "without tls, keep",
			fields: fields{
				removeTLS: false,
			},
			args: args{
				track: track{
					Response: response{
						TLS: nil,
					},
				},
			},
		},
		{
			name: "without tls, remove",
			fields: fields{
				removeTLS: true,
			},
			args: args{
				track: track{
					Response: response{
						TLS: nil,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveFilters := ResponseFilters{}
			if tt.fields.removeTLS {
				saveFilters.Add(ResponseSetTLS(nil))
			}

			mrt := &mockRoundTripper{addTLS: tt.args.track.Response.TLS != nil}
			transport := &vcrTransport{
				PCB: &pcb{
					DisableRecording: false,
					Transport:        mrt,
					RequestFilter:    nil,
					ResponseFilter:   nil,
					Logger:           logger,
					SaveFilter:       saveFilters.combined(),
					CassettePath:     "",
				},
				Cassette: newCassette("", ""),
			}

			req, err := http.NewRequest("GET", "https://example.com/path?query", toReadCloser([]byte("Lorem ipsum dolor sit amet")))
			if err != nil {
				t.Errorf("req http.NewRequest() error = %v", err)
				return
			}

			_, err = transport.RoundTrip(req)
			if err != nil {
				t.Errorf("vcrTransport.RoundTrip() error = %v", err)
				return
			}

			gotTLS := transport.Cassette.Tracks[0].Response.TLS != nil
			if gotTLS && tt.fields.removeTLS {
				t.Errorf("got TLS, but it should have been removed")
			}
			if !gotTLS && !tt.fields.removeTLS && tt.args.track.Response.TLS != nil {
				t.Errorf("tls was removed, but shouldn't")
			}
		})
	}
}
