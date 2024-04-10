package track_test

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/v15/cassette/track"
)

func Test_Mutator_On(t *testing.T) {
	mutatorCallCounter := 0

	unitMutator := track.Mutator(
		func(tk *track.Track) {
			mutatorCallCounter++
		},
	)

	pTrue := track.Predicate(
		func(trk *track.Track) bool {
			return true
		},
	)

	pFalse := track.Predicate(
		func(trk *track.Track) bool {
			return false
		},
	)

	trk := track.NewTrack(
		&track.Request{},
		&track.Response{
			StatusCode: 172,
		},
		nil,
	)

	mutatorCallCounter = 0
	unitMutator.On(pTrue)(nil)
	require.Equal(t, 0, mutatorCallCounter)

	mutatorCallCounter = 0
	unitMutator.On(pTrue)(trk)
	require.Equal(t, 1, mutatorCallCounter)

	mutatorCallCounter = 0
	unitMutator.On(pFalse)(trk)
	require.Equal(t, 0, mutatorCallCounter)
}

func Test_Mutator_Any(t *testing.T) {
	pTrue := track.Predicate(
		func(trk *track.Track) bool {
			return true
		},
	)

	pFalse := track.Predicate(
		func(trk *track.Track) bool {
			return false
		},
	)

	trk := track.NewTrack(nil, nil, nil)

	result := track.Any(pFalse, pTrue)(nil)
	require.True(t, result)

	result = track.Any(pFalse, pTrue)(trk)
	require.True(t, result)

	result = track.Any(pFalse, pFalse)(trk)
	require.False(t, result)

	result = track.Any(pTrue, pTrue)(trk)
	require.True(t, result)
}

func Test_Mutator_None(t *testing.T) {
	pTrue := track.Predicate(
		func(trk *track.Track) bool {
			return true
		},
	)

	pFalse := track.Predicate(
		func(trk *track.Track) bool {
			return false
		},
	)

	trk := track.NewTrack(nil, nil, nil)

	result := track.None(pFalse, pTrue)(nil)
	require.False(t, result)

	result = track.None(pFalse, pTrue)(trk)
	require.False(t, result)

	result = track.None(pFalse, pFalse)(trk)
	require.True(t, result)

	result = track.None(pTrue, pTrue)(trk)
	require.False(t, result)
}

func Test_Mutator_All(t *testing.T) {
	pTrue := track.Predicate(
		func(trk *track.Track) bool {
			return true
		},
	)

	pFalse := track.Predicate(
		func(trk *track.Track) bool {
			return false
		},
	)

	trk := track.NewTrack(nil, nil, nil)

	result := track.All(pFalse, pTrue)(nil)
	require.False(t, result)

	result = track.All(pFalse, pTrue)(trk)
	require.False(t, result)

	result = track.All(pFalse, pFalse)(trk)
	require.False(t, result)

	result = track.All(pTrue, pTrue)(trk)
	require.True(t, result)
}

func Test_Mutator_Not(t *testing.T) {
	pTrue := track.Predicate(
		func(trk *track.Track) bool {
			return true
		},
	)

	pFalse := track.Predicate(
		func(trk *track.Track) bool {
			return false
		},
	)

	result := track.Not(pFalse)(nil)
	require.True(t, result)
	result = track.Not(pTrue)(nil)
	require.False(t, result)

	trk := track.NewTrack(nil, nil, nil)

	result = track.Not(pFalse)(trk)
	require.True(t, result)
	result = track.Not(pTrue)(trk)
	require.False(t, result)

	result = track.Not(track.Any(pFalse, pTrue))(trk)
	require.False(t, result)
	result = track.Not(track.Any(pTrue, pFalse))(trk)
	require.False(t, result)
	result = track.Not(track.Any(pFalse, pFalse))(trk)
	require.True(t, result)
	result = track.Not(track.Any(pTrue, pTrue))(trk)
	require.False(t, result)

	result = track.Not(track.All(pFalse, pTrue))(trk)
	require.True(t, result)
	result = track.Not(track.All(pTrue, pFalse))(trk)
	require.True(t, result)
	result = track.Not(track.All(pFalse, pFalse))(trk)
	require.True(t, result)
	result = track.Not(track.All(pTrue, pTrue))(trk)
	require.False(t, result)
}

