package track_test

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/v6/cassette/track"
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

func Test_Mutator_Or(t *testing.T) {
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
	unitMutator.Or(pFalse, pTrue)(nil)
	require.Equal(t, 0, mutatorCallCounter)

	mutatorCallCounter = 0
	unitMutator.Or(pFalse, pTrue)(trk)
	require.Equal(t, 1, mutatorCallCounter)

	mutatorCallCounter = 0
	unitMutator.Or(pFalse, pFalse)(trk)
	require.Equal(t, 0, mutatorCallCounter)

	mutatorCallCounter = 0
	unitMutator.Or(pTrue, pTrue)(trk)
	require.Equal(t, 1, mutatorCallCounter)
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
			tk.Request.Method = tk.Request.Method + " has been mutated"
			tk.Response.Status = tk.Response.Status + " has been mutated"
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
			tk.Request.Method = tk.Request.Method + " has been mutated"
			tk.Response.Status = tk.Response.Status + " has been mutated"
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
			tk.Request.Method = tk.Request.Method + " has been mutated"
			tk.Response.Status = tk.Response.Status + " has been mutated"
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
			tk.Request.Method = tk.Request.Method + " has been mutated"
			tk.Response.Status = tk.Response.Status + " has been mutated"
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
	unitMutator := track.RequestAddHeaderValue("key-1", "value-1")

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
	unitMutator := track.RequestAddHeaderValue("key-1", "value-1")

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
	unitMutator := track.RequestDeleteHeaderKeys("other", "key-a")

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
	unitMutator := track.RequestDeleteHeaderKeys("other", "key-a")

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
	unitMutator := track.RequestChangeBody(
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

func TestRequestTransferHeaderKeys_NilTrack(t *testing.T) {
	var trk *track.Track
	track.RequestTransferHeaderKeys("unit-key-1", "unit-value-1")(trk)
	assert.Nil(t, trk)
}

func TestRequestTransferTrailerKeys_NilTrack(t *testing.T) {
	var trk *track.Track
	track.RequestTransferTrailerKeys("unit-key-1", "unit-value-1")(trk)
	assert.Nil(t, trk)
}

func TestResponseTransferHeaderKeys_NilTrack(t *testing.T) {
	var trk *track.Track
	track.ResponseTransferHeaderKeys("unit-key-1", "unit-value-1")(trk)
	assert.Nil(t, trk)
}

func TestResponseTransferTrailerKeys_NilTrack(t *testing.T) {
	var trk *track.Track
	track.ResponseTransferTrailerKeys("unit-key-1", "unit-value-1")(trk)
	assert.Nil(t, trk)
}

func Test_Mutator_RequestTransferHeaderKeys(t *testing.T) {
	tt := map[string]struct {
		reqHeader      http.Header
		respHeader     http.Header
		wantReqHeader  http.Header
		wantRespHeader http.Header
	}{
		"nil request and nil response headers": {
			reqHeader:      nil,
			respHeader:     nil,
			wantReqHeader:  nil,
			wantRespHeader: nil,
		},
		"nil request and blank response header": {
			reqHeader:      nil,
			respHeader:     http.Header{},
			wantReqHeader:  nil,
			wantRespHeader: http.Header{},
		},
		"blank request and nil response header": {
			reqHeader:      http.Header{},
			respHeader:     nil,
			wantReqHeader:  http.Header{},
			wantRespHeader: nil,
		},
		"blank request and blank response header": {
			reqHeader:      http.Header{},
			respHeader:     http.Header{},
			wantReqHeader:  http.Header{},
			wantRespHeader: http.Header{},
		},
		"nil request and eligible response header": {
			reqHeader:      nil,
			respHeader:     func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantReqHeader:  func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantRespHeader: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
		},
		"blank request and eligible response header": {
			reqHeader:      http.Header{},
			respHeader:     func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantReqHeader:  func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantRespHeader: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
		},
		"eligible response header with request containing other data": {
			reqHeader:  func() http.Header { h := http.Header{}; h.Set("unit-key-a", "unit-value-a"); return h }(),
			respHeader: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantReqHeader: func() http.Header {
				h := http.Header{}
				h.Set("unit-key-a", "unit-value-a")
				h.Add("unit-key-1", "unit-value-1")
				return h
			}(),
			wantRespHeader: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
		},
		"eligible response header with request already containing the transfer data": {
			reqHeader:  func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respHeader: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantReqHeader: func() http.Header {
				h := http.Header{}
				h.Set("unit-key-1", "unit-value-1")
				h.Add("unit-key-1", "unit-value-1")
				return h
			}(),
			wantRespHeader: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
		},
	}

	for name, tc := range tt {
		name := name
		tc := tc

		t.Run(name, func(t *testing.T) {
			trk := track.NewTrack(
				&track.Request{Header: tc.reqHeader},
				&track.Response{Header: tc.respHeader},
				nil,
			)

			track.RequestTransferHeaderKeys("unit-key-1", "unit-value-1")(trk)

			assert.Equal(t, tc.wantReqHeader, trk.Request.Header)
			assert.Equal(t, tc.wantRespHeader, trk.Response.Header)
		})
	}
}

func Test_Mutator_RequestTransferTrailerKeys(t *testing.T) {
	tt := map[string]struct {
		reqTrailer      http.Header
		respTrailer     http.Header
		wantReqTrailer  http.Header
		wantRespTrailer http.Header
	}{
		"nil request and nil response trailers": {
			reqTrailer:      nil,
			respTrailer:     nil,
			wantReqTrailer:  nil,
			wantRespTrailer: nil,
		},
		"nil request and blank response trailer": {
			reqTrailer:      nil,
			respTrailer:     http.Header{},
			wantReqTrailer:  nil,
			wantRespTrailer: http.Header{},
		},
		"blank request and nil response trailer": {
			reqTrailer:      http.Header{},
			respTrailer:     nil,
			wantReqTrailer:  http.Header{},
			wantRespTrailer: nil,
		},
		"blank request and blank response trailer": {
			reqTrailer:      http.Header{},
			respTrailer:     http.Header{},
			wantReqTrailer:  http.Header{},
			wantRespTrailer: http.Header{},
		},
		"nil request and eligible response trailer": {
			reqTrailer:      nil,
			respTrailer:     func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantReqTrailer:  func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantRespTrailer: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
		},
		"blank request and eligible response trailer": {
			reqTrailer:      http.Header{},
			respTrailer:     func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantReqTrailer:  func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantRespTrailer: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
		},
		"eligible response trailer with request containing other data": {
			reqTrailer:  func() http.Header { h := http.Header{}; h.Set("unit-key-a", "unit-value-a"); return h }(),
			respTrailer: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantReqTrailer: func() http.Header {
				h := http.Header{}
				h.Set("unit-key-a", "unit-value-a")
				h.Add("unit-key-1", "unit-value-1")
				return h
			}(),
			wantRespTrailer: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
		},
		"eligible response trailer with request already containing the transfer data": {
			reqTrailer:  func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respTrailer: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantReqTrailer: func() http.Header {
				h := http.Header{}
				h.Set("unit-key-1", "unit-value-1")
				h.Add("unit-key-1", "unit-value-1")
				return h
			}(),
			wantRespTrailer: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
		},
	}

	for name, tc := range tt {
		name := name
		tc := tc

		t.Run(name, func(t *testing.T) {
			trk := track.NewTrack(
				&track.Request{Trailer: tc.reqTrailer},
				&track.Response{Trailer: tc.respTrailer},
				nil,
			)

			track.RequestTransferTrailerKeys("unit-key-1", "unit-value-1")(trk)

			assert.Equal(t, tc.wantReqTrailer, trk.Request.Trailer)
			assert.Equal(t, tc.wantRespTrailer, trk.Response.Trailer)
		})
	}
}

func Test_Mutator_ResponseTransferHeaderKeys(t *testing.T) {
	tt := map[string]struct {
		reqHeader      http.Header
		respHeader     http.Header
		wantReqHeader  http.Header
		wantRespHeader http.Header
	}{
		"nil request and nil response headers": {
			reqHeader:      nil,
			respHeader:     nil,
			wantReqHeader:  nil,
			wantRespHeader: nil,
		},
		"nil request and blank response header": {
			reqHeader:      nil,
			respHeader:     http.Header{},
			wantReqHeader:  nil,
			wantRespHeader: http.Header{},
		},
		"blank request and nil response header": {
			reqHeader:      http.Header{},
			respHeader:     nil,
			wantReqHeader:  http.Header{},
			wantRespHeader: nil,
		},
		"blank request and blank response header": {
			reqHeader:      http.Header{},
			respHeader:     http.Header{},
			wantReqHeader:  http.Header{},
			wantRespHeader: http.Header{},
		},
		"nil response and eligible request header": {
			reqHeader:      func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respHeader:     nil,
			wantReqHeader:  func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantRespHeader: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
		},
		"blank response and eligible request header": {
			reqHeader:      func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respHeader:     http.Header{},
			wantReqHeader:  func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantRespHeader: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
		},
		"eligible request header with response containing other data": {
			reqHeader:     func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respHeader:    func() http.Header { h := http.Header{}; h.Set("unit-key-a", "unit-value-a"); return h }(),
			wantReqHeader: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantRespHeader: func() http.Header {
				h := http.Header{}
				h.Set("unit-key-a", "unit-value-a")
				h.Add("unit-key-1", "unit-value-1")
				return h
			}(),
		},
		"eligible request header with response already containing the transfer data": {
			reqHeader:     func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respHeader:    func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantReqHeader: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
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
				&track.Request{Header: tc.reqHeader},
				&track.Response{Header: tc.respHeader},
				nil,
			)

			track.ResponseTransferHeaderKeys("unit-key-1", "unit-value-1")(trk)

			assert.Equal(t, tc.wantReqHeader, trk.Request.Header)
			assert.Equal(t, tc.wantRespHeader, trk.Response.Header)
		})
	}
}

