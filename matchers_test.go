package govcr_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/seborama/govcr/v17"
	"github.com/seborama/govcr/v17/cassette/track"
)

func Test_DefaultHeaderMatcher(t *testing.T) {
	tt := []*struct {
		name         string
		reqHeaders   http.Header
		trackHeaders http.Header
		want         bool
	}{
		{
			name:         "matches nil headers",
			reqHeaders:   nil,
			trackHeaders: nil,
			want:         true,
		},
		{
			name:         "matches nil request header with empty track header",
			reqHeaders:   nil,
			trackHeaders: http.Header{},
			want:         true,
		},
		{
			name:         "matches empty request header with nil track header",
			reqHeaders:   http.Header{},
			trackHeaders: nil,
			want:         true,
		},
		{
			name:         "does not match nil request header with non-empty track header",
			reqHeaders:   nil,
			trackHeaders: http.Header{"header": {"value"}},
			want:         false,
		},
		{
			name:         "does not match non-empty request header with nil track header",
			reqHeaders:   http.Header{"header": {"value"}},
			trackHeaders: nil,
			want:         false,
		},
		{
			name:         "matches two complex unordered equivalent non-empty headers",
			reqHeaders:   http.Header{"header1": {"value1"}, "header2": {"value2b", "value2a"}},
			trackHeaders: http.Header{"header2": {"value2a", "value2b"}, "header1": {"value1"}},
			want:         true,
		},
		{
			name:         "does not match two non-identical non-empty headers",
			reqHeaders:   http.Header{"header": {"value"}},
			trackHeaders: http.Header{"other": {"something"}},
			want:         false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			httpReq := track.Request{Header: tc.reqHeaders}
			trackReq := track.Request{Header: tc.trackHeaders}
			actualMatch := govcr.DefaultHeaderMatcher(&httpReq, &trackReq)
			assert.Equal(t, tc.want, actualMatch)
		})
	}
}

func Test_DefaultMethodMatcher(t *testing.T) {
	tt := []*struct {
		name        string
		reqMethod   string
		trackMethod string
		want        bool
	}{
		{
			name:        "matches nil methods",
			reqMethod:   string([]byte(nil)),
			trackMethod: string([]byte(nil)),
			want:        true,
		},
		{
			name:        "matches nil request method with empty track method",
			reqMethod:   string([]byte(nil)),
			trackMethod: "",
			want:        true,
		},
		{
			name:        "matches empty request method with nil track method",
			reqMethod:   "",
			trackMethod: string([]byte(nil)),
			want:        true,
		},
		{
			name:        "does not match nil request method with non-empty track method",
			reqMethod:   string([]byte(nil)),
			trackMethod: http.MethodGet,
			want:        false,
		},
		{
			name:        "does not match non-empty request method with nil track method",
			reqMethod:   http.MethodGet,
			trackMethod: string([]byte(nil)),
			want:        false,
		},
		{
			name:        "matches two identical methods",
			reqMethod:   http.MethodGet,
			trackMethod: http.MethodGet,
			want:        true,
		},
		{
			name:        "does not match differing methods",
			reqMethod:   http.MethodGet,
			trackMethod: http.MethodPost,
			want:        false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			httpReq := track.Request{Method: tc.reqMethod}
			trackReq := track.Request{Method: tc.trackMethod}
			actualMatch := govcr.DefaultMethodMatcher(&httpReq, &trackReq)
			assert.Equal(t, tc.want, actualMatch)
		})
	}
}

