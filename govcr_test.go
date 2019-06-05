package govcr_test

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

	"github.com/seborama/govcr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVCR(t *testing.T) {
	unit := govcr.NewVCR()
	assert.NotNil(t, unit.Player())
}

func TestVCRControlPanel_LoadCassette_NewCassette(t *testing.T) {
	unit := govcr.NewVCR()
	err := unit.LoadCassette("govcr-fixtures/my.cassette")
	assert.NoError(t, err)
}

func TestVCRControlPanel_LoadCassette_WhenOneIsAlreadyLoaded(t *testing.T) {
	unit := govcr.NewVCR()
	err := unit.LoadCassette("govcr-fixtures/my.cassette")
	assert.NoError(t, err)

	err = unit.LoadCassette("govcr-fixtures/my-other.cassette")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already loaded")
}

func TestVCRControlPanel_LoadCassette_InvalidCassette(t *testing.T) {
	unit := govcr.NewVCR()
	err := unit.LoadCassette("test-fixtures/bad.cassette")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to interpret cassette data")
}

func TestVCRControlPanel_LoadCassette_ValidSimpleLongPlayCassette(t *testing.T) {
	unit := govcr.NewVCR()
	err := unit.LoadCassette("test-fixtures/good_zipped_one_track.cassette.gz")
	assert.NoError(t, err)
	assert.EqualValues(t, 1, unit.NumberOfTracks())
}

func TestVCRControlPanel_LoadCassette_ValidSimpleShortPlayCassette(t *testing.T) {
	unit := govcr.NewVCR()
	err := unit.LoadCassette("test-fixtures/good_one_track.cassette")
	assert.NoError(t, err)
	assert.EqualValues(t, 1, unit.NumberOfTracks())
}

func TestVCRControlPanel_LoadCassette_UnreadableCassette(t *testing.T) {
	removeUnreadableCassette(t)
	createUnreadableCassette(t)

	unit := govcr.NewVCR()
	err := unit.LoadCassette("test-fixtures/unreadable.cassette")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read cassette data from file")

	removeUnreadableCassette(t)
}

func createUnreadableCassette(t *testing.T) {
	err := ioutil.WriteFile("test-fixtures/unreadable.cassette", nil, 0111)
	require.NoError(t, err)
}

func removeUnreadableCassette(t *testing.T) {
	err := os.Chmod("test-fixtures/unreadable.cassette", 0711)
	if os.IsNotExist(err) {
		return
	}
	require.NoError(t, err)

	err = os.Remove("test-fixtures/unreadable.cassette")
	require.NoError(t, err)
}

func TestVCRControlPanel_Player(t *testing.T) {
	vcr := govcr.NewVCR()
	unit := vcr.Player()
	assert.IsType(t, (*http.Client)(nil), unit)
}

type GoVCRTestSuite struct {
	suite.Suite

	vcr          *govcr.ControlPanel
	testServer   *httptest.Server
	cassetteName string
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(GoVCRTestSuite))
}
func (suite *GoVCRTestSuite) SetupTest() {
	func() {
		counter := 0
		suite.testServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			counter++
			if r.URL.Query().Get("crash") == "1" {
				panic("simulate a server crash")
			}
			iQuery := r.URL.Query().Get("i")
			_, _ = fmt.Fprintf(w, "Hello, server responds '%d' to query '%s'", counter, iQuery)
		}))
	}()

	testServerClient := suite.testServer.Client()
	testServerClient.Timeout = 3 * time.Second
	suite.vcr = govcr.NewVCR(govcr.WithClient(testServerClient))
	suite.cassetteName = "test-fixtures/TestRecordsTrack.cassette"
	_ = os.Remove(suite.cassetteName)
}

func (suite *GoVCRTestSuite) TearDownTest() {
	_ = os.Remove(suite.cassetteName)
}

