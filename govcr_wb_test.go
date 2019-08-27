package govcr

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/seborama/govcr/cassette/track"
	"github.com/seborama/govcr/stats"
)

type GoVCRWBTestSuite struct {
	suite.Suite

	vcr          *ControlPanel
	testServer   *httptest.Server
	cassetteName string
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GoVCRWBTestSuite))
}

func (suite *GoVCRWBTestSuite) SetupTest() {
	func() {
		counter := 0
		suite.testServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			counter++
			iQuery := r.URL.Query().Get("i")
			_, _ = fmt.Fprintf(w, "Hello, server responds '%d' to query '%s'", counter, iQuery)
		}))
	}()

	testServerClient := suite.testServer.Client()
	testServerClient.Timeout = 3 * time.Second

	// example mutator, mutation is not too intrusive to allow replaying correctly.
	// for instance, when an Err is injected, the response is set to nil on replay, as per
	// go's HTTP client design.
	aTrackMutator := TrackMutator(
		func(trk *track.Track) {
			q := trk.Request.URL.Query()
			q.Set("mutated_query_key", "this_query_key_has_been_mutated")
			trk.Request.URL.RawQuery = q.Encode()

			trk.Response.Header.Add("TrackRecordingMutatorHeader", "headers have been mutated")
		})

	suite.vcr = NewVCR(WithClient(testServerClient), WithTrackRecordingMutators(aTrackMutator))
	suite.cassetteName = "govcr-fixtures/TestRoundTrip_SavesMutatedCassetteTracks.cassette"
	_ = os.Remove(suite.cassetteName)
}

func (suite *GoVCRWBTestSuite) TearDownTest() {
	_ = os.Remove(suite.cassetteName)
}

func (suite *GoVCRWBTestSuite) TestRoundTrip_DoesNotChangeLiveRequestOrResponse() {
	panic("implement me")
	// TODO: create a VCR with WithTrackRecordingMutators and WithTrackReplayingMutators
	//       and confirm that both the live request and response remain un-mutated.
}

func (suite *GoVCRWBTestSuite) TestRoundTrip_WithRecordingAndReplayingMutations() {
	panic("implement me")
	// TODO: create a VCR with WithTrackReplayingMutators
	//       and confirm that the replayed request and response are mutated correctly.
}

func (suite *GoVCRWBTestSuite) TestRoundTrip_SavesAndReplaysMutatedTracksToCassette() {
	err := suite.vcr.LoadCassette(suite.cassetteName)
	suite.NoError(err)

	// 1st execution of set of calls
	actualStats := suite.makeHTTPCalls_WithSuccess()
	expectedStats := stats.Stats{
		TracksLoaded:   0,
		TracksRecorded: 2,
		TracksPlayed:   0,
	}
	suite.Require().EqualValues(expectedStats, actualStats)

	// load the cassette and verify contents has been mutated.
	err = suite.vcr.LoadCassette(suite.cassetteName)
	suite.NoError(err)

	for trackNum, aTrack := range suite.vcr.vcrTransport().cassette.Tracks {
		suite.Require().EqualValues("this_query_key_has_been_mutated", aTrack.Request.URL.Query().Get("mutated_query_key"), "track #%d", trackNum)
		suite.Require().EqualValues("headers have been mutated", aTrack.Response.Header.Get("TrackRecordingMutatorHeader"), "track #%d", trackNum)
	}

	// 2nd execution of set of calls (replayed)
	actualStats = suite.replayHTTPCalls_WithMutations_WithSuccess()
	expectedStats = stats.Stats{
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

		resp, err := suite.vcr.Player().Do(req)
		suite.Require().NoError(err)

		suite.Require().Equal("200 OK", resp.Status)
		suite.Require().Equal(http.StatusOK, resp.StatusCode)
		suite.Require().EqualValues(strconv.Itoa(38+len(strconv.Itoa(i))), resp.Header.Get("Content-Length"))
		suite.Require().EqualValues("text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
		suite.Require().NotEmpty(resp.Header.Get("Date"))
		suite.Require().EqualValues(resp.Trailer, http.Header(nil))

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		suite.Require().NoError(err)
		_ = resp.Body.Close()
		suite.Require().Equal(fmt.Sprintf("Hello, server responds '%d' to query '%d'", i, i), string(bodyBytes))

		suite.Require().Equal(int64(38+len(strconv.Itoa(i))), resp.ContentLength)
		suite.Require().NotNil(resp.Request)
		suite.Require().NotNil(resp.TLS)
	}

	suite.Require().EqualValues(2, suite.vcr.NumberOfTracks())

	actualStats := *suite.vcr.Stats()
	suite.vcr.EjectCassette()

	return actualStats
}

func (suite *GoVCRWBTestSuite) replayHTTPCalls_WithMutations_WithSuccess() stats.Stats {
	for i := 1; i <= 2; i++ {
		req, err := http.NewRequest(http.MethodGet, suite.testServer.URL+fmt.Sprintf("?i=%d", i), nil)
		suite.Require().NoError(err)

		// manually modify the request inline with the previous mutations that took place.
		// not doing so would prevent matching our request against the (mutated) cassette.
		q := req.URL.Query()
		q.Set("mutated_query_key", "this_query_key_has_been_mutated")
		req.URL.RawQuery = q.Encode()

		req.Header.Add("header", "value")
		req.SetBasicAuth("not_a_username", "not_a_password")

		resp, err := suite.vcr.Player().Do(req)
		suite.Require().NoError(err)

		suite.Require().Equal("200 OK", resp.Status)
		suite.Require().Equal(http.StatusOK, resp.StatusCode)
		suite.Require().EqualValues(strconv.Itoa(38+len(strconv.Itoa(i))), resp.Header.Get("Content-Length"))
		suite.Require().EqualValues("text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
		suite.Require().NotEmpty(resp.Header.Get("Date"))
		suite.Require().EqualValues(resp.Trailer, http.Header(nil))
		suite.Require().EqualValues("headers have been mutated", resp.Header.Get("TrackRecordingMutatorHeader"))

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		suite.Require().NoError(err)
		_ = resp.Body.Close()
		suite.Require().Equal(fmt.Sprintf("Hello, server responds '%d' to query '%d'", i, i), string(bodyBytes))

		suite.Require().Equal(int64(38+len(strconv.Itoa(i))), resp.ContentLength)
		suite.Require().NotNil(resp.Request)
		suite.Require().NotNil(resp.TLS)
	}

	suite.Require().EqualValues(2, suite.vcr.NumberOfTracks())

	actualStats := *suite.vcr.Stats()
	suite.vcr.EjectCassette()

	return actualStats
}