func Test_DefaultURLMatcher(t *testing.T) {
	tt := []*struct {
		name     string
		reqURL   *url.URL
		trackURL *url.URL
		want     bool
	}{
		{
			name:     "matches nil URLs",
			reqURL:   nil,
			trackURL: nil,
			want:     true,
		},
		{
			name:     "matches empty request URL with nil track URL",
			reqURL:   &url.URL{},
			trackURL: nil,
			want:     true,
		},
		{
			name:     "matches nil request URL with empty track URL",
			reqURL:   &url.URL{},
			trackURL: nil,
			want:     true,
		},
		{
			name: "does not match non-empty request URL with nil track URL",
			reqURL: &url.URL{
				User: url.UserPassword("a", "b"),
			},
			trackURL: nil,
			want:     false,
		},
		{
			name: "does not match nil request URL with non-empty track URL",
			reqURL: &url.URL{
				User: url.UserPassword("a", "b"),
			},
			trackURL: nil,
			want:     false,
		},
		{
			name: "matches two identical URLs",
			reqURL: &url.URL{
				Scheme:     "scheme",
				Opaque:     "opaque",
				User:       url.UserPassword("a", "b"),
				Host:       "host",
				Path:       "path/",
				RawPath:    "/path/raw",
				ForceQuery: false,
				RawQuery:   "rawq",
				Fragment:   "frag",
			},
			trackURL: &url.URL{
				Scheme:     "scheme",
				Opaque:     "opaque",
				User:       url.UserPassword("a", "b"),
				Host:       "host",
				Path:       "path/",
				RawPath:    "/path/raw",
				ForceQuery: false,
				RawQuery:   "rawq",
				Fragment:   "frag",
			},
			want: true,
		},
		{
			name: "does not match differing URLs",
			reqURL: &url.URL{
				User: url.UserPassword("1", "2"),
			},
			trackURL: &url.URL{
				User: url.UserPassword("a", "b"),
			},
			want: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			httpReq := track.Request{URL: tc.reqURL}
			trackReq := track.Request{URL: tc.trackURL}
			actualMatch := govcr.DefaultURLMatcher(&httpReq, &trackReq)
			assert.Equal(t, tc.want, actualMatch)
		})
	}
}

func Test_DefaultBodyMatcher(t *testing.T) {
	tt := []*struct {
		name      string
		reqBody   []byte
		trackBody []byte
		want      bool
	}{
		{
			name:      "matches nil bodies",
			reqBody:   nil,
			trackBody: nil,
			want:      true,
		},
		{
			name:      "matches nil request bodies with empty track bodies",
			reqBody:   nil,
			trackBody: []byte{},
			want:      true,
		},
		{
			name:      "matches empty request bodies with nil track bodies",
			reqBody:   []byte{},
			trackBody: nil,
			want:      true,
		},
		{
			name:      "does not match nil request bodies with non-empty track bodies",
			reqBody:   nil,
			trackBody: []byte("something"),
			want:      false,
		},
		{
			name:      "does not match non-empty request bodies with nil track bodies",
			reqBody:   []byte("something"),
			trackBody: nil,
			want:      false,
		},
		{
			name:      "matches two identical bodies",
			reqBody:   []byte("something"),
			trackBody: []byte("something"),
			want:      true,
		},
		{
			name:      "does not match differing bodies",
			reqBody:   []byte("something"),
			trackBody: []byte("another thing"),
			want:      false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			httpReq := track.Request{Body: tc.reqBody}
			trackReq := track.Request{Body: tc.trackBody}
			actualMatch := govcr.DefaultBodyMatcher(&httpReq, &trackReq)
			assert.Equal(t, tc.want, actualMatch)
		})
	}
}

func Test_DefaultTrailerMatcher(t *testing.T) {
	tt := []*struct {
		name         string
		reqHeaders   http.Header
		trackHeaders http.Header
		want         bool
	}{
		{
			name:         "matches nil trailers",
			reqHeaders:   nil,
			trackHeaders: nil,
			want:         true,
		},
		{
			name:         "matches nil request trailer with empty track trailer",
			reqHeaders:   nil,
			trackHeaders: http.Header{},
			want:         true,
		},
		{
			name:         "matches empty request trailer with nil track trailer",
			reqHeaders:   http.Header{},
			trackHeaders: nil,
			want:         true,
		},
		{
			name:         "does not match nil request trailer with non-empty track trailer",
			reqHeaders:   nil,
			trackHeaders: http.Header{"trailer": {"value"}},
			want:         false,
		},
		{
			name:         "does not match non-empty request trailer with nil track trailer",
			reqHeaders:   http.Header{"trailer": {"value"}},
			trackHeaders: nil,
			want:         false,
		},
		{
			name:         "matches two complex unordered equivalent non-empty trailers",
			reqHeaders:   http.Header{"trailer1": {"value1"}, "trailer2": {"value2b", "value2a"}},
			trackHeaders: http.Header{"trailer2": {"value2a", "value2b"}, "trailer1": {"value1"}},
			want:         true,
		},
		{
			name:         "does not match two non-identical non-empty trailers",
			reqHeaders:   http.Header{"trailer": {"value"}},
			trackHeaders: http.Header{"other": {"something"}},
			want:         false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			httpReq := track.Request{Header: tc.reqHeaders}
			trackReq := track.Request{Header: tc.trackHeaders}
			actualMatch := govcr.DefaultHeaderMatcher(&httpReq, &trackReq)
			assert.Equal(t, tc.want, actualMatch)
		})
	}
}
