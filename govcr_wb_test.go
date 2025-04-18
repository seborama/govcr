package govcr

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/seborama/govcr/v17/cassette/track"
	"github.com/seborama/govcr/v17/stats"
)

type GoVCRWBTestSuite struct {
	suite.Suite

	testServer *httptest.Server
}

func TestGoVCRWBTestSuite(t *testing.T) {
	suite.Run(t, new(GoVCRWBTestSuite))
}

func (ts *GoVCRWBTestSuite) SetupTest() {
	func() {
		// note to the wiser: adding a trailer causes the content to be chunked and
		// content-length will be -1 (i.e. unknown)
		counter := 0
		ts.testServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Trailer", "trailer_1")
			w.Header().Set("Header_1", "header_1_value")
			w.WriteHeader(http.StatusOK)
			counter++
			iQuery := r.URL.Query().Get("i")
			_, _ = fmt.Fprintf(w, "Hello, server responds '%d' to query '%s'", counter, iQuery)
			w.Header().Set("Trailer_1", "trailer_1_value")
		}))
	}()

	testServerClient := ts.testServer.Client()
	testServerClient.Timeout = 3 * time.Second
}

func (ts *GoVCRWBTestSuite) TearDownTest() {
	ts.testServer.Close()
}

type action int

const (
	actionKeepCassette = iota
	actionDeleteCassette
)

func (ts *GoVCRWBTestSuite) newVCR(cassetteName string, a action) *ControlPanel {
	if a == actionDeleteCassette {
		_ = os.Remove(cassetteName)
	}

	testServerClient := ts.testServer.Client()
	testServerClient.Timeout = 3 * time.Second

	return NewVCR(
		NewCassetteLoader(cassetteName),
		WithClient(testServerClient),
		// WithTrackRecordingMutators(trackMutator),
	)
}

func (ts *GoVCRWBTestSuite) TestRoundTrip_RequestMatcherDoesNotMutateState() {
	const cassetteName = "temp-fixtures/TestRoundTrip_RequestMatcherDoesNotMutateState.cassette.json"

	requestMatcherCount := 0

	reqMatchers := func(outcome bool) RequestMatchers {
		return RequestMatchers{
			// attempt to mutate state
			func(httpRequest, trackRequest *track.Request) bool {
				requestMatcherCount++

				httpRequest.Method = "test"
				httpRequest.URL = &url.URL{}
				httpRequest.Body = nil

				trackRequest.Method = "test"
				trackRequest.URL = &url.URL{}
				trackRequest.Body = nil

				return outcome
			},
		}
	}

	// 1st call - live
	vcr := ts.newVCR(cassetteName, actionDeleteCassette)
	vcr.SetLiveOnlyMode()                         // ensure we record one track so we can have a request matcher execution later (no track on cassette = no request matching)
	vcr.SetRequestMatchers(reqMatchers(false)...) // false: we want to attempt but not satisfy request matching so to check if the live request was altered

	req, err := http.NewRequest(http.MethodGet, ts.testServer.URL, http.NoBody)
	ts.Require().NoError(err)

	preRoundTripReq := track.CloneHTTPRequest(req)

	resp, err := vcr.HTTPClient().Do(req)
	ts.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	expectedStats := &stats.Stats{
		TotalTracks:    1,
		TracksLoaded:   0,
		TracksRecorded: 1,
		TracksPlayed:   0,
	}
	ts.Require().Equal(expectedStats, vcr.Stats())

	ts.Require().Equal(0, requestMatcherCount) // no track on cassette so no matching attempted by govcr

	postRoundTripReq := track.CloneHTTPRequest(req)
	ts.Require().Equal(preRoundTripReq, postRoundTripReq)

	// for simplification, we're using our own track.Response
	// we'll make the assumption that if that's well, the rest ought to be too.
	vcrResp := track.ToResponse(resp)
	ts.Equal("Hello, server responds '1' to query ''", string(vcrResp.Body))
	vcrResp.Body = nil

	// 2nd call - live
	vcr = ts.newVCR(cassetteName, actionKeepCassette)
	vcr.SetNormalMode()                           // ensure we attempt request matching
	vcr.SetRequestMatchers(reqMatchers(false)...) // false: we want to attempt but not satisfy request matching so to check if the live request was altered

	req, err = http.NewRequest(http.MethodGet, ts.testServer.URL, http.NoBody)
	ts.Require().NoError(err)

	resp2, err := vcr.HTTPClient().Do(req)
	ts.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	expectedStats = &stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   1,
		TracksRecorded: 1,
		TracksPlayed:   0,
	}
	ts.Require().Equal(expectedStats, vcr.Stats())

	ts.Require().Equal(1, requestMatcherCount) // an attempt to match the request should be made (albeit unsuccessful)

	postRoundTripReq = track.CloneHTTPRequest(req)
	ts.Require().Equal(preRoundTripReq, postRoundTripReq)

	vcrResp2 := track.ToResponse(resp2)
	ts.Equal("Hello, server responds '2' to query ''", string(vcrResp2.Body))
	vcrResp2.Body = nil

	ts.Require().Equal(vcrResp, vcrResp2)

	// 3rd call - replayed
	vcr = ts.newVCR(cassetteName, actionKeepCassette)
	vcr.SetOfflineMode()

	requestMatcherCount = 0
	vcr.SetRequestMatchers(reqMatchers(true)...) // true: satisfy request matching and force replay from track to ensure no mutation

	req, err = http.NewRequest(http.MethodGet, ts.testServer.URL, http.NoBody)
	ts.Require().NoError(err)

	resp3, err := vcr.HTTPClient().Do(req)
	ts.Require().NoError(err)
	defer func() { _ = resp3.Body.Close() }()

	expectedStats = &stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   2,
		TracksRecorded: 0,
		TracksPlayed:   1,
	}
	ts.Require().Equal(expectedStats, vcr.Stats())

	ts.Require().Equal(1, requestMatcherCount)

	postRoundTripReq = track.CloneHTTPRequest(req)
	ts.Require().Equal(preRoundTripReq, postRoundTripReq)

	// for simplification, we're using our own track.Response
	// we'll make the assumption that if that's well, the rest ought to be too.
	vcrResp3 := track.ToResponse(resp3)
	ts.Equal("Hello, server responds '1' to query ''", string(vcrResp3.Body))
	vcrResp3.Body = nil

	vcrResp.TLS, vcrResp3.TLS = nil, nil // TLS will not match fully
	ts.Require().Equal(vcrResp, vcrResp3)
}

