package govcr

import (
	"net/http"
	"net/textproto"
	"net/url"
	"testing"
)

func mustParseURL(s string) url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return *u
}

func failIfCalledRequestFilter(t *testing.T) RequestFilter {
	return func(req Request) Request {
		t.Fatal("Filter was called unexpectedly")
		return req
	}
}

// mustCallRequestFilterOnce will return a request filter that will record
// how many times it was called.
// The returned function will test if the filter was called once and fail otherwise.
func mustCallRequestFilterOnce(t *testing.T) (RequestFilter, func()) {
	var n int
	return func(req Request) Request {
			n++
			return req
		}, func() {
			if n != 1 {
				t.Fatalf("Filter was called %d times, should be called once", n)
			}
		}
}

func requestTestBase() Request {
	return Request{
		Header: map[string][]string{textproto.CanonicalMIMEHeaderKey("a-header"): {"a-value"}},
		Body:   nil,
		Method: http.MethodGet,
		URL:    mustParseURL("https://127.0.0.1/example-url/id/42"),
	}
}

func TestRequestFilter_OnMethod(t *testing.T) {
	f := failIfCalledRequestFilter(t).OnMethod(http.MethodPost)
	f(requestTestBase())

	f, ok := mustCallRequestFilterOnce(t)
	f = f.OnMethod(http.MethodGet)
	f(requestTestBase())
	ok()
}

func TestRequestFilter_OnPath(t *testing.T) {
	f := failIfCalledRequestFilter(t).OnPath("non-existing-path")
	f(requestTestBase())

	f, ok := mustCallRequestFilterOnce(t)
	f = f.OnPath(`/example-url/id/`)
	f(requestTestBase())
	ok()

	// Empty matches everything
	f, ok = mustCallRequestFilterOnce(t)
	f = f.OnPath("")
	f(requestTestBase())
	ok()
}

func TestRequestAddHeaderValue(t *testing.T) {
	r := RequestAddHeaderValue("new-header", "new-value")(requestTestBase())
	if r.Header.Get("new-header") != "new-value" {
		t.Error("did not get expected new header")
	}
	// Check if existing is still untouched.
	if r.Header.Get("a-header") != "a-value" {
		t.Error("did not get expected old header")
	}
}

func TestRequestDeleteHeaderKeys(t *testing.T) {
	r := RequestDeleteHeaderKeys("non-existing", "a-header")(requestTestBase())
	if r.Header.Get("a-header") != "" {
		t.Error("'a-header' not removed")
	}
	if len(r.Header) != 0 {
		t.Errorf("want no headers, got %d (%+v)", len(r.Header), r.Header)
	}
}

func TestRequestExcludeHeaderFunc(t *testing.T) {
	req := requestTestBase()
	req.Header.Add("another-header", "yeah")
	header1, header2 := textproto.CanonicalMIMEHeaderKey("a-header"), textproto.CanonicalMIMEHeaderKey("another-header")

	// We expect both headers to be checked.
	want := map[string]struct{}{header1:{}, header2: {}}
	r := RequestExcludeHeaderFunc(func(key string) bool {
		_, ok := want[key]
		if !ok {
			t.Errorf("got unexpected key %q", key)
		}
		// Delete so we check we only get key once.
		delete(want, key)
		// Delete 'a-header'
		return header1 == key
	})
	req = r(req)
	if len(want) > 0 {
		t.Errorf("header was not checked: %v", want)
	}
	if len(req.Header) != 1 {
		t.Fatalf("unexpected header count, want one: %v", req.Header)
	}
	if req.Header.Get("another-header") != "yeah" {
		t.Errorf("unexpected header value: %s", req.Header.Get("another-header"))
	}
}

func TestRequestFilters_Add(t *testing.T) {
	var f RequestFilters
	f1, ok1 := mustCallRequestFilterOnce(t)
	f2, ok2 := mustCallRequestFilterOnce(t)
	f.Add(f1, f2)
	f.combined()(requestTestBase())
	ok1()
	ok2()
}

func TestRequestFilters_Prepend(t *testing.T) {
	var f RequestFilters
	var firstRan bool
	first := func(req Request) Request {
		firstRan = true
		return req
	}
	second := func(req Request) Request {
		if !firstRan {
			t.Fatal("second ran before first")
		}
		return req
	}
	third := func(req Request) Request {
		if !firstRan {
			t.Fatal("third ran before first")
		}
		return req
	}
	f.Add(second)
	f.Prepend(first)
	f.Add(third)
	f.combined()(requestTestBase())
}

func TestRequestFilters_combined(t *testing.T) {
	var f RequestFilters
	f1, ok1 := mustCallRequestFilterOnce(t)
	f2, ok2 := mustCallRequestFilterOnce(t)
	f3, ok3 := mustCallRequestFilterOnce(t)
	f.Add(f1, f2, f3)
	f.combined()(requestTestBase())
	ok1()
	ok2()
	ok3()
}
