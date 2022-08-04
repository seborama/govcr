package govcr

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/seborama/govcr/v6/cassette/track"
	"github.com/seborama/govcr/v6/stats"
)

type GoVCRWBTestSuite struct {
	suite.Suite

	vcr          *ControlPanel
	testServer   *httptest.Server
	cassetteName string
}

func TestGoVCRWBTestSuite(t *testing.T) {
	suite.Run(t, new(GoVCRWBTestSuite))
}

func (suite *GoVCRWBTestSuite) SetupTest() {
	func() {
		// note to the wiser: adding a trailer causes the content to be chunked and
		// content-length will be -1 (i.e. unknown)
		counter := 0
		suite.testServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Trailer", "trailer_1")
			w.Header().Set("header_1", "header_1_value")
			w.WriteHeader(http.StatusOK)
			counter++
			iQuery := r.URL.Query().Get("i")
			_, _ = fmt.Fprintf(w, "Hello, server responds '%d' to query '%s'", counter, iQuery)
			w.Header().Set("trailer_1", "trailer_1_value")
		}))
	}()

	testServerClient := suite.testServer.Client()
	testServerClient.Timeout = 3 * time.Second

	// example mutator, mutation is not too intrusive to allow replaying correctly.
	// for instance, when an Err is injected, the response is set to nil on replay, as per
	// go's HTTP client design.
	trackMutator := track.Mutator(
		func(trk *track.Track) {
			q := trk.Request.URL.Query()
			q.Set("mutated_query_key", "this_query_key_has_been_mutated")
			trk.Request.URL.RawQuery = q.Encode()

			trk.Response.Header.Add("TrackRecordingMutatorHeader", "headers have been mutated")
		})

	suite.vcr = NewVCR(
		WithClient(testServerClient),
		WithTrackRecordingMutators(trackMutator),
	)
	suite.cassetteName = "temp-fixtures/TestRoundTrip_SavesMutatedCassetteTracks.cassette.json"
	_ = os.Remove(suite.cassetteName)
}

func (suite *GoVCRWBTestSuite) TearDownTest() {
	_ = os.Remove(suite.cassetteName)
}