func (ts *GoVCRWBTestSuite) TestRoundTrip_WithRecordingAndReplayingMutations() {
	const cassetteName = "temp-fixtures/TestRoundTrip_WithRecordingAndReplayingMutations.cassette.json"

	// 1st execution of set of calls
	vcr := ts.newVCR(cassetteName, actionDeleteCassette)
	vcr.SetRecordingMutators(trackMutator)

	ts.makeHTTPCallsWithSuccess(vcr.HTTPClient())
	expectedStats := &stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   0,
		TracksRecorded: 2,
		TracksPlayed:   0,
	}
	ts.Require().Equal(expectedStats, vcr.Stats())

	// load the cassette and verify contents has been mutated.
	vcr = ts.newVCR(cassetteName, actionKeepCassette)
	vcr.SetRecordingMutators(trackMutator)

	// note: remember that it usually doesn't make sense to modify the request in the replaying track mutator
	trackMutatorAgain := track.Mutator(
		func(trk *track.Track) {
			trk.Response.Header.Set("Trackrecordingmutatorheader", "headers have been mutated AGAIN AT PLAYBACK")
		})
	vcr.AddReplayingMutators(trackMutatorAgain)

	tracks := vcr.vcrTransport().cassette.Tracks
	for n := range tracks {
		ts.Require().Equal("this_query_key_has_been_mutated", tracks[n].Request.URL.Query().Get("mutated_query_key"), "track #%d", n)
		ts.Require().Equal("headers have been mutated", tracks[n].Response.Header.Get("Trackrecordingmutatorheader"), "track #%d", n)
	}

	// 2nd execution of set of calls (replayed)
	ts.replayHTTPCallsWithMutationsWithSuccess(vcr.HTTPClient(), "headers have been mutated AGAIN AT PLAYBACK")
	expectedStats = &stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   2,
		TracksRecorded: 0,
		TracksPlayed:   2,
	}
	ts.Equal(expectedStats, vcr.Stats())
}

