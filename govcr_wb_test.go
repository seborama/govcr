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

	"github.com/seborama/govcr/cassette"
	"github.com/seborama/govcr/stats"
)

func TestRoundTrip_SavesMutatedTracksToCassette(t *testing.T) {
	const cassetteName = "govcr-fixtures/TestRoundTrip_SavesMutatedCassetteTracks.cassette"
	_ = os.Remove(cassetteName)

	var testServer *httptest.Server

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

	aMutator := TrackMutator(
		func(trk *cassette.Track) {
			trk.Request.Method = trk.Request.Method + " has been mutated"
			trk.Response.Status = trk.Response.Status + " has been mutated"
			trk.ErrType = "ErrType was mutated"
			trk.ErrMsg = "ErrMsg was mutated"
		})

	vcr := NewVCR(WithClient(testServerClient), WithTrackRecordingMutators(aMutator))

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

	err = vcr.LoadCassette(cassetteName)
	assert.NoError(t, err)

	for trackNum, aTrack := range vcr.vcrTransport().cassette.Tracks {
		require.EqualValues(t, "GET has been mutated", aTrack.Request.Method, "track #%d", trackNum)
		require.EqualValues(t, "200 OK has been mutated", aTrack.Response.Status, "track #%d", trackNum)
		require.EqualValues(t, "ErrType was mutated", aTrack.ErrType, "track #%d", trackNum)
		require.EqualValues(t, "ErrMsg was mutated", aTrack.ErrMsg, "track #%d", trackNum)
	}
}

func makeHTTPCalls_WithSuccess(testServerURL string, vcr *ControlPanel, t *testing.T) stats.Stats {
	for i := 1; i <= 2; i++ {
		req, err := http.NewRequest(http.MethodGet, testServerURL+fmt.Sprintf("?i=%d", i), nil)
		require.NoError(t, err)
		req.Header.Add("header", "value")
		req.SetBasicAuth("not_a_username", "not_a_password")

		resp, err := vcr.Player().Do(req)
		require.NoError(t, err)

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