func (suite *GoVCRWBTestSuite) TestRoundTrip_RequestMatcherDoesNotMutateState() {
	suite.vcr.ClearRecordingMutators() // mutators by definition cannot change the live request / response, only the track
	suite.vcr.ClearReplayingMutators() // mutators by definition cannot change the live request / response, only the track

	suite.vcr.SetLiveOnlyMode(true) // ensure we record one track so we can have a request matcher execution later (no track on cassette = no request matching)

	requestMatcherCount := 0

	suite.vcr.SetRequestMatcher(NewBlankRequestMatcher(
		WithRequestMatcherFunc(
			// attempt to mutate state
			func(httpRequest, trackRequest *track.Request) bool {
				requestMatcherCount++

				httpRequest.Method = "test"
				httpRequest.URL = &url.URL{}
				httpRequest.Body = nil

				trackRequest.Method = "test"
				trackRequest.URL = &url.URL{}
				trackRequest.Body = nil

				httpRequest = &track.Request{}
				trackRequest = &track.Request{}

				return false // we want to attempt but not satisfy request matching so to check if the live request was altered
			},
		),
	))

	// 1st call - live
	err := suite.vcr.LoadCassette(suite.cassetteName)
	suite.NoError(err)

	req, err := http.NewRequest(http.MethodGet, suite.testServer.URL, nil)
	suite.Require().NoError(err)

	preRoundTripReq := track.CloneHTTPRequest(req)

	resp, err := suite.vcr.HTTPClient().Do(req)
	suite.Require().NoError(err)

	expectedStats := &stats.Stats{
		TotalTracks:    1,
		TracksLoaded:   0,
		TracksRecorded: 1,
		TracksPlayed:   0,
	}
	suite.Require().EqualValues(expectedStats, suite.vcr.Stats())

	suite.Require().Equal(0, requestMatcherCount) // no track on cassette so no matching attempted by govcr

	postRoundTripReq := track.CloneHTTPRequest(req)
	suite.Require().EqualValues(preRoundTripReq, postRoundTripReq)

	// for simplification, we're using our own track.Response
	// we'll make the assumption that if that's well, the rest ought to be too.
	vcrResp := track.ToResponse(resp)
	suite.Assert().Equal("Hello, server responds '1' to query ''", string(vcrResp.Body))
	vcrResp.Body = nil

	// 2nd call - live
	suite.vcr.EjectCassette() // reset cassette state so to allow track replay (newly recorded tracks are marked at replayed)
	err = suite.vcr.LoadCassette(suite.cassetteName)
	suite.Require().NoError(err)
	suite.vcr.SetLiveOnlyMode(false) // ensure we attempt request matching

	req, err = http.NewRequest(http.MethodGet, suite.testServer.URL, nil)
	suite.Require().NoError(err)

	resp2, err := suite.vcr.HTTPClient().Do(req)
	suite.Require().NoError(err)

	expectedStats = &stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   1,
		TracksRecorded: 1,
		TracksPlayed:   0,
	}
	suite.Require().EqualValues(expectedStats, suite.vcr.Stats())

	suite.Require().Equal(1, requestMatcherCount) // an attempt to match the request should be made (albeit unsuccessful)

	postRoundTripReq = track.CloneHTTPRequest(req)
	suite.Require().EqualValues(preRoundTripReq, postRoundTripReq)

	vcrResp2 := track.ToResponse(resp2)
	suite.Assert().Equal("Hello, server responds '2' to query ''", string(vcrResp2.Body))
	vcrResp2.Body = nil

	suite.Require().EqualValues(vcrResp, vcrResp2)

	// 3rd call - replayed
	suite.vcr.EjectCassette() // reset cassette state so to allow track replay (newly recorded tracks are marked at replayed)
	err = suite.vcr.LoadCassette(suite.cassetteName)
	suite.Require().NoError(err)
	suite.vcr.SetLiveOnlyMode(false)
	suite.vcr.SetOfflineMode(true)

	requestMatcherCount = 0
	suite.vcr.SetRequestMatcher(NewBlankRequestMatcher(
		WithRequestMatcherFunc(
			// attempt to mutate state
			func(httpRequest, trackRequest *track.Request) bool {
				requestMatcherCount++

				httpRequest.Method = "test"
				httpRequest.URL = &url.URL{}
				httpRequest.Body = nil

				trackRequest.Method = "test"
				trackRequest.URL = &url.URL{}
				trackRequest.Body = nil

				httpRequest = &track.Request{}
				trackRequest = &track.Request{}

				return true // satisfy request matching and force replay from track to ensure no mutation
			},
		),
	))

	req, err = http.NewRequest(http.MethodGet, suite.testServer.URL, nil)
	suite.Require().NoError(err)

	resp3, err := suite.vcr.HTTPClient().Do(req)
	suite.Require().NoError(err)

	expectedStats = &stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   2,
		TracksRecorded: 0,
		TracksPlayed:   1,
	}
	suite.Require().EqualValues(expectedStats, suite.vcr.Stats())

	suite.Require().Equal(1, requestMatcherCount)

	postRoundTripReq = track.CloneHTTPRequest(req)
	suite.Require().EqualValues(preRoundTripReq, postRoundTripReq)

	// for simplification, we're using our own track.Response
	// we'll make the assumption that if that's well, the rest ought to be too.
	vcrResp3 := track.ToResponse(resp3)
	suite.Assert().Equal("Hello, server responds '1' to query ''", string(vcrResp3.Body))
	vcrResp3.Body = nil

	vcrResp.TLS, vcrResp3.TLS = nil, nil // TLS will not match fully
	suite.Require().EqualValues(vcrResp, vcrResp3)
}

func (suite *GoVCRWBTestSuite) TestRoundTrip_WithRecordingAndReplayingMutations() {
	err := suite.vcr.LoadCassette(suite.cassetteName)
	suite.NoError(err)

	// 1st execution of set of calls
	actualStats := suite.makeHTTPCalls_WithSuccess()
	expectedStats := stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   0,
		TracksRecorded: 2,
		TracksPlayed:   0,
	}
	suite.Require().EqualValues(expectedStats, actualStats)

	// load the cassette and verify contents has been mutated.
	err = suite.vcr.LoadCassette(suite.cassetteName)
	suite.NoError(err)

	// note: remember that it usually doesn't make sense to modify the request in the replaying track mutator
	trackMutator := track.Mutator(
		func(trk *track.Track) {
			trk.Response.Header.Set("TrackRecordingMutatorHeader", "headers have been mutated AGAIN AT PLAYBACK")
		})
	suite.vcr.AddReplayingMutators(trackMutator)

	for trackNum, trk := range suite.vcr.vcrTransport().cassette.Tracks {
		suite.Require().EqualValues("this_query_key_has_been_mutated", trk.Request.URL.Query().Get("mutated_query_key"), "track #%d", trackNum)
		suite.Require().EqualValues("headers have been mutated", trk.Response.Header.Get("TrackRecordingMutatorHeader"), "track #%d", trackNum)
	}

	// 2nd execution of set of calls (replayed)
	actualStats = suite.replayHTTPCalls_WithMutations_WithSuccess("headers have been mutated AGAIN AT PLAYBACK")
	expectedStats = stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   2,
		TracksRecorded: 0,
		TracksPlayed:   2,
	}
	suite.EqualValues(expectedStats, actualStats)
}

