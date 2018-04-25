# govcr

Records and replays HTTP interactions for offline unit / behavioural / integration tests thereby acting as an HTTP mock.

This project was inspired by [php-vcr](https://github.com/php-vcr/php-vcr) which is a PHP port of [VCR](https://github.com/vcr/vcr) for ruby.

This project is an adaptation for Google's Go / Golang programming language.

## Install

```bash
go get github.com/seborama/govcr
```

For all available releases, please check the [releases](https://github.com/seborama/govcr/releases) tab on github.

You can pick a specific major release for compatibility. For example, to use a v2.x release, use this command:

```bash
go get gopkg.in/seborama/govcr.v2
```

And your source code would use this import:

```go
import "gopkg.in/seborama/govcr.v2"
```

## Glossary of Terms

**VCR**: Video Cassette Recorder. In this context, a VCR refers to the overall engine and data that this project provides. A VCR is both an HTTP recorder and player. When you use a VCR, HTTP requests are replayed from previous recordings (**tracks** saved in **cassette** files on the filesystem). When no previous recording exists for the request, it is performed live on the HTTP server the request is intended to, after what it is saved to a **track** on the **cassette**.

**cassette**: a sequential collection of **tracks**. This is in effect a JSON file saved under directory `./govcr-fixtures` (default). The **cassette** is given a name when creating the **VCR** which becomes the filename (with an extension of `.cassette`).

**tracks**: a record of an HTTP request. It contains the request data, the response data, if available, or the error that occurred.

**PCB**: Printed Circuit Board. This is an analogy that refers to the ability to supply customisations to some aspects of the behaviour of the **VCR** (for instance, disable recordings or ignore certain HTTP headers in the request when looking for a previously recorded **track**).

## Documentation

**govcr** is a wrapper around the Go `http.Client` which offers the ability to replay pre-recorded HTTP requests ('**tracks**') instead of live HTTP calls.

**govcr** can replay both successful and failed HTTP transactions.

A given request may be repeated, again and again. They will be replayed in the same order as they were recorded. See the tests for an example (`TestPlaybackOrder`).

The code documentation can be found on [godoc](http://godoc.org/github.com/seborama/govcr).

When using **govcr**'s `http.Client`, the request is matched against the **tracks** on the '**cassette**':

- The **track** is played where a matching one exists on the **cassette**,
- or the request is executed live to the HTTP server and then recorded on **cassette** for the next time.

**Cassette** recordings are saved under `./govcr-fixtures` (by default) as `*.cassette` files in JSON format.

### VCRConfig

This structure contains parameters for configuring your **govcr** recorder.

#### `VCRConfig.CassettePath` - change the location of **cassette** files

Example:

```go
    vcr := govcr.NewVCR("MyCassette",
        &govcr.VCRConfig{
            CassettePath: "./govcr-fixtures",
        })
```

#### `VCRConfig.DisableRecording` - playback or execute live without recording

Example:

```go
    vcr := govcr.NewVCR("MyCassette",
        &govcr.VCRConfig{
            DisableRecording: true,
        })
```

In this configuration, govcr will still playback from **cassette** when a previously recorded **track** (HTTP interaction) exists or execute the request live if not. But in the latter case, it won't record a new **track** as per default behaviour.

#### `VCRConfig.Logging` - disable logging

Example:

```go
    vcr := govcr.NewVCR("MyCassette",
        &govcr.VCRConfig{
            Logging: false,
        })
```

This simply redirects all **govcr** logging to the OS's standard Null device (e.g. `nul` on Windows, or `/dev/null` on UN*X, etc).

## Features

- Record extensive details about the request, response or error (network error, timeout, etc) to provide as accurate a playback as possible compared to the live HTTP request.

- Recordings are JSON files and can be read in an editor.

- Custom Go `http.Client`'s can be supplied.

- Custom Go `http.Transport` / `http.RoundTrippers`.

- http / https supported and any other protocol implemented by the supplied `http.Client`'s `http.RoundTripper`.

- Hook to define HTTP headers that should be ignored from the HTTP request when attemtping to retrieve a **track** for playback.
  This is useful to deal with non-static HTTP headers (for example, containing a timestamp).

- Hook to transform the Header / Body of an HTTP request to deal with non-static data. The purpose is similar to the hook for headers described above but with the ability to modify the data.

- Hook to transform the Header / Body of the HTTP response to deal with non-static data. This is similar to the request hook however, the header / body of the request are also supplied (read-only) to help match data in the response with data in the request (such as a transaction Id).

- Ability to switch off automatic recordings.
  This allows to play back existing records or make
  a live HTTP call without recording it to the **cassette**.

- Record SSL certificates.

## Filter functions

### Influencing request comparison programatically at runtime.

`RequestFilterFunc` receives the request Header / Body to allow their transformation. Both the live request  and the replayed request are filtered at comparison time. **Transformations are not persisted and only for the purpose of influencing comparison**.

### Runtime transforming of the response before sending it back to the client.

`ResponseFilterFunc` is the flip side of `RequestFilterFunc`. It receives the response Header / Body to allow their transformation. Unlike `RequestFilterFunc`, this influences the response returned from the request to the client. The request header is also passed to `ResponseFilterFunc` but read-only and solely for the purpose of extracting request data for situations where it is needed to transform the Response.

## Examples

### Example 1 - Simple VCR

When no special HTTP Transport is required by your `http.Client`, you can use VCR with the default transport:

```go
package main

import (
    "fmt"

    "github.com/seborama/govcr"
)

const example1CassetteName = "MyCassette1"

// Example1 is an example use of govcr.
func Example1() {
    vcr := govcr.NewVCR(example1CassetteName, nil)
    vcr.Client.Get("http://example.com/foo")
    fmt.Printf("%+v\n", vcr.Stats())
}
```

If the **cassette** exists and a **track** matches the request, it will be played back without any real HTTP call to the live server.
Otherwise, a real live HTTP call will be made and recorded in a new track added to the **cassette**.

### Example 2 - Custom VCR Transport

Sometimes, your application will create its own `http.Client` wrapper or will initialise the `http.Client`'s Transport (for instance when using https).
In such cases, you can pass the `http.Client` object of your application to VCR.
VCR will wrap your `http.Client` with its own which you can inject back into your application.

```go
package main

import (
    "crypto/tls"
    "fmt"
    "net/http"
    "time"

    "github.com/seborama/govcr"
)

const example2CassetteName = "MyCassette2"

// myApp is an application container.
type myApp struct {
    httpClient *http.Client
}

func (app myApp) Get(url string) {
    app.httpClient.Get(url)
}

// Example2 is an example use of govcr.
// It shows the use of a VCR with a custom Client.
// Here, the app executes a GET request.
func Example2() {
    // Create a custom http.Transport.
    tr := http.DefaultTransport.(*http.Transport)
    tr.TLSClientConfig = &tls.Config{
        InsecureSkipVerify: true, // just an example, not recommended
    }

    // Create an instance of myApp.
    // It uses the custom Transport created above and a custom Timeout.
    myapp := &myApp{
        httpClient: &http.Client{
            Transport: tr,
            Timeout:   15 * time.Second,
        },
    }

    // Instantiate VCR.
    vcr := govcr.NewVCR(example2CassetteName,
        &govcr.VCRConfig{
            Client: myapp.httpClient,
        })

    // Inject VCR's http.Client wrapper.
    // The original transport has been preserved, only just wrapped into VCR's.
    myapp.httpClient = vcr.Client

    // Run request and display stats.
    myapp.Get("https://example.com/foo")
    fmt.Printf("%+v\n", vcr.Stats())
}
```

### Example 3 - Custom VCR, POST method

Please refer to the source file in the `examples` directory.
This example is identical to Example 2 but with a POST request rather than a GET.

### Example 4 - Custom VCR with a ExcludeHeaderFunc

This example shows how to handle situations where a header in the request needs to be ignored.

For this example, logging is switched on. This is achieved with `Logging: true` in `VCRConfig` when calling `NewVCR`.

```go
package main

import (
    "fmt"
    "strings"
    "time"

    "net/http"

    "github.com/seborama/govcr"
)

const example4CassetteName = "MyCassette4"

// Example4 is an example use of govcr.
// The request contains a custom header 'X-Custom-My-Date' which varies with every request.
// This example shows how to exclude a particular header from the request to facilitate
// matching a previous recording.
// Without the ExcludeHeaderFunc, the headers would not match and hence the playback would not
// happen!
func Example4() {
    vcr := govcr.NewVCR(example4CassetteName,
        &govcr.VCRConfig{
            ExcludeHeaderFunc: func(key string) bool {
                // HTTP headers are case-insensitive
                return strings.ToLower(key) == "x-custom-my-date"
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

### Example 5 - Custom VCR with a ExcludeHeaderFunc and ResponseFilterFunc

This example shows how to handle situations where a transaction Id in the header needs to be present in the response.
This could be as part of a contract validation between server and client.

Note: `RequestFilterFunc` achieves a similar purpose with the **request** Header / Body.
      This is useful when some of the data in the **request** Header / Body needs to be transformed
      before it can be evaluated for comparison for playback.

```go
package main

import (
    "fmt"
    "strings"
    "time"

    "net/http"

    "github.com/seborama/govcr"
)

const example5CassetteName = "MyCassette5"

// Example5 is an example use of govcr.
// Supposing a fictional application where the request contains a custom header
// 'X-Transaction-Id' which must be matched in the response from the server.
// When replaying, the request will have a different Transaction Id than that which was recorded.
// Hence the protocol (of this fictional example) is broken.
// To circumvent that, we inject the new request's X-Transaction-Id into the recorded response.
// Without the ResponseFilterFunc, the X-Transaction-Id in the header would not match that
// of the recorded response and our fictional application would reject the response on validation!
func Example5() {
    vcr := govcr.NewVCR(example5CassetteName,
        &govcr.VCRConfig{
            ExcludeHeaderFunc: func(key string) bool {
                // ignore the X-Transaction-Id since it changes per-request
                return strings.ToLower(key) == "x-transaction-id"
            },
            ResponseFilterFunc: func(respHeader http.Header, respBody string, reqHeader http.Header) (*http.Header, *string) {
                // overwrite X-Transaction-Id in the Response with that from the Request
                respHeader.Set("X-Transaction-Id", reqHeader.Get("X-Transaction-Id"))

                return &respHeader, &respBody
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

### Stats

VCR provides some statistics.

To access the stats, call `vcr.Stats()` where vcr is the `VCR` instance obtained from `NewVCR(...)`.

### Run the examples

Please refer to the `examples` directory for examples of code and uses.

**Observe the output of the examples between the `1st run` and the `2nd run` of each example.**

The first time they run, they perform a live HTTP call (`Executing request to live server`).

However, on second execution (and sub-sequent executions as long as the **cassette** is not deleted)
**govcr** retrieves the previously recorded request and plays it back without live HTTP call (`Found a matching track`). You can disconnect from the internet and still playback HTTP requests endlessly!

#### Make utility

```bash
make examples
```

#### Manually

```bash
cd examples
go run *.go
```

#### Output

First execution - notice the stats show that a **track** was recorded (from a live HTTP call).

Second execution - no **track** is recorded (no live HTTP call) but 1 **track** is loaded and played back.

```bash
Running Example1...
1st run =======================================================
{TracksLoaded:0 TracksRecorded:1 TracksPlayed:0}
2nd run =======================================================
{TracksLoaded:1 TracksRecorded:0 TracksPlayed:1}
Complete ======================================================


Running Example2...
1st run =======================================================
{TracksLoaded:0 TracksRecorded:1 TracksPlayed:0}
2nd run =======================================================
{TracksLoaded:1 TracksRecorded:0 TracksPlayed:1}
Complete ======================================================


Running Example3...
1st run =======================================================
{TracksLoaded:0 TracksRecorded:1 TracksPlayed:0}
2nd run =======================================================
{TracksLoaded:1 TracksRecorded:0 TracksPlayed:1}
Complete ======================================================


Running Example4...
1st run =======================================================
2016/09/12 22:22:20 INFO - Cassette 'MyCassette4' - Executing request to live server for POST http://example.com/foo
2016/09/12 22:22:20 INFO - Cassette 'MyCassette4' - Recording new track for POST http://example.com/foo
{TracksLoaded:0 TracksRecorded:1 TracksPlayed:0}
2nd run =======================================================
2016/09/12 22:22:20 INFO - Cassette 'MyCassette4' - Found a matching track for POST http://example.com/foo
{TracksLoaded:1 TracksRecorded:0 TracksPlayed:1}
Complete ======================================================


Running Example5...
1st run =======================================================
2016/09/12 22:22:20 INFO - Cassette 'MyCassette5' - Executing request to live server for POST http://example.com/foo5
2016/09/12 22:22:20 INFO - Cassette 'MyCassette5' - Recording new track for POST http://example.com/foo5
Header transaction Id verification failed - this would be the live request!
{TracksLoaded:0 TracksRecorded:1 TracksPlayed:0}
2nd run =======================================================
2016/09/12 22:22:20 INFO - Cassette 'MyCassette5' - Found a matching track for POST http://example.com/foo5
Header transaction Id verification passed - this would be the replayed track!
{TracksLoaded:1 TracksRecorded:0 TracksPlayed:1}
Complete ======================================================
```

## Run the tests

```bash
make test
```

or

```bash
go test -race -cover
```

## Bugs

- None known

## Improvements

- When unmarshaling the cassette fails, rather than fail altogether, it would be preferable to revert to live HTTP call.

- The code has a number of TODO's which should either be taken action upon or removed!

## Limitations

### Go empty interfaces (`interface{}`)

Some properties / objects in http.Response are defined as `interface{}`.
This can cause json.Unmarshall to fail (example: when the original type was `big.Int` with a big interger indeed - `json.Unmarshal` attempts to convert to float64 and fails).

Currently, this is dealt with by converting the output of the JSON produced by `json.Marshal` (big.Int is changed to a string).

### Support for multiple values in HTTP headers

Repeat HTTP headers may not be properly handled. A long standing TODO in the code exists but so far no one has complained :-)

### HTTP transport errors

**govcr** also records `http.Client` errors (network down, blocking firewall, timeout, etc) in the **track** for future play back.

Since `errors` is an interface, when it is unmarshalled into JSON, the Go type of the `error` is lost.

To circumvent this, **govcr** serialises the object type (`ErrType`) and the error message (`ErrMsg`) in the **track** record.

Objects cannot be created by name at runtime in Go. Rather than re-create the original error object, *govcr* creates a standard error object with an error string made of both the `ErrType` and `ErrMsg`.

In practice, the implications for you depend on how much you care about the error type. If all you need to know is that an error occurred, you won't mind this limitation.

Mitigation: Support for common errors (network down) has been implemented. Support for more error types can be implemented, if there is appetite for it.