func Test_Mutator_HasErr(t *testing.T) {
	assert.Panics(t, func() { track.HasErr()(nil) })

	assert.True(
		t,
		track.HasErr()(
			track.NewTrack(
				nil,
				nil,
				nil,
			),
		),
	)

	assert.False(
		t,
		track.HasErr()(
			track.NewTrack(
				nil,
				nil,
				errors.New("some error"),
			),
		),
	)
}

func Test_Mutator_HasNoErr(t *testing.T) {
	assert.Panics(t, func() { track.HasErr()(nil) })

	assert.False(
		t,
		track.HasNoErr()(
			track.NewTrack(
				nil,
				nil,
				nil,
			),
		),
	)

	assert.True(
		t,
		track.HasNoErr()(
			track.NewTrack(
				nil,
				nil,
				errors.New("some error"),
			),
		),
	)
}

func Test_Mutator_OnNoErr_WhenNoErr(t *testing.T) {
	unitMutator := track.Mutator(
		func(tk *track.Track) {
			tk.Request.Method += " has been mutated"
			tk.Response.Status += " has been mutated"
			tk.ErrType = strPtr("ErrType was mutated")
			tk.ErrMsg = strPtr("ErrMsg was mutated")
		}).OnNoErr()

	trk := track.NewTrack(
		&track.Request{
			Method: "BadMethod",
		},
		&track.Response{
			Status: "BadStatus",
		},
		nil,
	)

	assert.NotPanics(t, func() { unitMutator(nil) })

	unitMutator(trk)
	require.Equal(t, "BadMethod has been mutated", trk.Request.Method)
	require.Equal(t, "BadStatus has been mutated", trk.Response.Status)
	require.Equal(t, strPtr("ErrType was mutated"), trk.ErrType)
	require.Equal(t, strPtr("ErrMsg was mutated"), trk.ErrMsg)
}

func Test_Mutator_OnNoErr_WhenErr(t *testing.T) {
	unitMutator := track.Mutator(
		func(tk *track.Track) {
			tk.Request.Method += " has been mutated"
			tk.Response.Status += " has been mutated"
			tk.ErrType = strPtr("ErrType was mutated")
			tk.ErrMsg = strPtr("ErrMsg was mutated")
		}).OnNoErr()

	trk := track.NewTrack(
		&track.Request{
			Method: "BadMethod",
		},
		&track.Response{
			Status: "BadStatus",
		},
		errors.New("an error"),
	)

	assert.NotPanics(t, func() { unitMutator(nil) })

	unitMutator(trk)
	require.Equal(t, "BadMethod", trk.Request.Method)
	require.Equal(t, "BadStatus", trk.Response.Status)
	require.Equal(t, strPtr("*errors.errorString"), trk.ErrType)
	require.Equal(t, strPtr("an error"), trk.ErrMsg)
}

func Test_Mutator_OnErr_WhenErr(t *testing.T) {
	unitMutator := track.Mutator(
		func(tk *track.Track) {
			tk.Request.Method += " has been mutated"
			tk.Response.Status += " has been mutated"
			tk.ErrType = strPtr("ErrType was mutated")
			tk.ErrMsg = strPtr("ErrMsg was mutated")
		}).OnErr()

	trk := track.NewTrack(
		&track.Request{
			Method: "BadMethod",
		},
		&track.Response{
			Status: "BadStatus",
		},
		errors.New("an error"))

	assert.NotPanics(t, func() { unitMutator(nil) })

	unitMutator(trk)
	require.Equal(t, "BadMethod has been mutated", trk.Request.Method)
	require.Equal(t, "BadStatus has been mutated", trk.Response.Status)
	require.Equal(t, strPtr("ErrType was mutated"), trk.ErrType)
	require.Equal(t, strPtr("ErrMsg was mutated"), trk.ErrMsg)
}

