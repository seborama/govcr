# govcr

Record and replay HTTP interactions for offline unit / behavioural / integration tests.

This project was inspired by [php-vcr](https://github.com/php-vcr/php-vcr) which is a PHP port of [VCR](https://github.com/vcr/vcr) for ruby.

This project is an adaptation for Google's Go / Golang programming language.

## Install

```bash
go get github.com/seborama/govcr
```

## Documentation

**govcr** is a wrapper around the Go `http.Client` which offers the ability to run pre-recorded HTTP requests ('**tracks**') instead of live HTTP calls.

When using **govcr**'s `http.Client`, the request is matched against the **tracks** on the '**cassette**':

- The **track** is played where a matching one exists on the **cassette**,
- or the request is executed live to the HTTP server and then recorded on **cassette** for the next time.

**Cassette** recordings are saved under `./govcr-fixtures` as `*.cassette` files in JSON format.

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

TODO.

## Run the tests

```bash
go test -race -cover`
```

## Bugs

- Fields of type `interface{}` are not unmarshaled correctly. This can be observed with `x509.Certificate`'s `PublicKey` property.
- NewVCR does not copy all attributes of the `http.Client` that is supplied to it as an argument (for instance, Timeout, Jar, etc).

## Improvements

- When unmarshaling the cassette fails, rather than fail altogether, it would be preferable to revert to live HTTP call.

## Limitations

### HTTP errors

**govcr** also records `http.Client` errors (network down, blocking firewall, timeout, etc) in the **cassette** for future play back.
As `errors` is an interface, when it is unmarshalled into JSON, the Go type of the `error` is lost.
To circumvent this, **govcr** serialises the object type (`ErrType`) and the error message (`ErrMsg`) in the **track** record.

As objects cannot be created by name at runtime in Go, rather than re-create the original error object, *govcr* creates a standard error object with an error string made of both the `ErrType` and `ErrMsg`.

In practice, the implication depends on how much you care about the error type. If all you need to know is that an error occurred, you won't mind this limitation. However, if you need to know exactly what error happened, you will find this annoying.
In a future release, support for common error (network down) could be implemented, if there is appetite for it.
