package govcr_test

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/seborama/govcr/v17"
	"github.com/seborama/govcr/v17/cassette"
	"github.com/seborama/govcr/v17/cassette/track"
	"github.com/seborama/govcr/v17/encryption"
	"github.com/seborama/govcr/v17/stats"
)

func TestNewVCR(t *testing.T) {
	const cassetteName = "temp-fixtures/my.cassette.json"

	_ = os.Remove(cassetteName)
	defer func() { _ = os.Remove(cassetteName) }()

	unit := govcr.NewVCR(govcr.NewCassetteLoader(cassetteName))
	assert.NotNil(t, unit.HTTPClient())
}

func TestVCRControlPanel_NewVCR_InvalidCassette(t *testing.T) {
	assert.Panics(
		t,
		func() {
			_ = govcr.NewVCR(
				govcr.NewCassetteLoader("test-fixtures/bad.cassette.json"),
			)
		})
}

func TestVCRControlPanel_NewVCR_ValidSimpleLongPlayCassette(t *testing.T) {
	unit := govcr.NewVCR(
		govcr.NewCassetteLoader("test-fixtures/good_zipped_one_track.cassette.json.gz"),
	)
	assert.EqualValues(t, 1, unit.NumberOfTracks())
}

func TestVCRControlPanel_NewVCR_ValidSimpleShortPlayCassette(t *testing.T) {
	unit := govcr.NewVCR(
		govcr.NewCassetteLoader("test-fixtures/good_one_track.cassette.json"),
	)
	assert.EqualValues(t, 1, unit.NumberOfTracks())
}

func TestVCRControlPanel_NewVCR_UnreadableCassette(t *testing.T) {
	const cassetteName = "test-fixtures/unreadable.cassette.json"

	removeUnreadableCassette(t, cassetteName)
	createUnreadableCassette(t, cassetteName)

	assert.Panics(
		t,
		func() {
			_ = govcr.NewVCR(
				govcr.NewCassetteLoader(cassetteName),
			)
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
	vcr := govcr.NewVCR(govcr.NewCassetteLoader("./temp-fixtures/TestVCRControlPanel_HTTPClient.cassette"))
	unit := vcr.HTTPClient()
	assert.IsType(t, (*http.Client)(nil), unit)
}

func TestSetCrypto(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, "Hello: %d\n", rand.Intn(1e9))
	}))

	const cassetteName = "./temp-fixtures/TestSetCrypto.cassette"

	_ = os.Remove(cassetteName)

	// first, create an unencrypted cassette
	vcr := govcr.NewVCR(govcr.NewCassetteLoader(cassetteName))

	// add a track to the cassette to trigger its creation in the first place
	resp, err := vcr.HTTPClient().Get(testServer.URL)
	require.NoError(t, err)

	_ = resp.Body.Close()

	assert.Equal(t, "not encrypted", getCassetteCrypto(cassetteName))

	// encrypt cassette with AESGCM
	err = vcr.SetCipher(
		encryption.NewAESGCMWithRandomNonceGenerator,
		"test-fixtures/TestSetCrypto.1.key",
	)
	require.NoError(t, err)

	assert.Equal(t, "aesgcm", getCassetteCrypto(cassetteName))

	// re-encrypt cassette with ChaCha20Poly1305
	err = vcr.SetCipher(
		encryption.NewChaCha20Poly1305WithRandomNonceGenerator,
		"test-fixtures/TestSetCrypto.2.key",
	)
	require.NoError(t, err)

	assert.Equal(t, "chacha20poly1305", getCassetteCrypto(cassetteName))

	// lastly, attempt to decrypt cassette - this is not permitted
	err = vcr.SetCipher(nil, "")
	require.Error(t, err)
}

