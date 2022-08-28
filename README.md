# govcr

⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️\
⭐️\
⭐️ 📣 **Community Support Appeal** 📣\
⭐️\
⭐️ Please show your love by **giving a star to this project**.\
⭐️\
⭐️ It takes a lot of **personal time and effort** to maintain and expand features.\
⭐️\
⭐️ If you are using **govcr**, show me it is **worth my continuous effort** by giving it a star.\
⭐️\
⭐️ 🙏 **You'll be my star 😊** 🙏\
⭐️\
⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️
<br/>
<br/>
<p align="center">
  <a href="https://github.com/seborama/govcr/actions/workflows/ci.yml/badge.svg?branch=master">
    <img src="https://github.com/seborama/govcr/actions/workflows/ci.yml/badge.svg?branch=master" alt="govcr">
  </a>

  <a href="https://github.com/seborama/govcr/actions/workflows/codeql-analysis.yml/badge.svg?branch=master">
    <img src="https://github.com/seborama/govcr/actions/workflows/codeql-analysis.yml/badge.svg?branch=master" alt="govcr">
  </a>

  <a href="https://pkg.go.dev/github.com/seborama/govcr/v13">
    <img src="https://img.shields.io/badge/godoc-reference-blue.svg" alt="govcr">
  </a>

  <a href="https://goreportcard.com/report/github.com/seborama/govcr/v13">
    <img src="https://goreportcard.com/badge/github.com/seborama/govcr/v13" alt="govcr">
  </a>
</p>

Records and replays HTTP / HTTPS interactions for offline unit / behavioural / integration tests thereby acting as an HTTP mock. You can also use **goovcr** for API simulation.