func Test_Mutator_OnErr_WhenNoErr(t *testing.T) {
	unitMutator := track.Mutator(
		func(tk *track.Track) {
			tk.Request.Method += " has been mutated"
			tk.Response.Status += " has been mutated"
			tk.ErrType = strPtr("ErrType was mutated")
			tk.ErrMsg = strPtr("ErrMsg was mutated")
		}).OnErr()

	trk := track.NewTrack(
		&track.Request{
			Method: "BadMethod",
		},
		&track.Response{
			Status: "BadStatus",
		},
		nil,
	)

	assert.NotPanics(t, func() { unitMutator(nil) })

	unitMutator(trk)
	require.Equal(t, "BadMethod", trk.Request.Method)
	require.Equal(t, "BadStatus", trk.Response.Status)
	require.Nil(t, trk.ErrType)
	require.Nil(t, trk.ErrMsg)
}

func Test_Mutator_OnRequestPath(t *testing.T) {
	mutatorCallCounter := 0

	unitMutator := track.Mutator(
		func(tk *track.Track) {
			mutatorCallCounter++
		})

	u, err := url.Parse("http://127.0.0.1/some/test/url")
	require.NoError(t, err)

	trk := track.NewTrack(
		&track.Request{
			URL: u,
		},
		&track.Response{
			Status: "Status",
		},
		nil,
	)

	mutatorCallCounter = 0
	unitMutator.OnRequestPath(".*test.*")(nil)
	require.Equal(t, 0, mutatorCallCounter)

	mutatorCallCounter = 0
	unitMutator.OnRequestPath(".*test.*")(trk)
	require.Equal(t, 1, mutatorCallCounter)

	mutatorCallCounter = 0
	unitMutator.OnRequestPath("not-a-match")(trk)
	require.Equal(t, 0, mutatorCallCounter)
}

func Test_Mutator_OnStatus(t *testing.T) {
	mutatorCallCounter := 0

	unitMutator := track.Mutator(
		func(tk *track.Track) {
			mutatorCallCounter++
		})

	trk := track.NewTrack(
		&track.Request{},
		&track.Response{
			Status: "one",
		},
		nil,
	)

	mutatorCallCounter = 0
	unitMutator.OnStatus("one", "two", "three")(nil)
	require.Equal(t, 0, mutatorCallCounter)

	mutatorCallCounter = 0
	unitMutator.OnStatus("one", "two", "three")(trk)
	require.Equal(t, 1, mutatorCallCounter)

	mutatorCallCounter = 0
	unitMutator.OnStatus("not-a-match")(trk)
	require.Equal(t, 0, mutatorCallCounter)
}

func Test_Mutator_OnStatusCode(t *testing.T) {
	mutatorCallCounter := 0

	unitMutator := track.Mutator(
		func(tk *track.Track) {
			mutatorCallCounter++
		})

	trk := track.NewTrack(
		&track.Request{},
		&track.Response{
			StatusCode: 172,
		},
		nil,
	)

	mutatorCallCounter = 0
	unitMutator.OnStatusCode(1, 172, 4)(nil)
	require.Equal(t, 0, mutatorCallCounter)

	mutatorCallCounter = 0
	unitMutator.OnStatusCode(1, 172, 4)(trk)
	require.Equal(t, 1, mutatorCallCounter)

	mutatorCallCounter = 0
	unitMutator.OnStatusCode(-1)(trk)
	require.Equal(t, 0, mutatorCallCounter)
}

func Test_Mutator_RequestAddHeaderValue(t *testing.T) {
	unitMutator := track.TrackRequestAddHeaderValue("key-1", "value-1")

	h := http.Header{}
	h.Set("key-a", "value-b")

	trk := track.NewTrack(
		&track.Request{Header: h},
		&track.Response{},
		nil,
	)

	assert.NotPanics(t, func() { unitMutator(nil) })

	unitMutator(trk)
	assert.Equal(t, trk.Request.Header.Get("key-1"), "value-1")
}

func Test_Mutator_RequestAddHeaderValue_NilHeader(t *testing.T) {
	unitMutator := track.TrackRequestAddHeaderValue("key-1", "value-1")

	trk := track.NewTrack(
		&track.Request{},
		&track.Response{},
		nil,
	)

	assert.NotPanics(t, func() { unitMutator(nil) })

	unitMutator(trk)
	assert.Equal(t, trk.Request.Header.Get("key-1"), "value-1")
}

