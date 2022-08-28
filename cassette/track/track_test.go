package track_test

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/v12/cassette/track"
)

func TestTrack_ToHTTPResponse(t *testing.T) {
	trk := track.Track{
		Request: track.Request{
			Method: http.MethodConnect,
			URL: &url.URL{
				Scheme:      "req_url_scheme",
				Opaque:      "req_url_opaque",
				User:        url.UserPassword("req_url_user", "req_url_passw"),
				Host:        "req_url_host",
				Path:        "req_url_path",
				RawPath:     "req_url_rawpath",
				ForceQuery:  true,
				RawQuery:    "req_url_rawquery",
				Fragment:    "req_url_fragment",
				RawFragment: "req_url_rawfragment",
			},
			Proto:            "req_proto",
			ProtoMajor:       17,
			ProtoMinor:       32,
			Header:           http.Header{"req header": {"req header value"}},
			Body:             []byte("req body"),
			ContentLength:    273,
			TransferEncoding: []string{"req_tsfencoding"},
			Close:            true,
			Host:             "req_host",
			Form: map[string][]string{
				"req_fo1": {"req_fov11", "req_fov22"},
			},
			PostForm: map[string][]string{
				"req_pofo1": {"req_pofov11", "req_pofov22"},
			},
			MultipartForm: &multipart.Form{
				Value: map[string][]string{
					"req_mupafova1": {"req_mupafovava11", "req_mupafovava22"},
				},
				File: map[string][]*multipart.FileHeader{
					"req_mupafofi1": {{
						Filename: "req_mupafofifi1",
						Header: map[string][]string{
							"req_mupafofihe1": {"req_mupafofiheva11", "req_mupafofiheva22"},
						},
						Size: 4974,
					}},
				},
			},
			Trailer: map[string][]string{
				"tr1": {"trva11", "trva22"},
			},
			RemoteAddr: "req_remoteaddr",
			RequestURI: "req_requri",
		},
		Response: &track.Response{
			Status:           "resp status",
			StatusCode:       56,
			Proto:            "resp proto",
			ProtoMajor:       28,
			ProtoMinor:       85,
			Header:           http.Header{"resp header": {"resp header value"}},
			Body:             []byte("resp body"),
			ContentLength:    826,
			TransferEncoding: []string{"resp tsf_encoding"},
			Close:            true,
			Uncompressed:     true,
			Trailer: map[string][]string{
				"resptr1": {"resptrva11", "resptrva22"},
			},
			TLS:     &tls.ConnectionState{PeerCertificates: []*x509.Certificate{{SerialNumber: big.NewInt(1234)}}},
			Request: nil,
		},
	}

	httpResp := trk.ToHTTPResponse()

	respBody, err := io.ReadAll(httpResp.Body)
	require.NoError(t, err)
	httpResp.Body.Close()

	assert.Equal(t, []byte("resp body"), respBody)
	httpResp.Body = nil // now we've asserted the body, clear it for the next steps to succeed

	expectedhttpResp := http.Response{
		Status:           "resp status",
		StatusCode:       56,
		Proto:            "resp proto",
		ProtoMajor:       28,
		ProtoMinor:       85,
		Header:           http.Header{"resp header": {"resp header value"}},
		Body:             nil, // we assert Body separately because it's an io.Reader
		ContentLength:    826,
		TransferEncoding: []string{"resp tsf_encoding"},
		Close:            true,
		Uncompressed:     true,
		Trailer: map[string][]string{
			"resptr1": {"resptrva11", "resptrva22"},
		},
		Request: &http.Request{
			Method: http.MethodConnect,
			URL: &url.URL{
				Scheme:      "req_url_scheme",
				Opaque:      "req_url_opaque",
				User:        url.UserPassword("req_url_user", "req_url_passw"),
				Host:        "req_url_host",
				Path:        "req_url_path",
				RawPath:     "req_url_rawpath",
				ForceQuery:  true,
				RawQuery:    "req_url_rawquery",
				Fragment:    "req_url_fragment",
				RawFragment: "req_url_rawfragment",
			},
			Proto:            "req_proto",
			ProtoMajor:       17,
			ProtoMinor:       32,
			Header:           http.Header{"req header": {"req header value"}},
			Body:             nil, // as per http.Response.Request comment in Go sources
			ContentLength:    273,
			TransferEncoding: []string{"req_tsfencoding"},
			Close:            true,
			Host:             "req_host",
			Form: map[string][]string{
				"req_fo1": {"req_fov11", "req_fov22"},
			},
			PostForm: map[string][]string{
				"req_pofo1": {"req_pofov11", "req_pofov22"},
			},
			MultipartForm: &multipart.Form{
				Value: map[string][]string{
					"req_mupafova1": {"req_mupafovava11", "req_mupafovava22"},
				},
				File: map[string][]*multipart.FileHeader{
					"req_mupafofi1": {{
						Filename: "req_mupafofifi1",
						Header: map[string][]string{
							"req_mupafofihe1": {"req_mupafofiheva11", "req_mupafofiheva22"},
						},
						Size: 4974,
					}},
				},
			},
			Trailer: map[string][]string{
				"tr1": {"trva11", "trva22"},
			},
			RemoteAddr: "req_remoteaddr",
			RequestURI: "req_requri",
		},
		TLS: &tls.ConnectionState{PeerCertificates: []*x509.Certificate{{SerialNumber: big.NewInt(1234)}}},
	}

	assert.Equal(t, &expectedhttpResp, httpResp)
}