func getCassetteCrypto(cassetteName string) string {
	data, err := os.ReadFile(cassetteName)
	if err != nil {
		panic(err)
	}

	marker := "$ENC:V2$"

	if !bytes.HasPrefix(data, []byte(marker)) {
		return "not encrypted"
	}

	pos := len(marker)
	cipherNameLen := int(data[len(marker)])
	return string(data[pos+1 : pos+1+cipherNameLen])
}

type GoVCRTestSuite struct {
	suite.Suite

	testServer *httptest.Server
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
}

func (ts *GoVCRTestSuite) TearDownTest() {
	ts.testServer.Close()
}

type action int

const (
	actionKeepCassette = iota
	actionDeleteCassette
)

func (ts *GoVCRTestSuite) newVCR(cassetteName string, a action) *govcr.ControlPanel {
	if a == actionDeleteCassette {
		_ = os.Remove(cassetteName)
	}

	testServerClient := ts.testServer.Client()
	testServerClient.Timeout = 3 * time.Second

	return govcr.NewVCR(
		govcr.NewCassetteLoader(cassetteName),
		govcr.WithClient(testServerClient),
	)
}

func (ts *GoVCRTestSuite) TestVCR_ReadOnlyMode() {
	const k7Name = "temp-fixtures/TestGoVCRTestSuite.TestVCR_ReadOnlyMode.cassette.json"

	vcr := ts.newVCR(k7Name, actionDeleteCassette)
	vcr.SetReadOnlyMode(true)

	resp, err := vcr.HTTPClient().Get(ts.testServer.URL)
	ts.Require().NoError(err)
	ts.Require().NotNil(resp)
	defer func() { _ = resp.Body.Close() }()

	expectedStats := &stats.Stats{
		TotalTracks:    0,
		TracksLoaded:   0,
		TracksRecorded: 0,
		TracksPlayed:   0,
	}
	ts.Equal(expectedStats, vcr.Stats())
}

func (ts *GoVCRTestSuite) TestVCR_LiveOnlyMode() {
	const k7Name = "temp-fixtures/TestGoVCRTestSuite.TestVCR_LiveOnlyMode.cassette.json"

	// 1st execution of set of calls
	vcr := ts.newVCR(k7Name, actionDeleteCassette)
	vcr.SetLiveOnlyMode()
	vcr.SetRequestMatchers(alwaysMatchRequest) // ensure always matching

	ts.makeHTTPCallsWithSuccess(vcr.HTTPClient(), 0)
	expectedStats := &stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   0,
		TracksRecorded: 2,
		TracksPlayed:   0,
	}
	ts.Equal(expectedStats, vcr.Stats())
	ts.Require().FileExists(k7Name)

	// 2nd execution of set of calls
	vcr = ts.newVCR(k7Name, actionKeepCassette)
	vcr.SetLiveOnlyMode()
	vcr.SetRequestMatchers(alwaysMatchRequest) // ensure always matching

	ts.makeHTTPCallsWithSuccess(vcr.HTTPClient(), 2) // as we're making live requests, the sever keeps on increasing the counter
	expectedStats = &stats.Stats{
		TotalTracks:    4,
		TracksLoaded:   2,
		TracksRecorded: 2,
		TracksPlayed:   0,
	}
	ts.Equal(expectedStats, vcr.Stats())
}