func Test_Mutator_RequestDeleteHeaderKeys(t *testing.T) {
	unitMutator := track.TrackRequestDeleteHeaderKeys("other", "key-a")

	h := http.Header{}
	h.Set("key-a", "value-b")

	trk := track.NewTrack(
		&track.Request{Header: h},
		&track.Response{},
		nil,
	)

	assert.NotPanics(t, func() { unitMutator(nil) })

	unitMutator(trk)
	assert.Equal(t, trk.Request.Header.Values("key-a"), []string(nil))
}

func Test_Mutator_RequestDeleteHeaderKeys_NilHeader(t *testing.T) {
	unitMutator := track.TrackRequestDeleteHeaderKeys("other", "key-a")

	trk := track.NewTrack(
		&track.Request{},
		&track.Response{},
		nil,
	)

	assert.NotPanics(t, func() { unitMutator(nil) })

	unitMutator(trk)
	assert.Equal(t, trk.Request.Header.Values("key-a"), []string(nil))
}

func Test_Mutator_ResponseAddHeaderValue(t *testing.T) {
	unitMutator := track.ResponseAddHeaderValue("key-1", "value-1")

	h := http.Header{}
	h.Set("key-a", "value-b")

	trk := track.NewTrack(
		&track.Request{},
		&track.Response{Header: h},
		nil,
	)

	assert.NotPanics(t, func() { unitMutator(nil) })

	unitMutator(trk)
	assert.Equal(t, trk.Response.Header.Get("key-1"), "value-1")
}

func Test_Mutator_ResponseAddHeaderValue_NilHeader(t *testing.T) {
	unitMutator := track.ResponseAddHeaderValue("key-1", "value-1")

	trk := track.NewTrack(
		&track.Request{},
		&track.Response{},
		nil,
	)

	assert.NotPanics(t, func() { unitMutator(nil) })

	unitMutator(trk)
	assert.Equal(t, trk.Response.Header.Get("key-1"), "value-1")
}

func Test_Mutator_ResponseDeleteHeaderKeys(t *testing.T) {
	unitMutator := track.ResponseDeleteHeaderKeys("other", "key-a")

	h := http.Header{}
	h.Set("key-a", "value-b")

	trk := track.NewTrack(
		&track.Request{},
		&track.Response{Header: h},
		nil,
	)

	assert.NotPanics(t, func() { unitMutator(nil) })

	unitMutator(trk)
	assert.Equal(t, trk.Response.Header.Values("key-a"), []string(nil))
}

func Test_Mutator_ResponseDeleteHeaderKeys_NilHeader(t *testing.T) {
	unitMutator := track.ResponseDeleteHeaderKeys("other", "key-a")

	trk := track.NewTrack(
		&track.Request{},
		&track.Response{},
		nil,
	)

	assert.NotPanics(t, func() { unitMutator(nil) })

	unitMutator(trk)
	assert.Equal(t, trk.Response.Header.Values("key-a"), []string(nil))
}

func Test_Mutator_RequestChangeBody(t *testing.T) {
	unitMutator := track.TrackRequestChangeBody(
		func(b []byte) []byte {
			return []byte("changed")
		},
	)

	assert.NotPanics(t, func() { unitMutator(nil) })

	trk := track.NewTrack(
		&track.Request{
			Body: []byte("original"),
		},
		nil,
		nil,
	)
	unitMutator(trk)
	assert.Equal(t, "changed", string(trk.Request.Body))

	trk = track.NewTrack(
		&track.Request{},
		nil,
		nil,
	)
	unitMutator(trk)
	assert.Equal(t, "changed", string(trk.Request.Body))

	trk = track.NewTrack(
		nil,
		nil,
		nil,
	)
	unitMutator(trk)
	assert.Equal(t, "changed", string(trk.Request.Body))
}

