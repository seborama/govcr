# govcr

Record and replay HTTP interactions for offline unit / behavioural / integration tests thereby acting as an HTTP mock.

This project was inspired by [php-vcr](https://github.com/php-vcr/php-vcr) which is a PHP port of [VCR](https://github.com/vcr/vcr) for ruby.

This project is an adaptation for Google's Go / Golang programming language.

## Install

```bash
go get github.com/seborama/govcr
```

You can pick a specific major release for compatibility. For example, to use a v1.x release, use this command:

```bash
go get gopkg.in/seborama/govcr.v1
```

And your source code would use this import:

```go
import "gopkg.in/seborama/govcr.v1"
```

## Glossary of Terms

**VCR**: Video Cassette Recorder. In this context, a VCR refers to the overall engine and data that this project provides. A VCR is both an HTTP recorder and player. When you use a VCR, HTTP requests are replayed from previous recordings (**tracks** saved in **cassette** files on the filesystem). When no previous recording exists for the request, it is performed live on the HTTP server the request is intended to, after what it is saved to a **track** on the **cassette**.

**cassette**: a sequential collection of **tracks**. This is in effect a JSON file saved under directory `./govcr-fixtures`. The **cassette** is given a name when creating the **VCR** which becomes the filename (with an extension of `.cassette`).

**tracks**: a record of an HTTP request. It contains the request data, the response data, if available, or the error that occurred.

**PCB**: Printed Circuit Board. This is an analogy that refers to the ability to supply customisations to certain aspects of the behaviour of the **VCR** (for instance, disable recordings or ignore certain HTTP headers in the request when looking for a previously recorded **track**).

## Documentation

**govcr** is a wrapper around the Go `http.Client` which offers the ability to run pre-recorded HTTP requests ('**tracks**') instead of live HTTP calls.

**govcr** can replay both successful and failed HTTP transactions.

A given request may be repeated, again and again. They will be replayed in the same order as they were recorded. See the tests for an example (`TestPlaybackOrder`).

The code documentation can be found on [godoc](http://godoc.org/github.com/seborama/govcr).

When using **govcr**'s `http.Client`, the request is matched against the **tracks** on the '**cassette**':

- The **track** is played where a matching one exists on the **cassette**,
- or the request is executed live to the HTTP server and then recorded on **cassette** for the next time.

**Cassette** recordings are saved under `./govcr-fixtures` as `*.cassette` files in JSON format.

## Features

- Record extensive details about the request, response or error (network error, timeout, etc) to provide as accurate a playback as possible compared to the live HTTP request.

- Recordings are JSON files and can be read in an editor.

- Custom Go `http.Client`'s can be supplied.

- Custom Go `http.Transport` / `http.RoundTrippers`.

- http / https supported and any other protocol implemented by the supplied `http.Client`'s `http.RoundTripper`.

- Hook to define HTTP headers that should be excluded from the HTTP request when attemtping to retrieve a **track** for playback.
  This is useful to deal with non-static HTTP headers (for example, containing a timestamp).

- Hook to parse the Body of an HTTP request to deal with non-static data. The purpose is similar to the hook for headers described above.

- Ability to switch off automatic recordings.
  This allows to play back existing records or make
  a live HTTP call without recording it to the **cassette**.

- Record SSL certificates.

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

Note: `RequestBodyFilterFunc` achieves a similar purpose with the Body of the request.
      This is useful when some of the data in the Body needs to be transformed before it
      can be evaluated for comparison matching for playback.

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
// The request contains a customer header 'X-Custom-My-Date' which varies with every request.
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

### Stats

VCR provides some statistics.
The
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
2016/07/17 00:08:01 INFO - Cassette 'MyCassette4' - Executing request to live server for POST http://example.com/foo
2016/07/17 00:08:02 INFO - Cassette 'MyCassette4' - Recording new track for POST http://example.com/foo
{TracksLoaded:0 TracksRecorded:1 TracksPlayed:0}
2nd run =======================================================
2016/07/17 00:08:02 INFO - Cassette 'MyCassette4' - Found a matching track for POST http://example.com/foo
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

## Limitations

### Go empty interfaces (`interface{}`)

Some properties / objects in http.Response are defined as `interface{}`.
This can cause json.Unmarshall to fail (example: when the original type was `big.Int` with a big interger indeed - `json.Unmarshal` attempts to convert to float64 and fails).

Currently, this is dealt with by converting the output of the JSON produced by `json.Marshal` (big.Int is changed to a string).

### HTTP transport errors

**govcr** also records `http.Client` errors (network down, blocking firewall, timeout, etc) in the **track** for future play back.

Since `errors` is an interface, when it is unmarshalled into JSON, the Go type of the `error` is lost.

To circumvent this, **govcr** serialises the object type (`ErrType`) and the error message (`ErrMsg`) in the **track** record.

Objects cannot be created by name at runtime in Go. Rather than re-create the original error object, *govcr* creates a standard error object with an error string made of both the `ErrType` and `ErrMsg`.

In practice, the implications for you depends on how much you care about the error type. If all you need to know is that an error occurred, you won't mind this limitation.

Mitigation: Support for common errors (network down) has been implemented. More error types can be implemented, if there is appetite for it.