func (suite *GoVCRTestSuite) TestRoundTrip_ReplaysError() {
	tt := []struct {
		name    string
		reqURL  string
		wantErr string
	}{
		{
			name:    "should replay protocol error",
			reqURL:  "boom://127.1.2.3",
			wantErr: `Get boom://127.1.2.3: *http.badStringError: unsupported protocol scheme "boom"`,
		},
		{
			name:    "should replay request cancellation on connection failure",
			reqURL:  "https://127.1.2.3",
			wantErr: `Get https://127.1.2.3: *errors.errorString: net/http: request canceled while waiting for connection`,
		},
		{
			name:    "should replay request on server crash",
			reqURL:  suite.testServer.URL + "?crash=1",
			wantErr: `Get ` + suite.testServer.URL + `?crash=1: *errors.errorString: EOF`,
		},
	}

	for idx, tc := range tt {
		suite.T().Run(tc.name, func(t *testing.T) {
			cassetteName := suite.cassetteName + fmt.Sprintf("test_case_%d", idx)
			_ = os.Remove(cassetteName)
			defer func() { _ = os.Remove(cassetteName) }()

			// execute HTTP call and record on cassette
			err := suite.vcr.LoadCassette(cassetteName)
			suite.Require().NoError(err)

			resp, err := suite.vcr.Player().Get(tc.reqURL)
			suite.Require().Error(err)
			suite.Require().Nil(resp)

			suite.EqualValues(1, suite.vcr.NumberOfTracks())

			actualStats := *suite.vcr.Stats()
			suite.vcr.EjectCassette()
			suite.EqualValues(0, suite.vcr.NumberOfTracks())

			expectedStats := govcr.Stats{
				TracksLoaded:   0,
				TracksRecorded: 1,
				TracksPlayed:   0,
			}
			suite.EqualValues(expectedStats, actualStats)

			// replay from cassette
			suite.Require().FileExists(cassetteName)
			err = suite.vcr.LoadCassette(cassetteName)
			suite.Require().NoError(err)
			suite.EqualValues(1, suite.vcr.NumberOfTracks())

			resp, err = suite.vcr.Player().Get(tc.reqURL)
			suite.Require().Error(err)
			suite.EqualError(err, tc.wantErr)
			suite.Require().Nil(resp)

			suite.EqualValues(1, suite.vcr.NumberOfTracks())

			actualStats = *suite.vcr.Stats()
			suite.vcr.EjectCassette()
			suite.EqualValues(0, suite.vcr.NumberOfTracks())

			expectedStats = govcr.Stats{
				TracksLoaded:   1,
				TracksRecorded: 0,
				TracksPlayed:   1,
			}
			suite.EqualValues(expectedStats, actualStats)
		})
	}
}

func (suite *GoVCRTestSuite) TestRoundTrip_ReplaysResponse() {
	actualStats := suite.makeHTTPCalls_WithSuccess()
	expectedStats := govcr.Stats{
		TracksLoaded:   0,
		TracksRecorded: 2,
		TracksPlayed:   0,
	}
	suite.EqualValues(expectedStats, actualStats)
	suite.Require().FileExists(suite.cassetteName)

	actualStats = suite.makeHTTPCalls_WithSuccess()
	expectedStats = govcr.Stats{
		TracksLoaded:   2,
		TracksRecorded: 0,
		TracksPlayed:   2,
	}
	suite.EqualValues(expectedStats, actualStats)
}

func (suite *GoVCRTestSuite) TestRoundTrip_ReplaysResponse_WithTrackMutator() {
	suite.T().Fatal("implement me - SHOULD TEST TRACKREPLAYMUTATOR")
}

func (suite *GoVCRTestSuite) makeHTTPCalls_WithSuccess() govcr.Stats {
	err := suite.vcr.LoadCassette(suite.cassetteName)
	suite.Require().NoError(err)

	for i := 1; i <= 2; i++ {
		req, err := http.NewRequest(http.MethodGet, suite.testServer.URL+fmt.Sprintf("?i=%d", i), nil)
		suite.Require().NoError(err)
		req.Header.Add("header", "value")
		req.SetBasicAuth("not_a_username", "not_a_password")

		resp, err := suite.vcr.Player().Do(req)
		suite.Require().NoError(err)

		suite.Equal(http.StatusOK, resp.StatusCode)
		suite.EqualValues(strconv.Itoa(38+len(strconv.Itoa(i))), resp.Header.Get("Content-Length"))
		suite.EqualValues("text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
		suite.NotEmpty(resp.Header.Get("Date"))
		suite.EqualValues(resp.Trailer, http.Header(nil))

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		suite.Require().NoError(err)
		resp.Body.Close()
		suite.Equal(fmt.Sprintf("Hello, server responds '%d' to query '%d'", i, i), string(bodyBytes))

		suite.Equal(int64(38+len(strconv.Itoa(i))), resp.ContentLength)
		suite.NotNil(resp.Request)
		suite.NotNil(resp.TLS)
	}

	suite.EqualValues(2, suite.vcr.NumberOfTracks())

	actualStats := *suite.vcr.Stats()
	suite.vcr.EjectCassette()

	return actualStats
}

func TestRoundTrip_DefaultHeaderMatcher(t *testing.T) {
	tt := []struct {
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
			httpReq := govcr.Request{Header: tc.reqHeaders}
			trackReq := govcr.Request{Header: tc.trackHeaders}
			actualMatch := govcr.DefaultHeaderMatcher(&httpReq, &trackReq)
			assert.Equal(t, tc.want, actualMatch)
		})
	}
}

func TestRoundTrip_DefaultMethodMatcher(t *testing.T) {
	tt := []struct {
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
			httpReq := govcr.Request{Method: tc.reqMethod}
			trackReq := govcr.Request{Method: tc.trackMethod}
			actualMatch := govcr.DefaultMethodMatcher(&httpReq, &trackReq)
			assert.Equal(t, tc.want, actualMatch)
		})
	}
}

func TestRoundTrip_DefaultURLMatcher(t *testing.T) {
	t.Fatal("implement me")
}

func TestRoundTrip_DefaultBodyMatcher(t *testing.T) {
	t.Fatal("implement me")
}

func TestRoundTrip_DefaultTrailerMatcher(t *testing.T) {
	t.Fatal("implement me")
}
