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

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/v12"
	"github.com/seborama/govcr/v12/stats"
)

func TestConcurrencySafety(t *testing.T) {
	const cassetteName = "temp-fixtures/TestConcurrencySafety.cassette"
	threadMax := int8(50)

	// create a test server
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(50)))

		clientNum, err := strconv.ParseInt(r.URL.Query().Get("num"), 0, 8)
		require.NoError(t, err)

		data := generateBinaryBody(int8(clientNum))
		written, err := w.Write(data)
		if written != len(data) {
			t.Fatalf("** Only %d bytes out of %d were written", written, len(data))
		}
		if err != nil {
			t.Fatalf("err from w.Write(): Expected nil, got %s", err)
		}
	}))
	defer ts.Close()

	testServerClient := ts.Client()
	testServerClient.Timeout = 5 * time.Second

	_ = os.Remove(cassetteName)
	defer func() { _ = os.Remove(cassetteName) }()

	vcr := createVCR(cassetteName, testServerClient)
	client := vcr.HTTPClient()

	t.Run("main - phase 1", func(t *testing.T) {
		// run requests
		for i := int8(1); i <= threadMax; i++ {
			func(i1 int8) {
				t.Run(fmt.Sprintf("i=%d", i), func(t *testing.T) {
					t.Parallel()

					func() {
						resp, err := client.Get(fmt.Sprintf("%s?num=%d", ts.URL, i1))
						require.NoError(t, err)

						// check outcome of the request
						expectedBody := generateBinaryBody(i1)
						err = validateResponseForTestPlaybackOrder(resp, expectedBody)
						require.NoError(t, err)
					}()
				})
			}(i)
		}
	})

	expectedStats := stats.Stats{
		TotalTracks:    int32(threadMax),
		TracksLoaded:   0,
		TracksRecorded: int32(threadMax),
		TracksPlayed:   0,
	}
	require.EqualValues(t, expectedStats, *vcr.Stats())

	// re-run request and expect play back from vcr
	vcr = createVCR(cassetteName, testServerClient)
	client = vcr.HTTPClient()

	// run requests
	t.Run("main - phase 2 - playback", func(t *testing.T) {
		// run requests
		for i := int8(1); i <= threadMax; i++ {
			func(i1 int8) {
				t.Run(fmt.Sprintf("i=%d", i), func(t *testing.T) {
					t.Parallel()

					func() {
						resp, _ := client.Get(fmt.Sprintf("%s?num=%d", ts.URL, i1))

						// check outcome of the request
						expectedBody := generateBinaryBody(i1)
						if err := validateResponseForTestPlaybackOrder(resp, expectedBody); err != nil {
							t.Fatalf(err.Error())
						}
					}()
				})
			}(i)
		}
	})

	expectedStats = stats.Stats{
		TotalTracks:    int32(threadMax),
		TracksLoaded:   int32(threadMax),
		TracksRecorded: 0,
		TracksPlayed:   int32(threadMax),
	}
	require.EqualValues(t, expectedStats, *vcr.Stats())
}

func createVCR(cassetteName string, client *http.Client) *govcr.ControlPanel {
	return govcr.NewVCR(
		govcr.NewCassetteLoader(cassetteName),
		govcr.WithClient(client))
}

func generateBinaryBody(sequence int8) []byte {
	data := make([]byte, 256, 257)
	for i := range data {
		data[i] = byte(i)
	}
	data = append(data, byte(sequence))
	return data
}

func validateResponseForTestPlaybackOrder(resp *http.Response, expectedBody interface{}) error {
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("resp.StatusCode: Expected %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if resp.Body == nil {
		return errors.New("resp.Body: Expected non-nil, got nil")
	}

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Errorf("err from io.ReadAll(): Expected nil, got %s", err)
	}
	_ = resp.Body.Close()

	var expectedBodyBytes []byte
	switch exp := expectedBody.(type) {
	case []byte:
		expectedBodyBytes = exp

	case string:
		expectedBodyBytes = []byte(exp)

	default:
		return errors.New("Unexpected type for 'expectedBody' variable")
	}

	if !bytes.Equal(bodyData, expectedBodyBytes) {
		return errors.Errorf("Body: expected '%v', got '%v'", expectedBody, bodyData)
	}

	return nil
}
