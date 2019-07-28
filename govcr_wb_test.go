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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/cassette/track"
	"github.com/seborama/govcr/stats"
)

func TestRoundTrip_SavesMutatedTracksToCassette(t *testing.T) {
	const cassetteName = "govcr-fixtures/TestRoundTrip_SavesMutatedCassetteTracks.cassette"

	var testServer *httptest.Server

	// create a test server for the purpose of this test
	func() {
		counter := 0
		testServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			counter++
			if r.URL.Query().Get("crash") == "1" {
				panic("simulate a server crash")
			}
			iQuery := r.URL.Query().Get("i")
			_, _ = fmt.Fprintf(w, "Hello, server responds '%d' to query '%s'", counter, iQuery)
		}))
	}()

	testServerClient := testServer.Client()
	testServerClient.Timeout = 3 * time.Second

	// example mutator, mutation is not too intrusive to allow replaying correctly.
	// for instance, when an Err is injected, the response is set to nil on replay, as per
	// go's HTTP client design.
	aMutator := TrackMutator(
		func(trk *track.Track) {
			q := trk.Request.URL.Query()
			q.Set("mutated_query_key", "this_query_key_has_been_mutated")
			trk.Request.URL.RawQuery = q.Encode()
			trk.Response.Status = trk.Response.Status + " has been mutated"
		})

	// create a new VCR for the test
	vcr := NewVCR(WithClient(testServerClient), WithTrackRecordingMutators(aMutator))

	// load a fresh cassette
	_ = os.Remove(cassetteName)
	err := vcr.LoadCassette(cassetteName)
	assert.NoError(t, err)
	defer func() { _ = os.Remove(cassetteName) }()

	// 1st execution of set of calls
	actualStats := makeHTTPCalls_WithSuccess(testServer.URL, vcr, t)
	expectedStats := stats.Stats{
		TracksLoaded:   0,
		TracksRecorded: 2,
		TracksPlayed:   0,
	}
	require.EqualValues(t, expectedStats, actualStats)

	// load the cassette and verify contents has been mutated.
	err = vcr.LoadCassette(cassetteName)
	assert.NoError(t, err)

	for trackNum, aTrack := range vcr.vcrTransport().cassette.Tracks {
		require.EqualValues(t, "this_query_key_has_been_mutated", aTrack.Request.URL.Query().Get("mutated_query_key"), "track #%d", trackNum)
		require.EqualValues(t, "200 OK has been mutated", aTrack.Response.Status, "track #%d", trackNum)
	}

	// 2nd execution of set of calls (replayed)
	actualStats = replayHTTPCalls_WithMutations_WithSuccess(testServer.URL, vcr, t)
	expectedStats = stats.Stats{
		TracksLoaded:   2,
		TracksRecorded: 0,
		TracksPlayed:   2,
	}
	assert.EqualValues(t, expectedStats, actualStats)
}

func makeHTTPCalls_WithSuccess(testServerURL string, vcr *ControlPanel, t *testing.T) stats.Stats {
	for i := 1; i <= 2; i++ {
		req, err := http.NewRequest(http.MethodGet, testServerURL+fmt.Sprintf("?i=%d", i), nil)
		require.NoError(t, err)
		req.Header.Add("header", "value")
		req.SetBasicAuth("not_a_username", "not_a_password")

		resp, err := vcr.Player().Do(req)
		require.NoError(t, err)

		require.Equal(t, "200 OK", resp.Status)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.EqualValues(t, strconv.Itoa(38+len(strconv.Itoa(i))), resp.Header.Get("Content-Length"))
		require.EqualValues(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
		require.NotEmpty(t, resp.Header.Get("Date"))
		require.EqualValues(t, resp.Trailer, http.Header(nil))

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		_ = resp.Body.Close()
		require.Equal(t, fmt.Sprintf("Hello, server responds '%d' to query '%d'", i, i), string(bodyBytes))

		require.Equal(t, int64(38+len(strconv.Itoa(i))), resp.ContentLength)
		require.NotNil(t, resp.Request)
		require.NotNil(t, resp.TLS)
	}

	require.EqualValues(t, 2, vcr.NumberOfTracks())

	actualStats := *vcr.Stats()
	vcr.EjectCassette()

	return actualStats
}

func replayHTTPCalls_WithMutations_WithSuccess(testServerURL string, vcr *ControlPanel, t *testing.T) stats.Stats {
	for i := 1; i <= 2; i++ {
		req, err := http.NewRequest(http.MethodGet, testServerURL+fmt.Sprintf("?i=%d", i), nil)
		require.NoError(t, err)

		// manually modify the request inline with the previous mutations that took place.
		// not doing so would prevent matching our request against the (mutated) cassette.
		q := req.URL.Query()
		q.Set("mutated_query_key", "this_query_key_has_been_mutated")
		req.URL.RawQuery = q.Encode()

		req.Header.Add("header", "value")
		req.SetBasicAuth("not_a_username", "not_a_password")

		resp, err := vcr.Player().Do(req)
		require.NoError(t, err)

		require.Equal(t, "200 OK has been mutated", resp.Status)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.EqualValues(t, strconv.Itoa(38+len(strconv.Itoa(i))), resp.Header.Get("Content-Length"))
		require.EqualValues(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
		require.NotEmpty(t, resp.Header.Get("Date"))
		require.EqualValues(t, resp.Trailer, http.Header(nil))

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		_ = resp.Body.Close()
		require.Equal(t, fmt.Sprintf("Hello, server responds '%d' to query '%d'", i, i), string(bodyBytes))

		require.Equal(t, int64(38+len(strconv.Itoa(i))), resp.ContentLength)
		require.NotNil(t, resp.Request)
		require.NotNil(t, resp.TLS)
	}

	require.EqualValues(t, 2, vcr.NumberOfTracks())

	actualStats := *vcr.Stats()
	vcr.EjectCassette()

	return actualStats
}
