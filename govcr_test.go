package govcr_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/seborama/govcr/v8"
	"github.com/seborama/govcr/v8/stats"
)

func TestNewVCR(t *testing.T) {
	unit := govcr.NewVCR()
	assert.NotNil(t, unit.HTTPClient())
}

func TestVCRControlPanel_LoadCassette_NewCassette(t *testing.T) {
	unit := govcr.NewVCR()
	err := unit.LoadCassette("temp-fixtures/my.cassette.json")
	assert.NoError(t, err)
}

func TestVCRControlPanel_LoadCassette_WhenOneIsAlreadyLoaded(t *testing.T) {
	unit := govcr.NewVCR()
	err := unit.LoadCassette("temp-fixtures/my.cassette.json")
	assert.NoError(t, err)

	err = unit.LoadCassette("temp-fixtures/my-other.cassette.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already loaded")
}

func TestVCRControlPanel_LoadCassette_InvalidCassette(t *testing.T) {
	unit := govcr.NewVCR()
	assert.PanicsWithValue(
		t,
		"unable to load corrupted cassette 'test-fixtures/bad.cassette.json': failed to interpret cassette data in file: invalid character 'T' looking for beginning of value",
		func() {
			_ = unit.LoadCassette("test-fixtures/bad.cassette.json")
		})
}

func TestVCRControlPanel_LoadCassette_ValidSimpleLongPlayCassette(t *testing.T) {
	unit := govcr.NewVCR()
	err := unit.LoadCassette("test-fixtures/good_zipped_one_track.cassette.json.gz")
	assert.NoError(t, err)
	assert.EqualValues(t, 1, unit.NumberOfTracks())
}

func TestVCRControlPanel_LoadCassette_ValidSimpleShortPlayCassette(t *testing.T) {
	unit := govcr.NewVCR()
	err := unit.LoadCassette("test-fixtures/good_one_track.cassette.json")
	assert.NoError(t, err)
	assert.EqualValues(t, 1, unit.NumberOfTracks())
}

func TestVCRControlPanel_LoadCassette_UnreadableCassette(t *testing.T) {
	const cassetteName = "test-fixtures/unreadable.cassette.json"

	removeUnreadableCassette(t, cassetteName)
	createUnreadableCassette(t, cassetteName)

	unit := govcr.NewVCR()
	assert.PanicsWithValue(
		t,
		"unable to load corrupted cassette '"+cassetteName+"': failed to read cassette data from file: open "+cassetteName+": permission denied",
		func() {
			_ = unit.LoadCassette(cassetteName)
		})

	removeUnreadableCassette(t, cassetteName)
}

func createUnreadableCassette(t *testing.T, name string) {
	t.Helper()
	err := os.WriteFile(name, nil, 0o111)
	require.NoError(t, err)
}

func removeUnreadableCassette(t *testing.T, name string) {
	t.Helper()
	err := os.Chmod(name, 0o711)
	if os.IsNotExist(err) {
		return
	}
	require.NoError(t, err)

	err = os.Remove(name)
	require.NoError(t, err)
}

func TestVCRControlPanel_HTTPClient(t *testing.T) {
	vcr := govcr.NewVCR()
	unit := vcr.HTTPClient()
	assert.IsType(t, (*http.Client)(nil), unit)
}

type GoVCRTestSuite struct {
	suite.Suite

	vcr          *govcr.ControlPanel
	testServer   *httptest.Server
	cassetteName string
}

func TestGoVCRTestSuite(t *testing.T) {
	suite.Run(t, new(GoVCRTestSuite))
}

func (ts *GoVCRTestSuite) SetupTest() {
	func() {
		counter := 0
		ts.testServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			counter++
			if r.URL.Query().Get("crash") == "1" {
				panic("simulate a server crash")
			}
			iQuery := r.URL.Query().Get("i")
			_, _ = fmt.Fprintf(w, "Hello, server responds '%d' to query '%s'", counter, iQuery)
		}))
	}()

	testServerClient := ts.testServer.Client()
	testServerClient.Timeout = 3 * time.Second
	ts.vcr = govcr.NewVCR(govcr.WithClient(testServerClient))
	ts.cassetteName = "temp-fixtures/TestGoVCRTestSuite.cassette.json"
	_ = os.Remove(ts.cassetteName)
}

func (ts *GoVCRTestSuite) TearDownTest() {
	_ = os.Remove(ts.cassetteName)
}