func Test_Mutator_ResponseChangeBody(t *testing.T) {
	unitMutator := track.ResponseChangeBody(
		func(b []byte) []byte {
			return []byte("changed")
		},
	)

	assert.NotPanics(t, func() { unitMutator(nil) })

	trk := track.NewTrack(
		nil,
		&track.Response{
			Body: []byte("original"),
		},
		nil,
	)
	unitMutator(trk)
	assert.Equal(t, "changed", string(trk.Response.Body))

	trk = track.NewTrack(
		nil,
		&track.Response{},
		nil,
	)
	unitMutator(trk)
	assert.Equal(t, "changed", string(trk.Response.Body))

	trk = track.NewTrack(
		nil,
		nil,
		nil,
	)
	unitMutator(trk)
	assert.Nil(t, trk.Response) // response is nil and hence the body cannot be updated
}

func Test_Mutator_ResponseDeleteTLS(t *testing.T) {
	unitMutator := track.ResponseDeleteTLS()

	assert.NotPanics(t, func() { unitMutator(nil) })

	trk := track.NewTrack(
		nil,
		&track.Response{
			TLS: &tls.ConnectionState{},
		},
		nil,
	)
	unitMutator(trk)
	assert.Nil(t, trk.Response.TLS)

	trk = track.NewTrack(
		nil,
		&track.Response{},
		nil,
	)
	unitMutator(trk)
	assert.Nil(t, trk.Response.TLS)

	trk = track.NewTrack(
		nil,
		nil,
		nil,
	)
	unitMutator(trk)
	assert.Nil(t, trk.Response)
}

func TestResponseTransferHeaderKeys_NilTrack(t *testing.T) {
	var trk *track.Track
	track.ResponseTransferHTTPHeaderKeys("unit-key-1", "unit-key-2")(trk)
	assert.Nil(t, trk)
}

func TestResponseTransferTrailerKeys_NilTrack(t *testing.T) {
	var trk *track.Track
	track.ResponseTransferHTTPTrailerKeys("unit-key-1", "unit-key-2")(trk)
	assert.Nil(t, trk)
}

func Test_Mutator_ResponseTransferHTTPHeaderKeys(t *testing.T) {
	tt := map[string]struct {
		respReqHeader  http.Header
		respHeader     http.Header
		wantRespHeader http.Header
	}{
		"nil request header and nil response header": {
			respReqHeader:  nil,
			respHeader:     nil,
			wantRespHeader: nil,
		},
		"nil request header and blank response header": {
			respReqHeader:  nil,
			respHeader:     http.Header{},
			wantRespHeader: http.Header{},
		},
		"blank request header and nil response header": {
			respReqHeader:  http.Header{},
			respHeader:     nil,
			wantRespHeader: nil,
		},
		"blank request header and blank response header": {
			respReqHeader:  http.Header{},
			respHeader:     http.Header{},
			wantRespHeader: http.Header{},
		},
		"nil response header and eligible request header": {
			respReqHeader:  func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respHeader:     nil,
			wantRespHeader: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
		},
		"blank response header and eligible request header": {
			respReqHeader:  func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respHeader:     http.Header{},
			wantRespHeader: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
		},
		"eligible request header with response header containing other data": {
			respReqHeader: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respHeader:    func() http.Header { h := http.Header{}; h.Set("unit-key-a", "unit-value-a"); return h }(),
			wantRespHeader: func() http.Header {
				h := http.Header{}
				h.Set("unit-key-a", "unit-value-a")
				h.Add("unit-key-1", "unit-value-1")
				return h
			}(),
		},
		"eligible request header with response header already containing the transfer data": {
			respReqHeader: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respHeader:    func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantRespHeader: func() http.Header {
				h := http.Header{}
				h.Set("unit-key-1", "unit-value-1")
				h.Add("unit-key-1", "unit-value-1")
				return h
			}(),
		},
	}

	for name, tc := range tt {
		name := name
		tc := tc

		t.Run(name, func(t *testing.T) {
			trk := track.NewTrack(
				nil,
				&track.Response{Header: tc.respHeader, Request: &track.Request{Header: tc.respReqHeader}},
				nil,
			)

			track.ResponseTransferHTTPHeaderKeys("unit-key-1", "unit-key-2")(trk)

			assert.Equal(t, tc.wantRespHeader, trk.Response.Header)
		})
	}
}

