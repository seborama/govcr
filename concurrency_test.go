package govcr_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/seborama/govcr"
	"github.com/stretchr/testify/require"
)

func TestConcurrencySafety(t *testing.T) {
	cassetteName := "TestConcurrencySafety"
	threadMax := int8(50)

	// create a test server
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(50)))

		clientNum, _ := strconv.ParseInt(r.URL.Query().Get("num"), 0, 8)

		data := generateBinaryBody(int8(clientNum))
		written, err := w.Write(data)
		if written != len(data) {
			t.Fatalf("** Only %d bytes out of %d were written", written, len(data))
		}
		if err != nil {
			t.Fatalf("err from w.Write(): Expected nil, got %s", err)
		}
	}))

	testServerClient := ts.Client()
	testServerClient.Timeout = 3 * time.Second

	fmt.Println("Phase 1 ================================================")

	_ = os.Remove(cassetteName)
	defer func() { _ = os.Remove(cassetteName) }()

	vcr := createVCR(cassetteName, testServerClient, false)
	client := vcr.Player()

	t.Run("main - phase 1", func(t *testing.T) {
		// run requests
		for i := int8(1); i <= threadMax; i++ {
			func(i1 int8) {
				t.Run(fmt.Sprintf("i=%d", i), func(t *testing.T) {
					t.Parallel()

					func() {
						resp, _ := client.Get(fmt.Sprintf("%s?num=%d", ts.URL, i1))

						// check outcome of the request
						expectedBody := generateBinaryBody(i1)
						err := validateResponseForTestPlaybackOrder(resp, expectedBody)
						require.NoError(t, err)
					}()
				})
			}(i)
		}
	})

	// err := vcr.LoadCassette(cassetteName)
	// require.NoError(t, err)
	// assert.EqualValues(t, threadMax, vcr.NumberOfTracks())
	expectedStats := govcr.Stats{
		TracksLoaded:   0,
		TracksRecorded: int32(threadMax),
		TracksPlayed:   0,
	}
	require.EqualValues(t, expectedStats, *vcr.Stats())
	vcr.EjectCassette()

	fmt.Println("Phase 2 - Playback =====================================")

	// re-run request and expect play back from vcr
	vcr = createVCR(cassetteName, testServerClient, false)
	client = vcr.Player()

	// run requests
	t.Run("main - phase 1", func(t *testing.T) {
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

	expectedStats = govcr.Stats{
		TracksLoaded:   int32(threadMax),
		TracksRecorded: 0,
		TracksPlayed:   int32(threadMax),
	}
	require.EqualValues(t, expectedStats, *vcr.Stats())
}

func createVCR(cassetteName string, client *http.Client, lp bool) *govcr.ControlPanel {
	return govcr.NewVCR(
		govcr.WithClient(client),
		govcr.WithCassette(cassetteName))
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

	bodyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Errorf("err from ioutil.ReadAll(): Expected nil, got %s", err)
	}
	_ = resp.Body.Close()

	var expectedBodyBytes []byte
	switch expectedBody.(type) {
	case []byte:
		var ok bool
		expectedBodyBytes, ok = expectedBody.([]byte)
		if !ok {
			return errors.Errorf("expectedBody: cannot assert to type '[]byte'")
		}

	case string:
		expectedBodyString, ok := expectedBody.(string)
		if !ok {
			return errors.Errorf("expectedBody: cannot assert to type 'string'")
		}
		expectedBodyBytes = []byte(expectedBodyString)

	default:
		return errors.New("Unexpected type for 'expectedBody' variable")
	}

	if !bytes.Equal(bodyData, expectedBodyBytes) {
		return errors.Errorf("Body: expected '%v', got '%v'", expectedBody, bodyData)
	}

	return nil
}