This project was inspired by [php-vcr](https://github.com/php-vcr/php-vcr) which is a PHP port of [VCR](https://github.com/vcr/vcr) for ruby.

This project is an adaptation for Google's Go / Golang programming language.

## Table of content

- [Simple VCR example](#simple-vcr-example)
- [Install](#install)
- [Glossary of Terms](#glossary-of-terms)
- [Concepts](#concepts)
  - [VCRSettings](#vcrsettings)
- [Match a request to a cassette track](#match-a-request-to-a-cassette-track)
- [Track mutators](#track-mutators)
- [Cassette encryption](#cassette-encryption)
- [Cookbook](#cookbook)
  - [Run the examples](#run-the-examples)
  - [Recipe: VCR with custom `http.Client`](#recipe-vcr-with-custom-httpclient)
  - [Recipe: Remove Response TLS](#recipe-remove-response-tls)
  - [Recipe: Change the playback mode of the VCR](#recipe-change-the-playback-mode-of-the-vcr)
  - [Recipe: VCR with encrypted cassette](#recipe-vcr-with-encrypted-cassette)
  - [Recipe: VCR with encrypted cassette - custom nonce generator](#recipe-vcr-with-encrypted-cassette---custom-nonce-generator)
  - [Recipe: Cassette decryption](#recipe-cassette-decryption)
  - [Recipe: Changing cassette encryption](#recipe-changing-cassette-encryption)
  - [Recipe: VCR with a custom RequestMatcher](#recipe-vcr-with-a-custom-requestmatcher)
  - [Recipe: VCR with a replaying Track Mutator](#recipe-vcr-with-a-replaying-track-mutator)
  - [Recipe: VCR with a recording Track Mutator](#recipe-vcr-with-a-recording-track-mutator)
  - [More](#more)
- [Stats](#stats)
- [Run the tests](#run-the-tests)
- [Bugs](#bugs)
- [Improvements](#improvements)
- [Limitations](#limitations)
- [Contribute](#contribute)
- [Community Support Appeal](#community-support-appeal)

## Simple VCR example

```go
// See TestExample1 in tests for fully working example.
func TestExample1() {
    vcr := govcr.NewVCR(
        govcr.NewCassetteLoader("MyCassette1.json"),
        govcr.WithRequestMatchers(govcr.NewMethodURLRequestMatchers()...), // use a "relaxed" request matcher
    )

    vcr.Client.Get("http://example.com/foo")
}
```

The **first time** you run this example, `MyCassette1.json` won't exist and `TestExample1` will make a live HTTP call.

On **subsequent executions** (unless you delete the cassette file), the HTTP call will be played back from the cassette and no live HTTP call will occur.

Note:

We use a "relaxed" request matcher because `example.com` injects an "`Age`" header that varies per-request. Without a mutator, **govcr**'s default strict matcher would not match the track on the cassette and keep sending live requests (and record them to the cassette).

[(toc)](#table-of-content)

## Install

```bash
go get github.com/seborama/govcr/v13@latest
```

For all available releases, please check the [releases](https://github.com/seborama/govcr/releases) tab on github.

And your source code would use this import:

```go
import "github.com/seborama/govcr/v13"
```

For versions of **govcr** before v5 (which don't use go.mod), use a dependency manager to lock the version you wish to use (perhaps v4)!

```bash
# download legacy version of govcr (without go.mod)
go get gopkg.in/seborama/govcr.v4
```

[(toc)](#table-of-content)

## Glossary of Terms

**VCR**: Video Cassette Recorder. In this context, a VCR refers to the engine and data that this project provides. A VCR is both an HTTP recorder and player. When you use a VCR, HTTP requests are replayed from previous recordings (**tracks** saved in **cassette** files on the filesystem). When no previous recording exists for the request, it is performed live on the HTTP server, after what it is saved to a **track** on the **cassette**.

**cassette**: a sequential collection of **tracks**. This is in effect a JSON file.

**Long Play cassette**: a cassette compressed in gzip format. Such cassettes have a name that ends with '`.gz`'.

**tracks**: a record of an HTTP request. It contains the request data, the response data, if available, or the error that occurred.

**ControlPanel**: the creation of a VCR instantiates a ControlPanel for interacting with the VCR and conceal its internals.

[(toc)](#table-of-content)

## Concepts

**govcr** is a wrapper around the Go `http.Client`. It can record live HTTP traffic to files (called "**cassettes**") and later replay HTTP requests ("**tracks**") from them instead of live HTTP calls.

The code documentation can be found on [godoc](https://pkg.go.dev/github.com/seborama/govcr/v13).

When using **govcr**'s `http.Client`, the request is matched against the **tracks** on the '**cassette**':

- The **track** is played where a matching one exists on the **cassette**,
- otherwise the request is executed live to the HTTP server and then recorded on **cassette** for the next time.

**Note on a govcr typical flow**

The normal **govcr** flow is test-oriented. Traffic is recorded by default unless a track already existed on the cassette **at the time it was loaded**.

A typical usage:
- run your test once to produce the cassette
- from this point forward, when the test runs again, it will use the cassette

During live recording, the same request can be repeated and recorded many times. Playback occurs in the order the requests were saved on the cassette. See the tests for an example (`TestConcurrencySafety`).

[(toc)](#table-of-content)

### VCRSettings

This structure contains parameters for configuring your **govcr** recorder.

Settings are populated via `With*` options:

- Use `WithClient` to provide a custom http.Client otherwise the default Go http.Client will be used.
- See `vcrsettings.go` for more options such as `WithRequestMatchers`, `WithTrackRecordingMutators`, `WithTrackReplayingMutators`, ...
- TODO: `WithLogging` enables logging to help understand what **govcr** is doing internally.

[(toc)](#table-of-content)

## Match a request to a cassette track

By default, **govcr** uses a strict `RequestMatcher` function that compares the request's headers, method, full URL, body, and trailers.

Another RequestMatcher (obtained with `NewMethodURLRequestMatcher`) provides a more relaxed comparison based on just the method and the full URL.

In some scenarios, it may not possible to match **tracks** exactly as they were recorded.

This may be the case when the request contains a timestamp or a dynamically changing identifier, etc.

You can create your own matcher on any part of the request and in any manner (like ignoring or modifying some headers, etc).

The input parameters received by a `RequestMatcher` are scoped to the `RequestMatchers`. This affects the other `RequestMatcher`'s. But it does **not** permeate throughout the VCR to the original incoming HTTP request or the tracks read from or written to the cassette.

[(toc)](#table-of-content)

## Track mutators

The live HTTP request and response traffic is protected against modifications. While **govcr** could easily support in-place mutation of the live traffic, this is not a goal.

Nonetheless, **govcr** supports mutating tracks, either at **recording time** or at **playback time**.

In either case, this is achieved with track `Mutators`.

A `Mutator` can be combined with one or more `On` conditions. All `On` conditions attached to a mutator must be true for the mutator to apply - in other words,
they are logically "and-ed".

To help construct more complex yet readable predicates easily, **govcr** provides these pre-defined functions for use with `On`:
- `Any` achieves a logical "**or**" of the provided predicates.
- `All` achieves a logical "**and**" of the provided predicates.
- `Not` achieves a logical "**not**" of the provided predicates.
- `None` is synonymous of "`Not` `Any`".

Examples:

```go
myMutator.
    On(Any(...)). // proceeds if any of the "`...`" predicates is true
    On(Not(Any(...)))  // proceeds if none of the "`...`" predicates is true (i.e. all predicates are false)
    On(Not(All(...))).  // proceeds if not every (including none) of the "`...`" predicates is true (i.e. at least one predicate is false, possibly all of them).
```

A **track recording mutator** can change both the request and the response that will be persisted to the cassette.

A **track replaying mutator** transforms the track after it was matched and retrieved from the cassette. It does not change the cassette file.

While a track replaying mutator could change the request, it serves no purpose since the request has already been made and matched to a track by the time the replaying mutator is invoked. The reason for supplying the request in the replaying mutator is for information. In some situations, the request details are needed to transform the response.

The **track replaying mutator** additionally receives an informational copy of the current HTTP request in the track's `Response` under the `Request` field i.e. `Track.Response.Request`. This is useful for tailoring track replays with current request information. See TestExample3 for illustration.

Refer to the tests for examples (search for `WithTrackRecordingMutators` and `WithTrackReplayingMutators`).

[(toc)](#table-of-content)

## Cassette encryption

Your cassettes are likely to contain sensitive information in practice. You can choose to not persist it to the cassette with a recording track mutator. However, in some situations, this information is needed. Enters cassette encryption.

Cassettes can be encrypted with two Go-supported ciphers:
- AES-GCM (12-byte nonce, 16 or 32-byte key)
- ChaCha20Poly1305 (24-byte nonce, 32-byte key)

You will need to provide a secret key to a "`Crypter`" that will take care of encrypting to file and decrypting from file the cassette contents transparently.

The cryptographic "nonce" is stored with the cassette, in its header. The default strategy to generate a n-byte random nonce.

It is possible to provide a custom nonce generator.

Cassettes are expected to be of somewhat reasonable size (at the very most a few MiB). They are fully loaded in memory. Under these circumstances, chunking is not needed and not supported.

As a reminder, you should **never** use a nonce value more than once with the same private key. It would compromise the encryption.

Please refer to the [Cookbook](#cookbook) for decryption and changes to encryption (such as cipher & key rotation).

[(toc)](#table-of-content)

## Cookbook

### Run the examples

Please refer to the `examples` directory for examples of code and uses.

**Observe the output of the examples between the `1st run` and the `2nd run` of each example.**

The **first time** they run, they perform a live HTTP call (`Executing request to live server`).

However, on **second execution** (and subsequent executions as long as the **cassette** is not deleted)
**govcr** retrieves the previously recorded request and plays it back without live HTTP call (`Found a matching track`). You can disconnect from the internet and still playback HTTP requests endlessly!

[(toc)](#table-of-content)

### Recipe: VCR with custom `http.Client`

Sometimes, your application will create its own `http.Client` wrapper (for observation, etc) or will initialise the `http.Client`'s Transport (for instance when using https).

In such cases, you can pass the `http.Client` object of your application to VCR.

VCR will wrap your `http.Client`. You should use `vcr.HTTPClient()` in your tests when making HTTP calls.

```go
// See TestExample2 in tests for fully working example.
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
        govcr.NewCassetteLoader(exampleCassetteName2),
        govcr.WithClient(app.httpClient),
    )

    // Inject VCR's http.Client wrapper.
    // The original transport has been preserved, only just wrapped into VCR's.
    app.httpClient = vcr.HTTPClient()

    // Run request and display stats.
    app.Get("https://example.com/foo")
}
```

[(toc)](#table-of-content)

### Recipe: Remove Response TLS

Use the provided mutator `track.ResponseDeleteTLS`.

Remove Response.TLS from the cassette **recording**:

```go
vcr := govcr.NewVCR(
    govcr.NewCassetteLoader(exampleCassetteName2),
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
    govcr.NewCassetteLoader(exampleCassetteName2),
    govcr.WithTrackReplayingMutators(track.ResponseDeleteTLS()),
    //             ^^^^^^^^^
)
// or, similarly:
vcr.AddReplayingMutators(track.ResponseDeleteTLS())
//     ^^^^^^^^^
```

[(toc)](#table-of-content)

### Recipe: Change the playback mode of the VCR

**govcr** support operation modes:

- **Normal HTTP mode**: replay from the cassette if a track matches otherwise place a live call.
- **Live only**: never replay from the cassette.
- **Offline**: playback from cassette only, return a transport error if no track matches.
- **Read only**: normal behaviour except that recording to cassette is disabled.

#### Normal HTTP mode

```go
vcr := govcr.NewVCR(
    govcr.NewCassetteLoader(exampleCassetteName2),
    // Normal mode is default, no special option required :)
)
// or equally:
vcr.SetNormalMode()
```

#### Live only HTTP mode

```go
vcr := govcr.NewVCR(
    govcr.NewCassetteLoader(exampleCassetteName2),
    govcr.WithLiveOnlyMode(),
)
// or equally:
vcr.SetLiveOnlyMode()
```

#### Read only cassette mode

```go
vcr := govcr.NewVCR(
    govcr.NewCassetteLoader(exampleCassetteName2),
    govcr.WithReadOnlyMode(),
)
// or equally:
vcr.SetReadOnlyMode(true) // `false` to disable option
```

#### Offline HTTP mode

```go
vcr := govcr.NewVCR(
    govcr.NewCassetteLoader(exampleCassetteName2),
    govcr.WithOfflineMode(),
)
// or equally:
vcr.SetOfflineMode()
```

[(toc)](#table-of-content)

### Recipe: VCR with encrypted cassette

At time of creating a new VCR with **govcr**:

```go
// See TestExample4 in tests for fully working example.
vcr := govcr.NewVCR(
    govcr.NewCassetteLoader(exampleCassetteName4).
        WithCipher(
            encryption.NewChaCha20Poly1305WithRandomNonceGenerator,
            "test-fixtures/TestExample4.unsafe.key"),
)
```

[(toc)](#table-of-content)

### Recipe: VCR with encrypted cassette - custom nonce generator

This is nearly identical to the recipe ["VCR with encrypted cassette"](#recipe-vcr-with-encrypted-cassette), except we pass our custom nonce generator.

Example (this can also be achieved in the same way with the `ControlPanel`):

```go
type myNonceGenerator struct{}

func (ng myNonceGenerator) Generate() ([]byte, error) {
    nonce := make([]byte, 12)
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    return nonce, nil
}

vcr := govcr.NewVCR(
    govcr.NewCassetteLoader(exampleCassetteName4).
        WithCipherCustomNonce(
            encryption.NewChaCha20Poly1305,
            "test-fixtures/TestExample4.unsafe.key",
            nonceGenerator),
)
```

[(toc)](#table-of-content)

### Recipe: Cassette decryption

**govcr** provides a CLI utility to decrypt existing cassette files, should we want to.

The command is located in the `cmd/govcr` folder, to install it:

```bash
go install github.com/seborama/govcr/v13/cmd/govcr@latest
```

Example usage:

```bash
govcr decrypt -cassette-file my.cassette.json -key-file my.key
```

`decrypt` will cowardly refuse to write to a file to avoid errors or lingering decrypted files. It will write to the standard output.

[(toc)](#table-of-content)

### Recipe: Changing cassette encryption

The cassette cipher can be changed for another with `SetCipher`.

For safety reasons, you cannot use `SetCipher` to remove encryption and decrypt the cassette. See the [cassette decryption recipe](#recipe-cassette-decryption) for that.

```go
vcr := govcr.NewVCR(...)
err := vcr.SetCipher(
    encryption.NewChaCha20Poly1305WithRandomNonceGenerator,
    "my_secret.key",
)
```

[(toc)](#table-of-content)

### Recipe: VCR with a custom RequestMatcher

This example shows how to handle situations where a header in the request needs to be ignored, in this case header `X-Custom-Timestamp` (or the **track** would not match and hence would not be replayed).

This could be necessary because the header value is not predictable or changes for each request.

```go
vcr.SetRequestMatchers(
    govcr.DefaultMethodMatcher,
    govcr.DefaultURLMatcher,
    func(httpRequest, trackRequest *track.Request) bool {
        // we can safely mutate our inputs:
        // mutations affect other RequestMatcher's but _not_ the
        // original HTTP request or the cassette Tracks.
        httpRequest.Header.Del("X-Custom-Timestamp")
        trackRequest.Header.Del("X-Custom-Timestamp")

        return govcr.DefaultHeaderMatcher(httpRequest, trackRequest)
    },
)
```

[(toc)](#table-of-content)

### Recipe: VCR with a replaying Track Mutator

In this scenario, the API requires a "`X-Transaction-Id`" header to be present. Since the header changes per-request, as needed, replaying a track poses two concerns:
- the request won't match the previously recorded track because the value in "`X-Transaction-Id`" has changed since the track was recorded
- the response track contains the original values of "`X-Transaction-Id`" which is also a mis-match for the new request.

One of different solutions to address both concerns consists in:
- providing a custom request matcher that ignores "`X-Transaction-Id`"
- using the help of a replaying track mutator to inject the correct value for "`X-Transaction-Id`" from the current HTTP request.

How you specifically tackle this in practice really depends on how the API you are using behaves.

```go
// See TestExample3 in tests for fully working example.
vcr := govcr.NewVCR(
    govcr.NewCassetteLoader(exampleCassetteName3),
    govcr.WithRequestMatchers(
        func(httpRequest, trackRequest *track.Request) bool {
            // Remove the header from comparison.
            // Note: this removal is only scoped to the request matcher, it does not affect the original HTTP request
            httpRequest.Header.Del("X-Transaction-Id")
            trackRequest.Header.Del("X-Transaction-Id")

            return govcr.DefaultHeaderMatcher(httpRequest, trackRequest)
        },
    ),
    govcr.WithTrackReplayingMutators(
        // Note: although we deleted the headers in the request matcher, this was limited to the scope of
        // the request matcher. The replaying mutator's scope is past request matching.
        track.ResponseDeleteHeaderKeys("X-Transaction-Id"), // do not append to existing values
        track.ResponseTransferHTTPHeaderKeys("X-Transaction-Id"),
    ),
)
```

[(toc)](#table-of-content)

### Recipe: VCR with a recording Track Mutator

Recording and replaying track mutators are the same. The only difference is when the mutator is invoked.

To set recording mutators, use `govcr.WithTrackRecordingMutators` when creating a new `VCR`, or use the `SetRecordingMutators` or `AddRecordingMutators` methods of the `ControlPanel` that is returned by `NewVCR`.

See the recipe ["VCR with a replaying Track Mutator"](#recipe-vcr-with-a-replaying-track-mutator) for the general approach on creating a track mutator. You can also take a look at the recipe ["Remove Response TLS"](#recipe-remove-response-tls).

[(toc)](#table-of-content)

### More

**TODO: add example that includes the use of `.On*` predicates**

[(toc)](#table-of-content)

## Stats

VCR provides some statistics.

To access the stats, call `vcr.Stats()` where vcr is the `ControlPanel` instance obtained from `NewVCR(...)`.

[(toc)](#table-of-content)

## Run the tests

```bash
make test
```

[(toc)](#table-of-content)

## Bugs

- The recording of TLS data for PublicKey's is not reliable owing to a limitation in Go's json package and a non-deterministic and opaque use of a blank interface in Go's certificate structures. Some improvements are possible with `gob`.

[(toc)](#table-of-content)

## Improvements

- When unmarshaling the cassette fails, rather than fail altogether, it may be preferable to revert to live HTTP call.

- The code has a number of TODO's which should either be taken action upon or removed!

[(toc)](#table-of-content)

## Limitations

### Go empty interfaces (`interface{}` / `any`)

Some properties / objects in http.Response are defined as `interface{}` (or `any`).

This can cause `json.Unmarshal` to fail (example: when the original type was `big.Int` with a big integer indeed - `json.Unmarshal` attempts to convert to float64 and fails).

Currently, this is dealt with by removing known untyped fields from the tracks. This is the case for PublicKey in certificate chains of the TLS data structure.

[(toc)](#table-of-content)

### Support for multiple values in HTTP headers

Repeat HTTP headers may not be properly handled. A long standing TODO in the code exists but so far no one has complained :-)

[(toc)](#table-of-content)

### HTTP transport errors

**govcr** also records `http.Client` errors (network down, blocking firewall, timeout, etc) in the **track** for future playback.

Since `errors` is an interface, when it is unmarshalled into JSON, the Go type of the `error` is lost.

To circumvent this, **govcr** serialises the object type (`ErrType`) and the error message (`ErrMsg`) in the **track** record.

Objects cannot be created by name at runtime in Go. Rather than re-create the original error object, *govcr* creates a standard error object with an error string made of both the `ErrType` and `ErrMsg`.

In practice, the implications for you depend on how much you care about the error type. If all you need to know is that an error occurred, you won't mind this limitation.

Mitigation: Support for common errors (network down) has been implemented. Support for more error types can be implemented, if there is appetite for it.

[(toc)](#table-of-content)

## Contribute

You are welcome to submit a PR to contribute.

Please try and follow a TDD workflow: tests must be present and as much as is practical to you, avoid toxic DDT (development driven testing).

[(toc)](#table-of-content)

## Community Support Appeal

⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️\
⭐️\
⭐️ 📣 **Community Support Appeal** 📣\
⭐️\
⭐️ Please show your love by **giving a star to this project**.\
⭐️\
⭐️ It takes a lot of **personal time and effort** to maintain and expand features.\
⭐️\
⭐️ If you are using **govcr**, show me it is **worth my continuous effort** by giving it a star.\
⭐️\
⭐️ 🙏 **You'll be my star 😊** 🙏\
⭐️\
⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️⭐️

[(top)](#govcr)
