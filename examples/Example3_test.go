package examples_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/seborama/govcr/v8"
	"github.com/seborama/govcr/v8/cassette/track"
	"github.com/stretchr/testify/require"
)

const exampleCassetteName3 = "temp-fixtures/TestExample3.cassette.json"

// TestExample3 is an example use of govcr in a situation where a request-specific transaction ID is exchanged
// between the server and the client.
// There exist multiple ways to achieve this. This is only one possibility.
func TestExample3(t *testing.T) {
	// Instantiate VCR.
	vcr := govcr.NewVCR(
		govcr.WithCassette(exampleCassetteName3),
		govcr.WithRequestMatcher(
			govcr.NewBlankRequestMatcher(
				govcr.WithRequestMatcherFunc(
					func(httpRequest, trackRequest *track.Request) bool {
						// Remove the header from comparison.
						// Note: this removal is only scoped to the request matcher, it does not affect the original HTTP request
						httpRequest.Header.Del("X-Transaction-Id")
						trackRequest.Header.Del("X-Transaction-Id")

						return govcr.DefaultHeaderMatcher(httpRequest, trackRequest)
					},
				),
			),
		),
		govcr.WithTrackReplayingMutators(
			// Note: although we deleted the headers in the request matcher, this was limited to the scope of
			// the request matcher. The replaying mutator's scope is past request matching.
			track.ResponseDeleteHeaderKeys("X-Transaction-Id"), // do not append to existing values
			track.ResponseTransferHTTPHeaderKeys("X-Transaction-Id"),
		),
	)

	defer func() {
		// Display govcr Stats
		t.Logf("%+v\n", vcr.Stats())
	}()

	// Start mock server
	serverURL := mockServer()

	// Run request, we will receive Status Created.
	txID := uuid.NewString()

	req, err := http.NewRequest(http.MethodGet, serverURL+"/create", nil)
	require.NoError(t, err)
	req.Header.Set("X-Transaction-Id", txID)

	resp, err := vcr.HTTPClient().Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.Equal(t, txID, resp.Header.Get("X-Transaction-Id"))

	// Repeat the request, this time we'll get Status Conflict.
	req, err = http.NewRequest(http.MethodGet, serverURL+"/get", nil)
	require.NoError(t, err)
	req.Header.Set("X-Transaction-Id", txID)
	require.Equal(t, txID, resp.Header.Get("X-Transaction-Id"))

	resp, err = vcr.HTTPClient().Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

//
// Note: code past this point is purely to support the example
// There is no value in reading this from a govcr point-of-view.
//

func mockServer() string {
	txns := map[string]struct{}{}

	// Create a basic test server.
	// The server accepts a query param of "txid" or it will return HTTP Bad Request.
	// When the provided ID is new, the server will return HTTP Created.
	// When the provided ID is recognised, the server will return HTTP Conflict.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		txID := r.Header.Get("X-Transaction-Id")
		if txID == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "missing 'txid' parameter")
			return
		}

		if _, ok := txns[txID]; ok {
			w.Header().Set("X-Transaction-Id", txID)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "txid exists: %s\n", txID)
			return
		}

		txns[txID] = struct{}{}
		w.Header().Set("X-Transaction-Id", txID)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "created new txid: %s\n", txID)
	}))

	return ts.URL
}