func Test_Mutator_ResponseTransferHTTPTrailerKeys(t *testing.T) {
	tt := map[string]struct {
		respReqTrailer  http.Header
		respTrailer     http.Header
		wantRespTrailer http.Header
	}{
		"nil request trailer and nil response trailer": {
			respReqTrailer:  nil,
			respTrailer:     nil,
			wantRespTrailer: nil,
		},
		"nil request trailer and blank response trailer": {
			respReqTrailer:  nil,
			respTrailer:     http.Header{},
			wantRespTrailer: http.Header{},
		},
		"blank request trailer and nil response trailer": {
			respReqTrailer:  http.Header{},
			respTrailer:     nil,
			wantRespTrailer: nil,
		},
		"blank request trailer and blank response trailer": {
			respReqTrailer:  http.Header{},
			respTrailer:     http.Header{},
			wantRespTrailer: http.Header{},
		},
		"nil response trailer and eligible request trailer": {
			respReqTrailer:  func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respTrailer:     nil,
			wantRespTrailer: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
		},
		"blank response trailer and eligible request trailer": {
			respReqTrailer:  func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respTrailer:     http.Header{},
			wantRespTrailer: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
		},
		"eligible request trailer with response trailer containing other data": {
			respReqTrailer: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respTrailer:    func() http.Header { h := http.Header{}; h.Set("unit-key-a", "unit-value-a"); return h }(),
			wantRespTrailer: func() http.Header {
				h := http.Header{}
				h.Set("unit-key-a", "unit-value-a")
				h.Add("unit-key-1", "unit-value-1")
				return h
			}(),
		},
		"eligible request trailer with response trailer already containing the transfer data": {
			respReqTrailer: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respTrailer:    func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantRespTrailer: func() http.Header {
				h := http.Header{}
				h.Set("unit-key-1", "unit-value-1")
				h.Add("unit-key-1", "unit-value-1")
				return h
			}(),
		},
	}

	for name, tc := range tt {
		name := name
		tc := tc

		t.Run(name, func(t *testing.T) {
			trk := track.NewTrack(
				nil,
				&track.Response{Trailer: tc.respTrailer, Request: &track.Request{Trailer: tc.respReqTrailer}},
				nil,
			)

			track.ResponseTransferHTTPTrailerKeys("unit-key-1", "unit-key-2")(trk)

			assert.Equal(t, tc.wantRespTrailer, trk.Response.Trailer)
		})
	}
}

func Test_Mutator_Multiple_On(t *testing.T) {
	tt := map[string]struct {
		mutatorOnFn func(track.Mutator) track.Mutator
		wantMethod  string
	}{
		"2 On's, both matched": {
			mutatorOnFn: func(m track.Mutator) track.Mutator {
				return m.
					OnRequestMethod(http.MethodPost).
					OnNoErr()
			},
			wantMethod: http.MethodPost + " has been mutated",
		},
		"2 On's, 1st matches, 2nd does not": {
			mutatorOnFn: func(m track.Mutator) track.Mutator {
				return m.
					OnRequestMethod(http.MethodPost).
					OnErr()
			},
			wantMethod: http.MethodPost,
		},
		"2 On's, 1st does not matches, 2nd does": {
			mutatorOnFn: func(m track.Mutator) track.Mutator {
				return m.
					OnRequestMethod(http.MethodGet).
					OnNoErr()
			},
			wantMethod: http.MethodPost,
		},
		"2 On's, none matches": {
			mutatorOnFn: func(m track.Mutator) track.Mutator {
				return m.
					OnRequestMethod(http.MethodGet).
					OnErr()
			},
			wantMethod: http.MethodPost,
		},
	}

	mutator := track.Mutator(
		func(tk *track.Track) {
			tk.Request.Method += " has been mutated"
		})

	for name, tc := range tt {
		name := name
		tc := tc

		t.Run(name, func(t *testing.T) {
			trk := track.NewTrack(
				&track.Request{
					Method: http.MethodPost,
				},
				&track.Response{
					Status: "BadStatus",
				},
				nil,
			)

			tc.mutatorOnFn(mutator)(trk)

			require.Equal(t, tc.wantMethod, trk.Request.Method)
		})
	}
}

func strPtr(s string) *string { return &s }
