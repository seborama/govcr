package govcr_test

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/seborama/govcr"
)

func TestPlaybackOrder(t *testing.T) {
	cassetteName := "TestPlaybackOrder"
	clientNum := int8(1)

	// create a test server
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, client %d", clientNum)
		clientNum++
	}))

	fmt.Println("Phase 1 ================================================")

	if err := govcr.DeleteCassette(cassetteName, ""); err != nil {
		t.Fatalf("err from govcr.DeleteCassette(): Expected nil, got %s", err)
	}

	vcr := createVCR(cassetteName, false)
	client := vcr.Client

	// run requests
	for i := int32(1); i <= 10; i++ {
		resp, _ := client.Get(ts.URL)

		// check outcome of the request
		expectedBody := fmt.Sprintf("Hello, client %d", i)
		if err := validateResponseForTestPlaybackOrder(resp, expectedBody); err != nil {
			t.Fatal(err.Error())
		}

		if !govcr.CassetteExistsAndValid(cassetteName, "") {
			t.Fatalf("CassetteExists: expected true, got false")
		}

		if err := validateStats(vcr.Stats(), 0, i, 0); err != nil {
			t.Fatal(err.Error())
		}
	}

	fmt.Println("Phase 2 - Playback =====================================")
	clientNum = 1

	// re-run request and expect play back from vcr
	vcr = createVCR(cassetteName, false)
	client = vcr.Client

	// run requests
	for i := int32(1); i <= 10; i++ {
		resp, _ := client.Get(ts.URL)

		// check outcome of the request
		expectedBody := fmt.Sprintf("Hello, client %d", i)
		if err := validateResponseForTestPlaybackOrder(resp, expectedBody); err != nil {
			t.Fatal(err.Error())
		}

		if !govcr.CassetteExistsAndValid(cassetteName, "") {
			t.Fatalf("CassetteExists: expected true, got false")
		}

		if err := validateStats(vcr.Stats(), 10, 0, i); err != nil {
			t.Fatal(err.Error())
		}
	}
}

func TestNonUtf8EncodableBinaryBody(t *testing.T) {
	cassetteName := "TestNonUtf8EncodableBinaryBody"
	clientNum := int8(1)

	// create a test server
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := generateBinaryBody(clientNum)
		written, err := w.Write(data)
		if written != len(data) {
			t.Fatalf("** Only %d bytes out of %d were written", written, len(data))
		}
		if err != nil {
			t.Fatalf("err from w.Write(): Expected nil, got %s", err)
		}
		clientNum++
	}))

	fmt.Println("Phase 1 ================================================")

	if err := govcr.DeleteCassette(cassetteName, ""); err != nil {
		t.Fatalf("err from govcr.DeleteCassette(): Expected nil, got %s", err)
	}

	vcr := createVCR(cassetteName, false)
	client := vcr.Client

	// run requests
	for i := int8(1); i <= 10; i++ {
		resp, _ := client.Get(ts.URL)

		// check outcome of the request
		expectedBody := generateBinaryBody(i)
		if err := validateResponseForTestPlaybackOrder(resp, expectedBody); err != nil {
			t.Fatal(err.Error())
		}

		if !govcr.CassetteExistsAndValid(cassetteName, "") {
			t.Fatalf("CassetteExists: expected true, got false")
		}

		if err := validateStats(vcr.Stats(), 0, int32(i), 0); err != nil {
			t.Fatal(err.Error())
		}
	}

	fmt.Println("Phase 2 - Playback =====================================")
	clientNum = 1

	// re-run request and expect play back from vcr
	vcr = createVCR(cassetteName, false)
	client = vcr.Client

	// run requests
	for i := int32(1); i <= 10; i++ {
		resp, _ := client.Get(ts.URL)

		// check outcome of the request
		expectedBody := generateBinaryBody(int8(i))
		if err := validateResponseForTestPlaybackOrder(resp, expectedBody); err != nil {
			t.Fatal(err.Error())
		}

		if !govcr.CassetteExistsAndValid(cassetteName, "") {
			t.Fatalf("CassetteExists: expected true, got false")
		}

		if err := validateStats(vcr.Stats(), 10, 0, i); err != nil {
			t.Fatal(err.Error())
		}
	}
}

