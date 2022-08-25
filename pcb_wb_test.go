package govcr

import (
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/v11/cassette"
	"github.com/seborama/govcr/v11/cassette/track"
)

func TestPrintedCircuitBoard_trackMatches(t *testing.T) {
	// This test is in addition toTestRoundTrip_RequestMatcherDoesNotMutateState
	pcb := &PrintedCircuitBoard{}
	pcb.SetRequestMatcher(
		NewBlankRequestMatcher(
			WithRequestMatcherFunc(
				func(httpRequest, trackRequest *track.Request) bool {
					for k := range httpRequest.Header {
						httpRequest.Header.Set(k, "nil")
					}
					for i := range httpRequest.Body {
						httpRequest.Body[i] = 'X'
					}
					httpRequest.ContentLength = -1
					for k := range httpRequest.MultipartForm.File {
						for k2 := range httpRequest.MultipartForm.File[k] {
							httpRequest.MultipartForm.File[k][k2].Filename = "nil"
							for k3 := range httpRequest.MultipartForm.File[k][k2].Header {
								httpRequest.MultipartForm.File[k][k2].Header.Set(k3, "nil")
							}
						}
					}

					for k := range trackRequest.Header {
						trackRequest.Header.Set(k, "nil")
					}
					for i := range trackRequest.Body {
						trackRequest.Body[i] = 'X'
					}
					trackRequest.ContentLength = -1
					for k := range trackRequest.MultipartForm.File {
						for k2 := range trackRequest.MultipartForm.File[k] {
							trackRequest.MultipartForm.File[k][k2].Filename = "nil"
							for k3 := range trackRequest.MultipartForm.File[k][k2].Header {
								trackRequest.MultipartForm.File[k][k2].Header.Set(k3, "nil")
							}
						}
					}

					httpRequest = nil
					trackRequest = nil

					return true
				},
			),
		),
	)

	k7 := &cassette.Cassette{
		Tracks: []track.Track{
			{
				Request: track.Request{
					Header: http.Header{
						"req header": {"req header value"},
					},
					Body:          []byte("req body"),
					ContentLength: 456,
					MultipartForm: &multipart.Form{
						File: map[string][]*multipart.FileHeader{
							"req multipartpostform file": {
								{
									Filename: "req multipartpostform file filename",
									Header: textproto.MIMEHeader{
										"req form": []string{"req form value"},
									},
								},
							},
						},
					},
				},
				Response: &track.Response{
					Status: "resp status",
					Header: http.Header{
						"resp header": {"resp header value"},
					},
					Body: []byte("resp body"),
					TLS: &tls.ConnectionState{
						PeerCertificates: []*x509.Certificate{
							{
								SerialNumber: big.NewInt(1234),
							},
						},
					},
				},
				ErrType: strPtr("err type"),
				ErrMsg:  strPtr("err message"),
				UUID:    "test uuid",
			},
		},
	}

	httpRequest := &track.Request{
		Header: map[string][]string{
			"hreq header": {"hreq header value"},
		},
		Body:          []byte("hreq body"),
		ContentLength: 890,
		MultipartForm: &multipart.Form{
			File: map[string][]*multipart.FileHeader{
				"hreq multipartpostform file": {
					{
						Filename: "hreq multipartpostform file filename",
						Header: textproto.MIMEHeader{
							"hreq form": []string{"hreq form value"},
						},
					},
				},
			},
		},
	}

	res := pcb.trackMatches(k7, 0, httpRequest)
	require.True(t, res)

	// track validation
	require.Equal(
		t,
		http.Header{
			"req header": {"req header value"},
		},
		k7.Tracks[0].Request.Header,
	)
	require.Equal(
		t,
		[]byte("req body"),
		k7.Tracks[0].Request.Body,
	)
	require.Equal(
		t,
		int64(456),
		k7.Tracks[0].Request.ContentLength,
	)
	require.Len(
		t,
		k7.Tracks,
		1,
	)
	_, ok := k7.Tracks[0].Request.MultipartForm.File["req multipartpostform file"]
	require.True(
		t,
		ok,
	)
	require.Len(
		t,
		k7.Tracks[0].Request.MultipartForm.File["req multipartpostform file"],
		1,
	)
	require.Equal(
		t,
		"req multipartpostform file filename",
		k7.Tracks[0].Request.MultipartForm.File["req multipartpostform file"][0].Filename,
	)

	// httpRequest validation
	require.Equal(
		t,
		http.Header{
			"hreq header": {"hreq header value"},
		},
		httpRequest.Header,
	)
	require.Equal(
		t,
		[]byte("hreq body"),
		httpRequest.Body,
	)
	require.Equal(
		t,
		int64(890),
		httpRequest.ContentLength,
	)
	require.Len(
		t,
		k7.Tracks,
		1,
	)
	_, ok2 := httpRequest.MultipartForm.File["hreq multipartpostform file"]
	require.True(
		t,
		ok2,
	)
	require.Len(
		t,
		httpRequest.MultipartForm.File["hreq multipartpostform file"],
		1,
	)
	require.Equal(
		t,
		"hreq multipartpostform file filename",
		httpRequest.MultipartForm.File["hreq multipartpostform file"][0].Filename,
	)
}

func strPtr(s string) *string { return &s }
