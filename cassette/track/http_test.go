package track_test

import (
	"encoding/json"
	"mime/multipart"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/v13/cassette/track"
)

func TestRequest_Clone(t *testing.T) {
	tt := map[string]struct {
		req       *track.Request
		wantClone *track.Request
	}{
		"nil": {
			req:       nil,
			wantClone: nil,
		},
		"full request": {
			req: &track.Request{
				Method: "me",
				URL: &url.URL{
					Scheme:      "usc",
					Opaque:      "uop",
					User:        url.UserPassword("uusus", "uuspa"),
					Host:        "uho",
					Path:        "upa",
					RawPath:     "urapa",
					ForceQuery:  true,
					RawQuery:    "uraqu",
					Fragment:    "ufr",
					RawFragment: "urfr",
				},
				Proto:      "pr",
				ProtoMajor: 2834,
				ProtoMinor: 82659,
				Header: map[string][]string{
					"he1": {"heva11", "heva22"},
				},
				Body:             []byte("bo"),
				ContentLength:    1283,
				TransferEncoding: []string{"te1", "te2"},
				Close:            true,
				Host:             "ho",
				Form: map[string][]string{
					"fo1": {"fov11", "fov22"},
				},
				PostForm: map[string][]string{
					"pofo1": {"pofov11", "pofov22"},
				},
				MultipartForm: &multipart.Form{
					Value: map[string][]string{
						"mupafova1": {"mupafovava11", "mupafovava22"},
					},
					File: map[string][]*multipart.FileHeader{
						"mupafofi1": {{
							Filename: "mupafofifi1",
							Header: map[string][]string{
								"mupafofihe1": {"mupafofiheva11", "mupafofiheva22"},
							},
							Size: 4974,
						}},
					},
				},
				Trailer: map[string][]string{
					"tr1": {"trva11", "trva22"},
				},
				RemoteAddr: "read",
				RequestURI: "reur",
			},
			wantClone: &track.Request{
				Method: "me",
				URL: &url.URL{
					Scheme:      "usc",
					Opaque:      "uop",
					User:        url.UserPassword("uusus", "uuspa"),
					Host:        "uho",
					Path:        "upa",
					RawPath:     "urapa",
					ForceQuery:  true,
					RawQuery:    "uraqu",
					Fragment:    "ufr",
					RawFragment: "urfr",
				},
				Proto:      "pr",
				ProtoMajor: 2834,
				ProtoMinor: 82659,
				Header: map[string][]string{
					"he1": {"heva11", "heva22"},
				},
				Body:             []byte("bo"),
				ContentLength:    1283,
				TransferEncoding: []string{"te1", "te2"},
				Close:            true,
				Host:             "ho",
				Form: map[string][]string{
					"fo1": {"fov11", "fov22"},
				},
				PostForm: map[string][]string{
					"pofo1": {"pofov11", "pofov22"},
				},
				MultipartForm: &multipart.Form{
					Value: map[string][]string{
						"mupafova1": {"mupafovava11", "mupafovava22"},
					},
					File: map[string][]*multipart.FileHeader{
						"mupafofi1": {{
							Filename: "mupafofifi1",
							Header: map[string][]string{
								"mupafofihe1": {"mupafofiheva11", "mupafofiheva22"},
							},
							Size: 4974,
						}},
					},
				},
				Trailer: map[string][]string{
					"tr1": {"trva11", "trva22"},
				},
				RemoteAddr: "read",
				RequestURI: "reur",
			},
		},
	}

	for name, tc := range tt {
		name := name
		tc := tc

		t.Run(name, func(t *testing.T) {
			reqJSON, err := json.MarshalIndent(tc.req, "", "  ")
			require.NoError(t, err)

			got := tc.req.Clone()
			assert.Equal(t, tc.wantClone, got)
			if tc.req == nil {
				return
			}

			// mutate "got" and confirm it does not affect "req" to prove both objects are independent entities
			require.Contains(t, got.MultipartForm.File, "mupafofi1")
			got.MultipartForm.File["mupafofi1"][0].Header.Set("mupafofihe1", "changed11")

			reqJSON2, err := json.MarshalIndent(tc.req, "", "  ")
			require.NoError(t, err)
			assert.Equal(t, string(reqJSON), string(reqJSON2))
		})
	}
}