func (ts *GoVCRTestSuite) TestVCR_OfflineMode() {
	const k7Name = "temp-fixtures/TestGoVCRTestSuite.TestVCR_OfflineMode.cassette.json"

	// 1st execution of set of calls - populate cassette
	vcr := ts.newVCR(k7Name, actionDeleteCassette)
	vcr.SetRequestMatchers(alwaysMatchRequest) // ensure always matching
	vcr.SetNormalMode()                        // get data in the cassette

	ts.makeHTTPCallsWithSuccess(vcr.HTTPClient(), 0)
	expectedStats := &stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   0,
		TracksRecorded: 2,
		TracksPlayed:   0,
	}
	ts.Equal(expectedStats, vcr.Stats())
	ts.Require().FileExists(k7Name)

	// 2nd execution of set of calls -- offline only
	vcr = ts.newVCR(k7Name, actionKeepCassette)
	vcr.SetOfflineMode()

	ts.makeHTTPCallsWithSuccess(vcr.HTTPClient(), 0)
	expectedStats = &stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   2,
		TracksRecorded: 0,
		TracksPlayed:   2,
	}
	ts.Equal(expectedStats, vcr.Stats())

	// 3rd execution of set of calls -- still offline only
	// we've run out of tracks on the cassette and we're in offline mode so we expect a transport error
	req, err := http.NewRequest(http.MethodGet, ts.testServer.URL, http.NoBody)
	ts.Require().NoError(err)
	resp, err := vcr.HTTPClient().Do(req)
	ts.Require().Error(err)
	ts.Contains(err.Error(), "no track matched on cassette and offline mode is active")
	ts.Nil(resp)
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

	const k7Name = "temp-fixtures/TestGoVCRTestSuite.TestRoundTrip_ReplaysError.cassette.json"

	for idx, tc := range tt {
		ts.Run(tc.name, func() {
			cassetteName := k7Name + fmt.Sprintf(".test_case_%d", idx)

			// execute HTTP call and record on cassette
			vcr := ts.newVCR(cassetteName, actionDeleteCassette)

			resp, err := vcr.HTTPClient().Get(tc.reqURL)
			ts.Require().EqualError(err, tc.wantErr)
			ts.Require().Nil(resp)

			expectedStats := &stats.Stats{
				TotalTracks:    1,
				TracksLoaded:   0,
				TracksRecorded: 1,
				TracksPlayed:   0,
			}
			ts.Equal(expectedStats, vcr.Stats())

			// replay from cassette
			ts.Require().FileExists(cassetteName)
			vcr = ts.newVCR(cassetteName, actionKeepCassette)
			ts.EqualValues(1, vcr.NumberOfTracks())

			resp, err = vcr.HTTPClient().Get(tc.reqURL)
			ts.Require().EqualError(err, tc.wantVCRErr)
			ts.Require().Nil(resp)

			expectedStats = &stats.Stats{
				TotalTracks:    1,
				TracksLoaded:   1,
				TracksRecorded: 0,
				TracksPlayed:   1,
			}
			ts.Equal(expectedStats, vcr.Stats())
		})
	}
}

func (ts *GoVCRTestSuite) TestRoundTrip_ReplaysPlainResponse() {
	const k7Name = "temp-fixtures/TestGoVCRTestSuite.TestRoundTrip_ReplaysPlainResponse.cassette.json"

	// 1st execution of set of calls
	vcr := ts.newVCR(k7Name, actionDeleteCassette)

	ts.makeHTTPCallsWithSuccess(vcr.HTTPClient(), 0)
	expectedStats := &stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   0,
		TracksRecorded: 2,
		TracksPlayed:   0,
	}
	ts.Equal(expectedStats, vcr.Stats())
	ts.Require().FileExists(k7Name)

	// 2nd execution of set of calls (replayed with cassette reload)
	vcr = ts.newVCR(k7Name, actionKeepCassette)

	ts.makeHTTPCallsWithSuccess(vcr.HTTPClient(), 0)
	expectedStats = &stats.Stats{
		TotalTracks:    2,
		TracksLoaded:   2,
		TracksRecorded: 0,
		TracksPlayed:   2,
	}
	ts.Equal(expectedStats, vcr.Stats())

	// 3rd execution of set of calls (replayed without cassette reload)
	ts.makeHTTPCallsWithSuccess(vcr.HTTPClient(), int(expectedStats.TotalTracks)) // as we're making live requests, the sever keeps on increasing the counter
	expectedStats = &stats.Stats{
		TotalTracks:    4,
		TracksLoaded:   2,
		TracksRecorded: 2,
		TracksPlayed:   2,
	}
	ts.Equal(expectedStats, vcr.Stats())
}

