# govcr issues

This folder contains test files used to analyse reported issues.

## Issue 27

Execution:
```bash
rm -rf issues/govcr-fixtures/

# will make a live HTTP call
go test -v issues/issue27_test.go

# will replay from the cassette
go test -v issues/issue27_test.go
```

Body payloads are not encrypted but encoded (base 64) as per design by the Go
developers and by http.Client.

