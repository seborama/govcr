package track_test

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/v17/cassette/track"
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
			assert.JSONEq(t, string(reqJSON), string(reqJSON2))
		})
	}
}

func TestRequestURLUserinfo(t *testing.T) {
	tests := map[string]*url.Userinfo{
		"no userinfo":       nil,
		"username only":     url.User("reader"),
		"empty username":    url.User(""),
		"empty password":    url.UserPassword("reader", ""),
		"plain password":    url.UserPassword("reader", "secret"),
		"reserved username": url.User("reader:@/% name"),
		"reserved password": url.UserPassword("reader", "p:@/% word"),
		"unicode":           url.UserPassword("lecteuré", "秘密"),
		"literal escapes":   url.UserPassword("reader%40", "p%3Ass"),
	}

	cloners := map[string]func(*url.URL) *url.URL{
		"ToRequest": func(requestURL *url.URL) *url.URL {
			return track.ToRequest(&http.Request{URL: requestURL}).URL
		},
		"Request.Clone": func(requestURL *url.URL) *url.URL {
			return (&track.Request{URL: requestURL}).Clone().URL
		},
		"CloneHTTPRequest": func(requestURL *url.URL) *url.URL {
			return track.CloneHTTPRequest(&http.Request{URL: requestURL}).URL
		},
	}

	for name, clone := range cloners {
		t.Run(name, func(t *testing.T) {
			t.Run("nil URL", func(t *testing.T) {
				assert.Nil(t, clone(nil))
			})

			for testName, user := range tests {
				t.Run(testName, func(t *testing.T) {
					original := &url.URL{
						Scheme: "https",
						User:   user,
						Host:   "example.com",
						Path:   "/recorded",
					}
					wantURL := original.String()
					wantPassword, wantHasPassword := user.Password()

					cloned := clone(original)
					require.NotNil(t, cloned)
					assert.NotSame(t, original, cloned)
					assert.Equal(t, wantURL, cloned.String())
					assert.Equal(t, user == nil, cloned.User == nil)
					assert.Equal(t, user.Username(), cloned.User.Username())
					password, hasPassword := cloned.User.Password()
					assert.Equal(t, wantPassword, password)
					assert.Equal(t, wantHasPassword, hasPassword)
					assert.Equal(t, wantURL, clone(cloned).String())
					assert.Equal(t, wantURL, original.String())
					cloned.Host = "clone.example.com"
					assert.Equal(t, "example.com", original.Host)
				})
			}
		})
	}
}