// This test checks that on recording a new track to an existing cassette, the
// Response.Request of replayed tracks is not persisted from the replay.
func TestRecordReplayRecord(t *testing.T) {
	const k7Name = "temp-fixtures/TestRecordReplayRecord.cassette.json"

	_ = os.Remove(k7Name)

	vcr := govcr.NewVCR(
		govcr.NewCassetteLoader(k7Name),
		govcr.WithRequestMatchers(govcr.NewMethodURLRequestMatchers()...), // use a "relaxed" request matcher
	)

	// The first request will be live and transparently recorded by govcr since the cassette is empty
	vcr.HTTPClient().Get("http://example.com/foo")
	assert.Equal(
		t,
		&stats.Stats{
			TotalTracks:    1,
			TracksLoaded:   0,
			TracksRecorded: 1,
			TracksPlayed:   0,
		},
		vcr.Stats(),
	)
	k789 := cassette.LoadCassette(k7Name)
	assert.Len(t, k789.Tracks, 1)
	assert.Nil(t, k789.Tracks[0].Response.Request, "the Response.Request is not nil")

	// The second request will be transparently replayed from the cassette by govcr
	// No live HTTP request is placed to the live server.
	vcr = govcr.NewVCR(
		govcr.NewCassetteLoader(k7Name),
		govcr.WithRequestMatchers(govcr.NewMethodURLRequestMatchers()...), // use a "relaxed" request matcher
	)

	vcr.HTTPClient().Get("http://example.com/foo")
	assert.Equal(
		t,
		&stats.Stats{
			TotalTracks:    1,
			TracksLoaded:   1,
			TracksRecorded: 0,
			TracksPlayed:   1,
		},
		vcr.Stats(),
	)

	// The third request will be live and transparently recorded by govcr since no existing
	// track on the cassette will match.
	vcr.HTTPClient().Get("http://example.com/foo/bar")
	assert.Equal(
		t,
		&stats.Stats{
			TotalTracks:    2,
			TracksLoaded:   1,
			TracksRecorded: 1,
			TracksPlayed:   1,
		},
		vcr.Stats(),
	)

	// Verify the 1st cassette track has not recorded Response.Request from the track replay.
	k7 := cassette.LoadCassette(k7Name)
	assert.Len(t, k7.Tracks, 2)
	assert.Nil(t, k7.Tracks[0].Response.Request)
}

func (ts *GoVCRTestSuite) makeHTTPCallsWithSuccess(httpClient *http.Client, serverCurrentCount int) {
	for i := 1; i <= 2; i++ {
		req, err := http.NewRequest(http.MethodGet, ts.testServer.URL+fmt.Sprintf("?i=%d", i), http.NoBody)
		ts.Require().NoError(err)
		req.Header.Add("Header", "value")
		req.SetBasicAuth("not_a_username", "not_a_password")

		resp, err := httpClient.Do(req)
		ts.Require().NoError(err)

		ts.Equal(http.StatusOK, resp.StatusCode)
		ts.Equal(strconv.Itoa(38+len(strconv.Itoa(i))), resp.Header.Get("Content-Length"))
		ts.Equal("text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
		ts.NotEmpty(resp.Header.Get("Date"))
		ts.Equal(resp.Trailer, http.Header(nil))

		bodyBytes, err := io.ReadAll(resp.Body)
		ts.Require().NoError(err)
		_ = resp.Body.Close()
		ts.Equal(fmt.Sprintf("Hello, server responds '%d' to query '%d'", serverCurrentCount+i, i), string(bodyBytes))

		ts.Equal(int64(38+len(strconv.Itoa(serverCurrentCount+i))), resp.ContentLength)
		ts.NotNil(resp.Request)
		ts.NotNil(resp.TLS)
	}
}

func alwaysMatchRequest(_, _ *track.Request) bool {
	return true
}
