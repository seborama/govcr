# govcr

Record and replay HTTP interactions for offline unit / behavioural / integration tests thereby acting as an HTTP mock.

This project was inspired by [php-vcr](https://github.com/php-vcr/php-vcr) which is a PHP port of [VCR](https://github.com/vcr/vcr) for ruby.

This project is an adaptation for Google's Go / Golang programming language.

## Install

```bash
go get github.com/seborama/govcr
```

## Glossary of Terms

**VCR**: Video Cassette Recorder. In this context, a VCR refers to the overall engine and data that this project provides. A VCR is both an HTTP recorder and player. When you use a VCR, HTTP requests are replayed from previous recordings (**tracks saved in **cassette** files on the filesystem). When no previous recording exists for the request, it is performed live on the HTTP server the request is intended to, after what it is saved to a **track** on the **cassette**.

**cassette**: a sequential collection of **tracks**. This is in effect a JSON file saved under directory `./govcr-fixtures`. The **cassette** is given a name when creating the **VCR** which becomes the filename (with an extension of `.cassette`).

**tracks**: a record of an HTTP request. It contains the request data, the response data, if available, or the error that occurred.

**PCB**: Printed Circuit Board. This is an analogy that refers to the ability to supply customisations to certain aspects of the behaviour of the **VCR** (for instance, disable recordings).

## Documentation

**govcr** is a wrapper around the Go `http.Client` which offers the ability to run pre-recorded HTTP requests ('**tracks**') instead of live HTTP calls.

The code documentation can be found on [godoc](http://godoc.org/github.com/seborama/govcr).

When using **govcr**'s `http.Client`, the request is matched against the **tracks** on the '**cassette**':

- The **track** is played where a matching one exists on the **cassette**,
- or the request is executed live to the HTTP server and then recorded on **cassette** for the next time.

**Cassette** recordings are saved under `./govcr-fixtures` as `*.cassette` files in JSON format.

## Features

- Record extensive details about the request, response or error to provide as accurate a playback as possible compared to the live HTTP request.

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

### Simple VCR

When no special HTTP Transport is required by your `http.Client`, you can use VCR with the default transport:

```go
package main

import "github.com/seborama/govcr"

// Example1 is an example use of govcr.
func Example1() {
    vcr := govcr.NewVCR("MyCassette1", nil)
    vcr.Client.Get("http://example.com/foo")
}
```

If the **cassette** exists and a **track** matches the request, it will be played back without any real HTTP call to the live server.
Otherwise, a real live HTTP call will be made and recorded in a new track added to the **cassette**.

### Custom VCR Transport

Sometimes, your application will create its own `http.Client` wrapper or will initialise the `http.Client`'s Transport (for instance when using https).
In such cases, you can pass the Transport object of your application's `http.Client` instance to VCR.
VCR will wrap your `http.Client` with its own which you can inject back into your application.

```go
package main

import (
    "crypto/tls"
    "net/http"
    "time"

    "github.com/seborama/govcr"
)

// myApp is an application container.
type myApp struct {
    httpClient *http.Client
}

func (app myApp) Get(url string) {
    app.httpClient.Get(url)
}

// Example2 is an example use of govcr.
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
    vcr := govcr.NewVCR("MyCassette2", myapp.httpClient.Transport)

    // Inject VCR's http.Client wrapper.
    // The original transport has been preserved, only just wrapped into VCR's.
    myapp.httpClient = vcr.Client

    myapp.Get("https://example.com/foo")
}
```

### Stats

VCR provides some statistics.

To access the stats, call `vcr.Stats()` where vcr is the `VCR` instance obtained from `NewVCR(...)`.

### Run the examples

```bash
cd examples

# clear the fixtures
rm govcr-fixtures/*.cassette

# the first time, live calls are made to the HTTP server
go run *.go

# the second time, VCR plays back the tracks from the cassette
# observe the info messages displayed in the output
go run *.go
```

#### Output

First execution - notice the 'No track found' INFO messages for both **cassettes**:

```bash
Running Example1...
=======================================================
2016/07/13 10:42:03 stat ./govcr-fixtures/MyCassette1.cassette: no such file or directory
2016/07/13 10:42:03 WARNING - loadCassette - No cassette. Creating a blank one
2016/07/13 10:42:03 INFO - Cassette 'MyCassette1' - No track found for 'GET' 'http://example.com/foo' in the tracks that remain at this stage ([]govcr.track(nil)). Recording a new track from live server

Running Example2...
=======================================================
2016/07/13 10:42:03 stat ./govcr-fixtures/MyCassette2.cassette: no such file or directory
2016/07/13 10:42:03 WARNING - loadCassette - No cassette. Creating a blank one
2016/07/13 10:42:03 INFO - Cassette 'MyCassette2' - No track found for 'GET' 'https://example.com/foo' in the tracks that remain at this stage ([]govcr.track(nil)). Recording a new track from live server
```

Second execution (when **cassettes** exist with applicable **tracks**) - notice the 'Replaying roundtrip from track' INFO messages for both **cassettes** - no more live HTTP call 😊 :

```bash
Running Example1...
=======================================================
2016/07/13 10:44:09 INFO - Cassette 'MyCassette1' - Replaying roundtrip from track 'GET' 'http://example.com/foo'

Running Example2...
=======================================================
2016/07/13 14:26:30 INFO - Cassette 'MyCassette2' - Replaying roundtrip from track 'GET' 'https://example.com/foo'
```

## Run the tests

```bash
go test -race -cover
```

## Bugs

- NewVCR does not copy all attributes of the `http.Client` that is supplied to it as an argument (for instance, Timeout, Jar, etc).

## Improvements

- When unmarshaling the cassette fails, rather than fail altogether, it would be preferable to revert to live HTTP call.

## Limitations

### Go empty interfaces (`interface{}`)

Some properties / objects in http.Response are defined as `interface{}`.
This can cause json.Unmarshall to fail (example: when the original type was `big.Int` with a big interger indeed - `json.Unmarshal` attempts to convert to float64 and fails).

Currently, this is dealt with by converting the output of the JSON produced by `json.Marshal` (big.Int is changed to a string).

### HTTP errors

**govcr** also records `http.Client` errors (network down, blocking firewall, timeout, etc) in the **cassette** for future play back.
As `errors` is an interface, when it is unmarshalled into JSON, the Go type of the `error` is lost.
To circumvent this, **govcr** serialises the object type (`ErrType`) and the error message (`ErrMsg`) in the **track** record.

As objects cannot be created by name at runtime in Go, rather than re-create the original error object, *govcr* creates a standard error object with an error string made of both the `ErrType` and `ErrMsg`.

In practice, the implication depends on how much you care about the error type. If all you need to know is that an error occurred, you won't mind this limitation. However, if you need to know exactly what error happened, you will find this annoying.
In a future release, support for common error (network down) could be implemented, if there is appetite for it.
