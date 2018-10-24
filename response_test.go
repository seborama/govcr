package govcr

import (
	"bytes"
	"net/http"
	"net/textproto"
	"testing"
)

func failIfCalledResponseFilter(t *testing.T) ResponseFilter {
	return func(resp Response) Response {
		t.Fatal("Filter was called unexpectedly")
		return resp
	}
}

// mustCallResponseFilterOnce will return a request filter that will record
// how many times it was called.
// The returned function will test if the filter was called once and fail otherwise.
func mustCallResponseFilterOnce(t *testing.T) (ResponseFilter, func()) {
	var n int
	return func(resp Response) Response {
			n++
			return resp
		}, func() {
			if n != 1 {
				t.Fatalf("Filter was called %d times, should be called once", n)
			}
		}
}

func responseTestBase() Response {
	return Response{
		req:        requestTestBase(),
		Header:     map[string][]string{textproto.CanonicalMIMEHeaderKey("a-respheader"): {"a-value"}},
		Body:       []byte(`sample body`),
		StatusCode: http.StatusCreated,
	}
}

func TestResponse_Request(t *testing.T) {
	r := responseTestBase()
	want := requestTestBase()
	req := r.Request()
	if want.URL.String() != req.URL.String() {
		t.Errorf("Request does not match: (want) %v != (got) %v", want, req)
	}
}

func TestResponseAddHeaderValue(t *testing.T) {
	resp := ResponseAddHeaderValue("header-key", "value")(responseTestBase())
	if resp.Header.Get("header-key") != "value" {
		t.Fatalf("new header not found %+v", resp.Header)
	}
	if resp.Header.Get("a-respheader") != "a-value" {
		t.Fatalf("existing header not found %+v", resp.Header)
	}
	// Check request is untouched.
	req := resp.Request()
	if req.Header.Get("header-key") != "" {
		t.Error("'header-key' was added on request")
	}
}

func TestResponseDeleteHeaderKeys(t *testing.T) {
	r := responseTestBase()
	r.Header.Add("a-header", "a-value")
	r = ResponseDeleteHeaderKeys("non-existing", "a-respheader", "a-header")(r)
	if r.Header.Get("a-respheader") != "" {
		t.Error("'a-header' not removed")
	}
	if len(r.Header) != 0 {
		t.Errorf("want no headers, got %d (%+v)", len(r.Header), r.Header)
	}
	// Check request is untouched.
	req := r.Request()
	if req.Header.Get("a-header") != "a-value" {
		t.Error("'a-header' was changed on request")
	}
}

func TestResponseTransferHeaderKeys(t *testing.T) {
	r := ResponseTransferHeaderKeys("a-header")(responseTestBase())
	if r.Header.Get("a-header") != "a-value" {
		t.Errorf("'a-header' not transferred, %v", r.Header)
	}
}

func TestResponseChangeBody(t *testing.T) {
	r := ResponseChangeBody(func(b []byte) []byte {
		if !bytes.Equal(b, []byte(`sample body`)) {
			t.Fatalf("unexpected body: %s", string(b))
		}
		return []byte(`new body`)
	})(responseTestBase())
	if !bytes.Equal(r.Body, []byte(`new body`)) {
		t.Fatalf("unexpected body after filter: %s", string(r.Body))
	}
}

func TestResponseFilter_OnMethod(t *testing.T) {
	f := failIfCalledResponseFilter(t).OnMethod(http.MethodPost)
	f(responseTestBase())

	f, ok := mustCallResponseFilterOnce(t)
	f.OnMethod(http.MethodGet)(responseTestBase())
	ok()
}

func TestResponseFilter_OnPath(t *testing.T) {
	f := failIfCalledResponseFilter(t).OnPath(`non-existing-path`)
	f(responseTestBase())

	f, ok := mustCallResponseFilterOnce(t)
	f.OnPath(`/example-url/id/`)(responseTestBase())
	ok()

	// Empty matches everything
	f, ok = mustCallResponseFilterOnce(t)
	f = f.OnPath("")
	f(responseTestBase())
	ok()
}

func TestResponseFilter_OnStatus(t *testing.T) {
	f := failIfCalledResponseFilter(t).OnStatus(http.StatusNotFound)
	f(responseTestBase())

	f, ok := mustCallResponseFilterOnce(t)
	f.OnStatus(http.StatusCreated)(responseTestBase())
	ok()
}

func TestResponseFilter_Add(t *testing.T) {
	var f ResponseFilters
	f1, ok1 := mustCallResponseFilterOnce(t)
	f2, ok2 := mustCallResponseFilterOnce(t)
	f.Add(f1, f2)
	f.combined()(responseTestBase())
	ok1()
	ok2()
}

func TestResponseFilter_Prepend(t *testing.T) {
	var f ResponseFilters
	var firstRan bool
	first := func(r Response) Response {
		firstRan = true
		return r
	}
	second := func(r Response) Response {
		if !firstRan {
			t.Fatal("second ran before first")
		}
		return r
	}
	third := func(r Response) Response {
		if !firstRan {
			t.Fatal("third ran before first")
		}
		return r
	}
	f.Add(second)
	f.Prepend(first)
	f.Add(third)
	f.combined()(responseTestBase())
}

func TestResponseFilter_combined(t *testing.T) {
	var f ResponseFilters
	f1, ok1 := mustCallResponseFilterOnce(t)
	f2, ok2 := mustCallResponseFilterOnce(t)
	f3, ok3 := mustCallResponseFilterOnce(t)
	f.Add(f1, f2, f3)
	f.combined()(responseTestBase())
	ok1()
	ok2()
	ok3()
}