func Test_Mutator_ResponseTransferTrailerKeys(t *testing.T) {
	tt := map[string]struct {
		reqTrailer      http.Header
		respTrailer     http.Header
		wantReqTrailer  http.Header
		wantRespTrailer http.Header
	}{
		"nil request and nil response trailers": {
			reqTrailer:      nil,
			respTrailer:     nil,
			wantReqTrailer:  nil,
			wantRespTrailer: nil,
		},
		"nil request and blank response trailer": {
			reqTrailer:      nil,
			respTrailer:     http.Header{},
			wantReqTrailer:  nil,
			wantRespTrailer: http.Header{},
		},
		"blank request and nil response trailer": {
			reqTrailer:      http.Header{},
			respTrailer:     nil,
			wantReqTrailer:  http.Header{},
			wantRespTrailer: nil,
		},
		"blank request and blank response trailer": {
			reqTrailer:      http.Header{},
			respTrailer:     http.Header{},
			wantReqTrailer:  http.Header{},
			wantRespTrailer: http.Header{},
		},
		"nil response and eligible request trailer": {
			reqTrailer:      func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respTrailer:     nil,
			wantReqTrailer:  func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantRespTrailer: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
		},
		"blank response and eligible request trailer": {
			reqTrailer:      func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respTrailer:     http.Header{},
			wantReqTrailer:  func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantRespTrailer: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
		},
		"eligible request trailer with response containing other data": {
			reqTrailer:     func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respTrailer:    func() http.Header { h := http.Header{}; h.Set("unit-key-a", "unit-value-a"); return h }(),
			wantReqTrailer: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantRespTrailer: func() http.Header {
				h := http.Header{}
				h.Set("unit-key-a", "unit-value-a")
				h.Add("unit-key-1", "unit-value-1")
				return h
			}(),
		},
		"eligible request trailer with response already containing the transfer data": {
			reqTrailer:     func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			respTrailer:    func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
			wantReqTrailer: func() http.Header { h := http.Header{}; h.Set("unit-key-1", "unit-value-1"); return h }(),
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
				&track.Request{Trailer: tc.reqTrailer},
				&track.Response{Trailer: tc.respTrailer},
				nil,
			)

			track.ResponseTransferTrailerKeys("unit-key-1", "unit-value-1")(trk)

			assert.Equal(t, tc.wantReqTrailer, trk.Request.Trailer)
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
			tk.Request.Method = tk.Request.Method + " has been mutated"
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