func (ts *GoVCRWBTestSuite) TestRoundTrip_SavesAndReplaysMutatedTracksToCassette() {
	const cassetteName = "temp-fixtures/TestRoundTrip_SavesAndReplaysMutatedTracksToCassette.cassette.json"

	// 1st execution of set of calls
	vcr := ts.newVCR(cassetteName, actionDeleteCassette)
	vcr.SetRecordingMutators(trackMutator)

	ts.makeHTTPCallsWithSuccess(vcr.HTTPClient())
	expectedStats := &stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   0,
		TracksRecorded: 2,
		TracksPlayed:   0,
	}
	ts.Require().Equal(expectedStats, vcr.Stats())

	// load the cassette and verify contents has been mutated.
	vcr = ts.newVCR(cassetteName, actionKeepCassette)
	vcr.SetRecordingMutators(trackMutator)

	tracks := vcr.vcrTransport().cassette.Tracks
	for n := range tracks {
		ts.Require().Equal("this_query_key_has_been_mutated", tracks[n].Request.URL.Query().Get("mutated_query_key"), "track #%d", n)
		ts.Require().Equal("headers have been mutated", tracks[n].Response.Header.Get("Trackrecordingmutatorheader"), "track #%d", n)
	}

	// 2nd execution of set of calls (replayed)
	ts.replayHTTPCallsWithMutationsWithSuccess(vcr.HTTPClient(), "headers have been mutated")
	expectedStats = &stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   2,
		TracksRecorded: 0,
		TracksPlayed:   2,
	}
	ts.Equal(expectedStats, vcr.Stats())
}

func (ts *GoVCRWBTestSuite) makeHTTPCallsWithSuccess(httpClient *http.Client) {
	for i := 1; i <= 2; i++ {
		req, err := http.NewRequest(http.MethodGet, ts.testServer.URL+fmt.Sprintf("?i=%d", i), http.NoBody)
		ts.Require().NoError(err)
		req.Header.Add("Header", "value")
		req.SetBasicAuth("not_a_username", "not_a_password")

		resp, err := httpClient.Do(req)
		ts.Require().NoError(err)

		// read body first because the server is passing Trailers in http.Response.
		bodyBytes, err := io.ReadAll(resp.Body)
		ts.Require().NoError(err)
		_ = resp.Body.Close()
		ts.Equal(fmt.Sprintf("Hello, server responds '%d' to query '%d'", i, i), string(bodyBytes))

		ts.Equal("200 OK", resp.Status)
		ts.Equal(http.StatusOK, resp.StatusCode)

		ts.Empty(resp.Header.Get("Content-Length"))
		ts.EqualValues(-1, resp.ContentLength)
		ts.Equal("text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
		ts.Equal("header_1_value", resp.Header.Get("Header_1"))
		ts.Require().Empty(resp.Header.Get("Trackrecordingmutatorheader")) // the header is injected, not present in the live traffic

		ts.Len(resp.Trailer, 1)
		ts.Equal("trailer_1_value", resp.Trailer.Get("Trailer_1"))

		ts.NotNil(resp.Request)
		ts.NotNil(resp.TLS)
	}
}

func (ts *GoVCRWBTestSuite) replayHTTPCallsWithMutationsWithSuccess(httpClient *http.Client, trackRecordingMutatorHeaderValue string) {
	for i := 1; i <= 2; i++ {
		req, err := http.NewRequest(http.MethodGet, ts.testServer.URL+fmt.Sprintf("?i=%d", i), http.NoBody)
		ts.Require().NoError(err)

		// update our request in line with the previous recording mutations that took place.
		// not doing so would prevent matching our request against the (mutated) cassette.
		q := req.URL.Query()
		q.Set("mutated_query_key", "this_query_key_has_been_mutated")
		req.URL.RawQuery = q.Encode()

		req.Header.Add("Header", "value")
		req.SetBasicAuth("not_a_username", "not_a_password")

		resp, err := httpClient.Do(req)
		ts.Require().NoError(err)

		ts.Require().Equal("200 OK", resp.Status)
		ts.Require().Equal(http.StatusOK, resp.StatusCode)

		ts.Empty(resp.Header.Get("Content-Length"))
		ts.EqualValues(-1, resp.ContentLength)
		ts.Equal("text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
		ts.Equal("header_1_value", resp.Header.Get("Header_1"))
		ts.Require().Equal(trackRecordingMutatorHeaderValue, resp.Header.Get("Trackrecordingmutatorheader"))

		ts.Len(resp.Trailer, 1)
		ts.Equal("trailer_1_value", resp.Trailer.Get("Trailer_1"))

		bodyBytes, err := io.ReadAll(resp.Body)
		ts.Require().NoError(err)
		_ = resp.Body.Close()
		ts.Require().Equal(fmt.Sprintf("Hello, server responds '%d' to query '%d'", i, i), string(bodyBytes))

		ts.Require().NotNil(resp.Request)
		ts.Require().NotNil(resp.TLS)
	}
}

func trackMutator(trk *track.Track) {
	q := trk.Request.URL.Query()
	q.Set("mutated_query_key", "this_query_key_has_been_mutated")
	trk.Request.URL.RawQuery = q.Encode()

	trk.Response.Header.Add("Trackrecordingmutatorheader", "headers have been mutated")
	trk.Response.Header.Del("Date") // to avoid matching issues
}