func (ts *GoVCRTestSuite) TestVCR_ReadOnlyMode() {
	ts.vcr.SetReadOnlyMode(true)

	err := ts.vcr.LoadCassette(ts.cassetteName)
	ts.Require().NoError(err)

	resp, err := ts.vcr.HTTPClient().Get(ts.testServer.URL)
	ts.Require().NoError(err)
	ts.Require().NotNil(resp)
	defer func() { _ = resp.Body.Close() }()

	actualStats := *ts.vcr.Stats()
	ts.vcr.EjectCassette()

	expectedStats := stats.Stats{
		TotalTracks:    0,
		TracksLoaded:   0,
		TracksRecorded: 0,
		TracksPlayed:   0,
	}
	ts.EqualValues(expectedStats, actualStats)
}

func (ts *GoVCRTestSuite) TestVCR_LiveOnlyMode() {
	ts.vcr.SetLiveOnlyMode()
	ts.vcr.SetRequestMatcher(govcr.NewBlankRequestMatcher()) // ensure always matching

	// 1st execution of set of calls
	err := ts.vcr.LoadCassette(ts.cassetteName)
	ts.Require().NoError(err)

	actualStats := ts.makeHTTPCalls_WithSuccess(0)
	expectedStats := stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   0,
		TracksRecorded: 2,
		TracksPlayed:   0,
	}
	ts.EqualValues(expectedStats, actualStats)
	ts.Require().FileExists(ts.cassetteName)
	ts.vcr.EjectCassette()

	// 2nd execution of set of calls
	err = ts.vcr.LoadCassette(ts.cassetteName)
	ts.Require().NoError(err)

	actualStats = ts.makeHTTPCalls_WithSuccess(2) // as we're making live requests, the sever keeps on increasing the counter
	expectedStats = stats.Stats{
		TotalTracks:    4,
		TracksLoaded:   2,
		TracksRecorded: 2,
		TracksPlayed:   0,
	}
	ts.EqualValues(expectedStats, actualStats)
}

func (ts *GoVCRTestSuite) TestVCR_OfflineMode() {
	ts.vcr.SetRequestMatcher(govcr.NewBlankRequestMatcher()) // ensure always matching

	// 1st execution of set of calls - populate cassette
	ts.vcr.SetNormalMode() // get data in the cassette
	err := ts.vcr.LoadCassette(ts.cassetteName)
	ts.Require().NoError(err)

	actualStats := ts.makeHTTPCalls_WithSuccess(0)
	expectedStats := stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   0,
		TracksRecorded: 2,
		TracksPlayed:   0,
	}
	ts.EqualValues(expectedStats, actualStats)
	ts.Require().FileExists(ts.cassetteName)
	ts.vcr.EjectCassette()

	// 2nd execution of set of calls -- offline only
	ts.vcr.SetOfflineMode()

	err = ts.vcr.LoadCassette(ts.cassetteName)
	ts.Require().NoError(err)

	actualStats = ts.makeHTTPCalls_WithSuccess(0)
	expectedStats = stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   2,
		TracksRecorded: 0,
		TracksPlayed:   2,
	}
	ts.EqualValues(expectedStats, actualStats)

	// 3rd execution of set of calls -- still offline only
	// we've run out of tracks and we're in offline mode so we expect a transport error
	req, err := http.NewRequest(http.MethodGet, ts.testServer.URL, nil)
	ts.Require().NoError(err)
	resp, err := ts.vcr.HTTPClient().Do(req) //nolint: bodyclose
	ts.Require().Error(err)
	ts.Assert().Contains(err.Error(), "no track matched on cassette and offline mode is active")
	ts.Assert().Nil(resp)
}