func TestLongPlay(t *testing.T) {
	cassetteName := t.Name() + ".gz"
	clientNum := int8(1)

	// create a test server
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, client %d", clientNum)
		clientNum++
	}))

	fmt.Println("Phase 1 ================================================")

	if err := govcr.DeleteCassette(cassetteName, ""); err != nil {
		t.Fatalf("err from govcr.DeleteCassette(): Expected nil, got %s", err)
	}

	vcr := createVCR(cassetteName, true)
	client := vcr.Client

	// run requests
	for i := int32(1); i <= 10; i++ {
		resp, _ := client.Get(ts.URL)

		// check outcome of the request
		expectedBody := fmt.Sprintf("Hello, client %d", i)
		if err := validateResponseForTestPlaybackOrder(resp, expectedBody); err != nil {
			t.Fatal(err.Error())
		}

		if !govcr.CassetteExistsAndValid(cassetteName, "") {
			t.Fatalf("CassetteExists: expected true, got false")
		}

		if err := validateStats(vcr.Stats(), 0, i, 0); err != nil {
			t.Fatal(err.Error())
		}
	}

	fmt.Println("Phase 2 - Playback =====================================")
	clientNum = 1

	// re-run request and expect play back from vcr
	vcr = createVCR(cassetteName, false)
	client = vcr.Client

	// run requests
	for i := int32(1); i <= 10; i++ {
		resp, _ := client.Get(ts.URL)

		// check outcome of the request
		expectedBody := fmt.Sprintf("Hello, client %d", i)
		if err := validateResponseForTestPlaybackOrder(resp, expectedBody); err != nil {
			t.Fatal(err.Error())
		}

		if !govcr.CassetteExistsAndValid(cassetteName, "") {
			t.Fatalf("CassetteExists: expected true, got false")
		}

		if err := validateStats(vcr.Stats(), 10, 0, i); err != nil {
			t.Fatal(err.Error())
		}
	}
}

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

	fmt.Println("Phase 1 ================================================")

	if err := govcr.DeleteCassette(cassetteName, ""); err != nil {
		t.Fatalf("err from govcr.DeleteCassette(): Expected nil, got %s", err)
	}

	vcr := createVCR(cassetteName, false)
	client := vcr.Client

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

						if !govcr.CassetteExistsAndValid(cassetteName, "") {
							t.Fatalf("CassetteExists: expected true, got false")
						}
					}()
				})
			}(i)
		}
	})

	if err := validateStats(vcr.Stats(), 0, int32(threadMax), 0); err != nil {
		t.Fatal(err.Error())
	}

	fmt.Println("Phase 2 - Playback =====================================")

	// re-run request and expect play back from vcr
	vcr = createVCR(cassetteName, false)
	client = vcr.Client

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

						if !govcr.CassetteExistsAndValid(cassetteName, "") {
							t.Fatalf("CassetteExists: expected true, got false")
						}
					}()
				})
			}(i)
		}
	})

	if err := validateStats(vcr.Stats(), int32(threadMax), 0, int32(threadMax)); err != nil {
		t.Fatal(err.Error())
	}
}

func createVCR(cassetteName string, lp bool) *govcr.VCRControlPanel {
	// create a custom http.Transport.
	tr := http.DefaultTransport.(*http.Transport)
	tr.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true, // just an example, strongly discouraged
	}

	// create a vcr
	return govcr.NewVCR(cassetteName,
		&govcr.VCRConfig{
			Client:   &http.Client{Transport: tr},
			LongPlay: lp,
		})
}

func validateResponseForTestPlaybackOrder(resp *http.Response, expectedBody interface{}) error {
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("resp.StatusCode: Expected %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if resp.Body == nil {
		return fmt.Errorf("resp.Body: Expected non-nil, got nil")
	}

	bodyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("err from ioutil.ReadAll(): Expected nil, got %s", err)
	}
	resp.Body.Close()

	var expectedBodyBytes []byte
	switch expectedBody.(type) {
	case []byte:
		var ok bool
		expectedBodyBytes, ok = expectedBody.([]byte)
		if !ok {
			return fmt.Errorf("expectedBody: cannot assert to type '[]byte'")
		}

	case string:
		expectedBodyString, ok := expectedBody.(string)
		if !ok {
			return fmt.Errorf("expectedBody: cannot assert to type 'string'")
		}
		expectedBodyBytes = []byte(expectedBodyString)

	default:
		return fmt.Errorf("Unexpected type for 'expectedBody' variable")
	}

	if !bytes.Equal(bodyData, expectedBodyBytes) {
		return fmt.Errorf("Body: expected '%v', got '%v'", expectedBody, bodyData)
	}

	return nil
}

func validateStats(actualStats govcr.Stats, expectedTracksLoaded, expectedTracksRecorded, expectedTrackPlayed int32) error {
	if actualStats.TracksLoaded != expectedTracksLoaded {
		return fmt.Errorf("Expected %d track loaded, got %d", expectedTracksLoaded, actualStats.TracksLoaded)
	}

	if actualStats.TracksRecorded != expectedTracksRecorded {
		return fmt.Errorf("Expected %d track recorded, got %d", expectedTracksRecorded, actualStats.TracksRecorded)
	}

	if actualStats.TracksPlayed != expectedTrackPlayed {
		return fmt.Errorf("Expected %d track played, got %d", expectedTrackPlayed, actualStats.TracksPlayed)
	}

	return nil
}

func generateBinaryBody(sequence int8) []byte {
	data := make([]byte, 256, 257)
	for i := range data {
		data[i] = byte(i)
	}
	data = append(data, byte(sequence))
	return data
}