func (suite *GoVCRWBTestSuite) TestRoundTrip_SavesAndReplaysMutatedTracksToCassette() {
	err := suite.vcr.LoadCassette(suite.cassetteName)
	suite.NoError(err)

	// 1st execution of set of calls
	actualStats := suite.makeHTTPCalls_WithSuccess()
	expectedStats := stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   0,
		TracksRecorded: 2,
		TracksPlayed:   0,
	}
	suite.Require().EqualValues(expectedStats, actualStats)

	// load the cassette and verify contents has been mutated.
	err = suite.vcr.LoadCassette(suite.cassetteName)
	suite.NoError(err)

	for trackNum, trk := range suite.vcr.vcrTransport().cassette.Tracks {
		suite.Require().EqualValues("this_query_key_has_been_mutated", trk.Request.URL.Query().Get("mutated_query_key"), "track #%d", trackNum)
		suite.Require().EqualValues("headers have been mutated", trk.Response.Header.Get("TrackRecordingMutatorHeader"), "track #%d", trackNum)
	}

	// 2nd execution of set of calls (replayed)
	actualStats = suite.replayHTTPCalls_WithMutations_WithSuccess("headers have been mutated")
	expectedStats = stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   2,
		TracksRecorded: 0,
		TracksPlayed:   2,
	}
	suite.EqualValues(expectedStats, actualStats)
}

func (suite *GoVCRWBTestSuite) makeHTTPCalls_WithSuccess() stats.Stats {
	for i := 1; i <= 2; i++ {
		req, err := http.NewRequest(http.MethodGet, suite.testServer.URL+fmt.Sprintf("?i=%d", i), nil)
		suite.Require().NoError(err)
		req.Header.Add("header", "value")
		req.SetBasicAuth("not_a_username", "not_a_password")

		resp, err := suite.vcr.HTTPClient().Do(req)
		suite.Require().NoError(err)

		// read body first because the server is passing Trailers in http.Response.
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		suite.Require().NoError(err)
		_ = resp.Body.Close()
		suite.Assert().Equal(fmt.Sprintf("Hello, server responds '%d' to query '%d'", i, i), string(bodyBytes))

		suite.Assert().Equal("200 OK", resp.Status)
		suite.Assert().Equal(http.StatusOK, resp.StatusCode)

		suite.Assert().EqualValues("", resp.Header.Get("Content-Length"))
		suite.Assert().EqualValues(-1, resp.ContentLength)
		suite.Assert().EqualValues("text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
		suite.Assert().EqualValues("header_1_value", resp.Header.Get("header_1"))
		suite.Assert().NotEmpty(resp.Header.Get("Date"))
		suite.Require().EqualValues("", resp.Header.Get("TrackRecordingMutatorHeader")) // the header is injected, not present in the live traffic

		suite.Assert().Len(resp.Trailer, 1)
		suite.Assert().EqualValues("trailer_1_value", resp.Trailer.Get("trailer_1"))

		suite.Assert().NotNil(resp.Request)
		suite.Assert().NotNil(resp.TLS)
	}

	actualStats := *suite.vcr.Stats()
	suite.vcr.EjectCassette()

	return actualStats
}

func (suite *GoVCRWBTestSuite) replayHTTPCalls_WithMutations_WithSuccess(trackRecordingMutatorHeaderValue string) stats.Stats {
	for i := 1; i <= 2; i++ {
		req, err := http.NewRequest(http.MethodGet, suite.testServer.URL+fmt.Sprintf("?i=%d", i), nil)
		suite.Require().NoError(err)

		// update our request inline with the previous recording mutations that took place.
		// not doing so would prevent matching our request against the (mutated) cassette.
		q := req.URL.Query()
		q.Set("mutated_query_key", "this_query_key_has_been_mutated")
		req.URL.RawQuery = q.Encode()

		req.Header.Add("header", "value")
		req.SetBasicAuth("not_a_username", "not_a_password")

		resp, err := suite.vcr.HTTPClient().Do(req)
		suite.Require().NoError(err)

		suite.Require().Equal("200 OK", resp.Status)
		suite.Require().Equal(http.StatusOK, resp.StatusCode)

		suite.Assert().EqualValues("", resp.Header.Get("Content-Length"))
		suite.Assert().EqualValues(-1, resp.ContentLength)
		suite.Assert().EqualValues("text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
		suite.Assert().EqualValues("header_1_value", resp.Header.Get("header_1"))
		suite.Assert().NotEmpty(resp.Header.Get("Date"))
		suite.Require().EqualValues(trackRecordingMutatorHeaderValue, resp.Header.Get("TrackRecordingMutatorHeader"))

		suite.Assert().Len(resp.Trailer, 1)
		suite.Assert().EqualValues("trailer_1_value", resp.Trailer.Get("trailer_1"))

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		suite.Require().NoError(err)
		_ = resp.Body.Close()
		suite.Require().Equal(fmt.Sprintf("Hello, server responds '%d' to query '%d'", i, i), string(bodyBytes))

		suite.Require().NotNil(resp.Request)
		suite.Require().NotNil(resp.TLS)
	}

	actualStats := *suite.vcr.Stats()
	suite.vcr.EjectCassette()

	return actualStats
}
