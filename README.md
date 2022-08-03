# govcr

Records and replays HTTP / HTTPS interactions for offline unit / behavioural / integration tests thereby acting as an HTTP mock.

This project was inspired by [php-vcr](https://github.com/php-vcr/php-vcr) which is a PHP port of [VCR](https://github.com/vcr/vcr) for ruby.

This project is an adaptation for Google's Go / Golang programming language.

## Simple VCR example

```go
// See TestExample1 in tests for full working example

func TestExample1() {
	vcr := govcr.NewVCR(
        govcr.WithCassette("MyCassette1.json"),
        govcr.WithRequestMatcher(govcr.NewMethodURLRequestMatcher()), // use a "relaxed" request matcher
    )

    vcr.Client.Get("http://example.com/foo")
}
```

The **first time** you run this example, `MyCassette1.json` won't exist and `TestExample1` will make a live HTTP call.

On **subsequent executions** (unless you delete the cassette file), the HTTP call will be played back from the cassette and no live HTTP call will occur.

Note:

We use a "relaxed" request matcher because `example.com` injects an "`Age`" header that varies per-request. Without a mutator, govcr's default strict matcher would not match the track on the cassette and keep sending live requests (and record them to the cassette).

## Install

```bash
go get github.com/seborama/govcr/v6@latest
```

For all available releases, please check the [releases](https://github.com/seborama/govcr/releases) tab on github.

And your source code would use this import:

```go
import "github.com/seborama/govcr/v6"
```

For versions of **govcr** before v5 (which don't use go.mod), use a dependency manager to lock the version you wish to use (perhaps v4)!

```bash
# download legacy version of govcr (without go.mod)
go get gopkg.in/seborama/govcr.v4
```

## Glossary of Terms

**VCR**: Video Cassette Recorder. In this context, a VCR refers to the engine and data that this project provides. A VCR is both an HTTP recorder and player. When you use a VCR, HTTP requests are replayed from previous recordings (**tracks** saved in **cassette** files on the filesystem). When no previous recording exists for the request, it is performed live on the HTTP server, after what it is saved to a **track** on the **cassette**.

**cassette**: a sequential collection of **tracks**. This is in effect a JSON file.

**Long Play cassette**: a cassette compressed in gzip format. Such cassettes have a name that ends with '`.gz`'.

**tracks**: a record of an HTTP request. It contains the request data, the response data, if available, or the error that occurred.

**ControlPanel**: the creation of a VCR instantiates a ControlPanel for interacting with the VCR and conceal its internals.

## Documentation

**govcr** is a wrapper around the Go `http.Client`. It can record live HTTP traffic to files (called "**cassettes**") and later replay HTTP requests ("**tracks**") from them instead of live HTTP calls.

The code documentation can be found on [godoc](http://pkg.go.dev/github.com/seborama/govcr).

When using **govcr**'s `http.Client`, the request is matched against the **tracks** on the '**cassette**':

- The **track** is played where a matching one exists on the **cassette**,
- otherwise the request is executed live to the HTTP server and then recorded on **cassette** for the next time.

**Note on a govcr typical flow**

The normal **govcr** flow is test-oriented. Traffic is recorded by default unless a track already existed on the cassette **at the time it was loaded**.

A typical usage:
- run your test once to produce the cassette
- from this point forward, when the test runs again, it will use the cassette

During live recording, the same request can be repeated and recorded many times. Playback occurs in the order the requests were saved on the cassette. See the tests for an example (`TestConcurrencySafety`).

### VCRSettings

This structure contains parameters for configuring your **govcr** recorder.

Settings are populated via `With*` options:

- Use `WithClient` to provide a custom http.Client otherwise the default Go http.Client will be used.
- `WithCassette` loads the specified cassette.\
  Note that it is also possible to call `LoadCassette` from the vcr instance.
- See `vcrsettings.go` for more options such as `WithRequestMatcher`, `WithTrackRecordingMutators`, `WithTrackReplayingMutators`, ...
- TODO in v5: `WithLogging` enables logging to help understand what govcr is doing internally.

## Match a request to a cassette track

By default, **govcr** uses a strict `RequestMatcher` function that compares the request's headers, method, full URL, body, and trailers.

Another RequestMatcher (obtained with `NewMethodURLRequestMatcher`) provides a more relaxed comparison based on just the method and the full URL.

In some scenarios, it may not possible to match **tracks** exactly as they were recorded.

This may be the case when the request contains a timestamp or a dynamically changing identifier, etc.

You can create your own matcher on any part of the request and in any manner (like ignoring or modifying some headers, etc).

## Track mutators

The live HTTP request and response traffic is protected against modifications. While **govcr** could easily support in-place mutation of the live traffic, this is not a goal.

Nonetheless, **govcr** supports mutating tracks, either at **recording time** or at **playback time**.

In either case, this is achieved with track `Mutators`.

A `Mutator` can be combined with one or more `On` conditions. At present, all `On` conditions attached to a mutator must be true for the mutator to apply.

A **track recording mutator** can change both the request and the response that will be persisted to the cassette.

A **track replaying mutator** transforms the track after it was matched and retrieved from the cassette. It does not change the cassette file.

While a track replaying mutator could change the request, it serves no purpose since the request has already been made and matched to a track by the time the replaying mutator is invoked. The reason for supplying the request in the replaying mutator is for information. In some situations, the request details are needed to transform the response.

Refer to the tests for examples (search for `WithTrackRecordingMutators` and `WithTrackReplayingMutators`).

## Cookbook

### Run the examples

Please refer to the `examples` directory for examples of code and uses.

**Observe the output of the examples between the `1st run` and the `2nd run` of each example.**

The **first time** they run, they perform a live HTTP call (`Executing request to live server`).

However, on **second execution** (and subsequent executions as long as the **cassette** is not deleted)
**govcr** retrieves the previously recorded request and plays it back without live HTTP call (`Found a matching track`). You can disconnect from the internet and still playback HTTP requests endlessly!

### Recipe: VCR with custom `http.Client`

Sometimes, your application will create its own `http.Client` wrapper (for observation, etc) or will initialise the `http.Client`'s Transport (for instance when using https).

In such cases, you can pass the `http.Client` object of your application to VCR.

VCR will wrap your `http.Client`. You should use `vcr.HTTPClient()` in your tests when making HTTP calls.

```go
// See TestExample2 in tests for full working example

func TestExample2() {
	// Create a custom http.Transport for our app.
	tr := http.DefaultTransport.(*http.Transport)
	tr.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true, // just an example, not recommended
	}

	// Create an instance of myApp.
	// It uses the custom Transport created above and a custom Timeout.
	app := &myApp{
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   15 * time.Second,
		},
	}

	// Instantiate VCR.
	vcr := govcr.NewVCR(
		govcr.WithCassette(exampleCassetteName2),
		govcr.WithClient(app.httpClient),
	)

	// Inject VCR's http.Client wrapper.
	// The original transport has been preserved, only just wrapped into VCR's.
	app.httpClient = vcr.HTTPClient()

	// Run request and display stats.
	app.Get("https://example.com/foo")
}
```

### Recipe: Remove Response TLS

Use the provided mutator `track.ResponseDeleteTLS`.

Remove Response.TLS from the cassette **recording**:

```go
	vcr := govcr.NewVCR(
		govcr.WithCassette(exampleCassetteName2),
		govcr.WithTrackRecordingMutators(track.ResponseDeleteTLS()),
        //             ^^^^^^^^^
	)
    // or, similarly:
    vcr.AddRecordingMutators(track.ResponseDeleteTLS())
    //     ^^^^^^^^^
```

Remove Response.TLS from the track at **playback** time:

```go
	vcr := govcr.NewVCR(
		govcr.WithCassette(exampleCassetteName2),
		govcr.WithTrackReplayingMutators(track.ResponseDeleteTLS()),
        //             ^^^^^^^^^
	)
    // or, similarly:
    vcr.AddReplayingMutators(track.ResponseDeleteTLS())
    //     ^^^^^^^^^
```

### Recipe: Change the playback mode of the VCR

**govcr** support operation modes:

- Live only: never replay from the cassette.
- Read only: normal behaviour except that recording to cassette is disabled.
- Offline: playback from cassette only, return a transport error if no track matches.

#### Live only

```go
	vcr := govcr.NewVCR(
		govcr.WithCassette(exampleCassetteName2),
		govcr.WithLiveOnlyMode(),
	)
    // or equally:
    vcr.SetLiveOnlyMode(true) // `false` to disable option
```

#### Read only

```go
	vcr := govcr.NewVCR(
		govcr.WithCassette(exampleCassetteName2),
		govcr.WithReadOnlyMode(),
	)
    // or equally:
    vcr.SetReadOnlyMode(true) // `false` to disable option
```

#### Offline

```go
	vcr := govcr.NewVCR(
		govcr.WithCassette(exampleCassetteName2),
		govcr.WithOfflineMode(),
	)
    // or equally:
    vcr.SetOfflineMode(true) // `false` to disable option
```

### Recipe: VCR with a RequestFilter

**TODO: THIS EXAMPLE FOR v4 NOT v5**

This example shows how to handle situations where a header in the request needs to be ignored (or the **track** would not match and hence would not be replayed).

For this example, logging is switched on. This is achieved with `Logging: true` in `VCRSettings` when calling `NewVCR`.

```go
package main

import (
    "fmt"
    "strings"
    "time"

    "net/http"

    "github.com/seborama/govcr/v6"
)

const example4CassetteName = "MyCassette4"

// Example4 is an example use of govcr.
// The request contains a custom header 'X-Custom-My-Date' which varies with every request.
// This example shows how to exclude a particular header from the request to facilitate
// matching a previous recording.
// Without the RequestFilters, the headers would not match and hence the playback would not
// happen!
func Example4() {
    vcr := govcr.NewVCR(example4CassetteName,
        &govcr.VCRSettings{
            RequestFilters: govcr.RequestFilters{
                govcr.RequestDeleteHeaderKeys("X-Custom-My-Date"),
            },
            Logging: true,
        })

    // create a request with our custom header
    req, err := http.NewRequest("POST", "http://example.com/foo", nil)
    if err != nil {
        fmt.Println(err)
    }
    req.Header.Add("X-Custom-My-Date", time.Now().String())

    // run the request
    vcr.Client.Do(req)
    fmt.Printf("%+v\n", vcr.Stats())
}
```

**Tip:**

Remove the RequestFilters from the VCRSettings and re-run the example. Check the stats: notice how the tracks **no longer** replay.

### Recipe: VCR with a recoding Track Mutator

**TODO: THIS EXAMPLE FOR v4 NOT v5**

This example shows how to handle situations where a transaction Id in the header needs to be present in the response.
This could be as part of a contract validation between server and client.

Note: This is useful when some of the data in the **request** Header / Body needs to be transformed
      before it can be evaluated for comparison for playback.

```go
package main

import (
    "fmt"
    "strings"
    "time"

    "net/http"

    "github.com/seborama/govcr/v6"
)

const example5CassetteName = "MyCassette5"

// Example5 is an example use of govcr.
// Supposing a fictional application where the request contains a custom header
// 'X-Transaction-Id' which must be matched in the response from the server.
// When replaying, the request will have a different Transaction Id than that which was recorded.
// Hence the protocol (of this fictional example) is broken.
// To circumvent that, we inject the new request's X-Transaction-Id into the recorded response.
// Without the ResponseFilters, the X-Transaction-Id in the header would not match that
// of the recorded response and our fictional application would reject the response on validation!
func Example5() {
    vcr := govcr.NewVCR(example5CassetteName,
        &govcr.VCRSettings{
            RequestFilters: govcr.RequestFilters{
                govcr.RequestDeleteHeaderKeys("X-Transaction-Id"),
            },
			ResponseFilters: govcr.ResponseFilters{
				// overwrite X-Transaction-Id in the Response with that from the Request
				govcr.ResponseTransferHeaderKeys("X-Transaction-Id"),
			},
            Logging: true,
        })

    // create a request with our custom header
    req, err := http.NewRequest("POST", "http://example.com/foo5", nil)
    if err != nil {
        fmt.Println(err)
    }
    req.Header.Add("X-Transaction-Id", time.Now().String())

    // run the request
    resp, err := vcr.Client.Do(req)
    if err != nil {
        fmt.Println(err)
    }

    // verify outcome
    if req.Header.Get("X-Transaction-Id") != resp.Header.Get("X-Transaction-Id") {
        fmt.Println("Header transaction Id verification failed - this would be the live request!")
    } else {
        fmt.Println("Header transaction Id verification passed - this would be the replayed track!")
    }

    fmt.Printf("%+v\n", vcr.Stats())
}
```

### Recipe: VCR with a replaying Track Mutator

**TODO: add example that includes the use of `.On*` predicates**

## Stats

VCR provides some statistics.

To access the stats, call `vcr.Stats()` where vcr is the `ControlPanel` instance obtained from `NewVCR(...)`.

## Run the tests

```bash
make test
```

## Bugs

- The recording of TLS data for PublicKeys is not reliable owing to a limitation in Go's json package and a non-deterministic and opaque use of a blank interface in Go's certificate structures. Some improvements are possible with `gob`.

## Improvements

- When unmarshaling the cassette fails, rather than fail altogether, it would be preferable to revert to live HTTP call.

- The code has a number of TODO's which should either be taken action upon or removed!

## Limitations

### Go empty interfaces (`interface{}`)

Some properties / objects in http.Response are defined as `interface{}` (or `any`).

This can cause `json.Unmarshal` to fail (example: when the original type was `big.Int` with a big integer indeed - `json.Unmarshal` attempts to convert to float64 and fails).

Currently, this is dealt with by converting the output of the JSON produced by `json.Marshal` (big.Int is changed to a string).

### Support for multiple values in HTTP headers

Repeat HTTP headers may not be properly handled. A long standing TODO in the code exists but so far no one has complained :-)

### HTTP transport errors

**govcr** also records `http.Client` errors (network down, blocking firewall, timeout, etc) in the **track** for future playback.

Since `errors` is an interface, when it is unmarshalled into JSON, the Go type of the `error` is lost.

To circumvent this, **govcr** serialises the object type (`ErrType`) and the error message (`ErrMsg`) in the **track** record.

Objects cannot be created by name at runtime in Go. Rather than re-create the original error object, *govcr* creates a standard error object with an error string made of both the `ErrType` and `ErrMsg`.

In practice, the implications for you depend on how much you care about the error type. If all you need to know is that an error occurred, you won't mind this limitation.

Mitigation: Support for common errors (network down) has been implemented. Support for more error types can be implemented, if there is appetite for it.

## Contribute

You are welcome to submit a PR to contribute.

Please try and follow a TDD workflow: tests must be present and as much as is practical to you, avoid toxic DDT (development driven testing).