func (ts *GoVCRTestSuite) TestRoundTrip_ReplaysError() {
	tt := []*struct {
		name       string
		reqURL     string
		wantErr    string
		wantVCRErr string
	}{
		// NOTE: different versions of Go have variations of these actual errors - below are for Go 1.18
		{
			name:       "should replay protocol error",
			reqURL:     "boom://127.1.2.3",
			wantErr:    `Get "boom://127.1.2.3": unsupported protocol scheme "boom"`,
			wantVCRErr: `Get "boom://127.1.2.3": *errors.errorString: unsupported protocol scheme "boom"`,
		},
		// This test is flaky: it can return 2 different types of errors (Go 1.18)
		// {
		// 	name:       "should replay request cancellation on connection failure",
		// 	reqURL:     "https://127.1.2.3",
		// 	wantErr:    `Get "https://127.1.2.3": net/http: request canceled while waiting for connection (Client.Timeout exceeded while awaiting headers)`,
		// 	wantVCRErr: `Get "https://127.1.2.3": *errors.errorString: net/http: request canceled while waiting for connection`,
		// },
		{
			name:       "should replay request on server crash",
			reqURL:     ts.testServer.URL + "?crash=1",
			wantErr:    `Get "` + ts.testServer.URL + `?crash=1": EOF`,
			wantVCRErr: `Get "` + ts.testServer.URL + `?crash=1": *errors.errorString: EOF`,
		},
	}

	for idx, tc := range tt {
		ts.T().Run(tc.name, func(t *testing.T) {
			cassetteName := ts.cassetteName + fmt.Sprintf("test_case_%d", idx)
			_ = os.Remove(cassetteName)
			defer func() { _ = os.Remove(cassetteName) }()

			// execute HTTP call and record on cassette
			err := ts.vcr.LoadCassette(cassetteName)
			ts.Require().NoError(err)

			resp, err := ts.vcr.HTTPClient().Get(tc.reqURL) //nolint: bodyclose
			ts.Require().Error(err)
			ts.EqualError(err, tc.wantErr)
			ts.Require().Nil(resp)

			actualStats := *ts.vcr.Stats()
			ts.vcr.EjectCassette()

			expectedStats := stats.Stats{
				TotalTracks:    1,
				TracksLoaded:   0,
				TracksRecorded: 1,
				TracksPlayed:   0,
			}
			ts.EqualValues(expectedStats, actualStats)

			// replay from cassette
			ts.Require().FileExists(cassetteName)
			err = ts.vcr.LoadCassette(cassetteName)
			ts.Require().NoError(err)
			ts.EqualValues(1, ts.vcr.NumberOfTracks())

			resp, err = ts.vcr.HTTPClient().Get(tc.reqURL) //nolint: bodyclose
			ts.Require().Error(err)
			ts.EqualError(err, tc.wantVCRErr)
			ts.Require().Nil(resp)

			actualStats = *ts.vcr.Stats()
			ts.vcr.EjectCassette()

			expectedStats = stats.Stats{
				TotalTracks:    1,
				TracksLoaded:   1,
				TracksRecorded: 0,
				TracksPlayed:   1,
			}
			ts.EqualValues(expectedStats, actualStats)
		})
	}
}

func (suite *GoVCRTestSuite) TestRoundTrip_ReplaysPlainResponse() {
	// 1st execution of set of calls
	err := suite.vcr.LoadCassette(suite.cassetteName)
	suite.Require().NoError(err)

	actualStats := suite.makeHTTPCalls_WithSuccess(0)
	expectedStats := stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   0,
		TracksRecorded: 2,
		TracksPlayed:   0,
	}
	suite.EqualValues(expectedStats, actualStats)
	suite.Require().FileExists(suite.cassetteName)
	suite.vcr.EjectCassette()

	// 2nd execution of set of calls (replayed with cassette reload)
	err = suite.vcr.LoadCassette(suite.cassetteName)
	suite.Require().NoError(err)

	actualStats = suite.makeHTTPCalls_WithSuccess(0)
	expectedStats = stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   2,
		TracksRecorded: 0,
		TracksPlayed:   2,
	}
	suite.EqualValues(expectedStats, actualStats)

	// 3rd execution of set of calls (replayed without cassette reload)
	actualStats = suite.makeHTTPCalls_WithSuccess(int(expectedStats.TotalTracks)) // as we're making live requests, the sever keeps on increasing the counter
	expectedStats = stats.Stats{
		TotalTracks:    4,
		TracksLoaded:   2,
		TracksRecorded: 2,
		TracksPlayed:   2,
	}
	suite.EqualValues(expectedStats, actualStats)
	suite.vcr.EjectCassette()
}

func (suite *GoVCRTestSuite) makeHTTPCalls_WithSuccess(serverCurrentCount int) stats.Stats {
	for i := 1; i <= 2; i++ {
		req, err := http.NewRequest(http.MethodGet, suite.testServer.URL+fmt.Sprintf("?i=%d", i), nil)
		suite.Require().NoError(err)
		req.Header.Add("header", "value")
		req.SetBasicAuth("not_a_username", "not_a_password")

		resp, err := suite.vcr.HTTPClient().Do(req)
		suite.Require().NoError(err)

		suite.Equal(http.StatusOK, resp.StatusCode)
		suite.EqualValues(strconv.Itoa(38+len(strconv.Itoa(i))), resp.Header.Get("Content-Length"))
		suite.EqualValues("text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
		suite.NotEmpty(resp.Header.Get("Date"))
		suite.EqualValues(resp.Trailer, http.Header(nil))

		bodyBytes, err := io.ReadAll(resp.Body)
		suite.Require().NoError(err)
		_ = resp.Body.Close()
		suite.Equal(fmt.Sprintf("Hello, server responds '%d' to query '%d'", serverCurrentCount+i, i), string(bodyBytes))

		suite.Equal(int64(38+len(strconv.Itoa(serverCurrentCount+i))), resp.ContentLength)
		suite.NotNil(resp.Request)
		suite.NotNil(resp.TLS)
	}

	actualStats := *suite.vcr.Stats()

	return actualStats
}
