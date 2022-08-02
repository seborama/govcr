package govcr

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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

func (suite *GoVCRWBTestSuite) TestRoundTrip_DoesNotChangeLiveRequestOrResponse() {
	suite.vcr.SetRequestMatcher(NewBlankRequestMatcher())
	suite.Fail("implement me")
	// TODO: create a VCR with WithTrackRecordingMutators and WithTrackReplayingMutators
	//       and confirm that both the live request and response remain un-mutated.
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
