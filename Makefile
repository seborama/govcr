MACHINE := $(shell uname -m)
ifneq ($(MACHINE),aarch64)
	GORACE := -race
endif

deps:
	go get -d -t -v ./...

test: deps
    # to run a single test inside a stretchr suite (e.g.):
    # go test -v ./... -run ^TestHandlerTestSuite$ -testify.m TestRoundTrip_ReplaysResponse
	go test -timeout 15s -cover $(GORACE) -parallel 100 ./...

examples: deps
	go run ./examples/*.go

lint: deps
	./golangci-lint.sh || :

mutate: deps
	./gomutesting.sh || :

