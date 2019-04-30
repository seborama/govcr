package govcr_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

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

func TestVCRControlPanel_LoadCassette_ValidBlankCassette(t *testing.T) {
	unit := govcr.NewVCR()
	err := unit.LoadCassette("test-fixtures/good_blank.cassette")
	assert.NoError(t, err)
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
		counter := 1
		suite.testServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprintf(w, "Hello, from server: %d", counter)
			counter++
		}))
	}()

	suite.vcr = govcr.NewVCR(govcr.WithClient(suite.testServer.Client()))
	suite.cassetteName = "test-fixtures/TestRecordsTrack.cassette"
	_ = os.Remove(suite.cassetteName)
}

func (suite *GoVCRTestSuite) TearDownTest() {
	_ = os.Remove(suite.cassetteName)
}

func (suite *GoVCRTestSuite) TestRoundTrip_ReplaysResponse() {
	actualStats := suite.makeHTTPCalls()
	expectedStats := govcr.Stats{
		TracksLoaded:   0,
		TracksRecorded: 2,
		TracksPlayed:   0,
	}
	suite.EqualValues(expectedStats, actualStats)
	suite.Require().FileExists(suite.cassetteName)

	actualStats = suite.makeHTTPCalls()
	expectedStats = govcr.Stats{
		TracksLoaded:   2,
		TracksRecorded: 0,
		TracksPlayed:   2,
	}
	suite.EqualValues(expectedStats, actualStats)
}

func (suite *GoVCRTestSuite) makeHTTPCalls() govcr.Stats {
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
		suite.EqualValues(strconv.Itoa(20+len(strconv.Itoa(i))), resp.Header.Get("Content-Length"))
		suite.EqualValues("text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
		suite.NotEmpty(resp.Header.Get("Date"))
		suite.EqualValues(resp.Trailer, http.Header(nil))

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		suite.Require().NoError(err)
		resp.Body.Close()
		suite.Equal(fmt.Sprintf("Hello, from server: %d", i), string(bodyBytes))

		suite.Equal(int64(20+len(strconv.Itoa(i))), resp.ContentLength)
		suite.NotNil(resp.Request)
		suite.NotNil(resp.TLS)
	}

	suite.EqualValues(2, suite.vcr.NumberOfTracks())

	actualStats := *suite.vcr.Stats()
	suite.vcr.EjectCassette()

	return actualStats
}

func (suite *GoVCRTestSuite) TestRoundTrip_DefaultRequestMatcher() {
}

func (suite *GoVCRTestSuite) TestRoundTrip_DefaultHeaderMatcher() {
}

func (suite *GoVCRTestSuite) TestRoundTrip_DefaultMethodMatcher() {
}

func (suite *GoVCRTestSuite) TestRoundTrip_DefaultURLMatcher() {
}

func (suite *GoVCRTestSuite) TestRoundTrip_DefaultBodyMatcher() {
}

func (suite *GoVCRTestSuite) TestRoundTrip_DefaultTrailerMatcher() {
}
