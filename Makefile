MACHINE := $(shell uname -m)
ifneq ($(MACHINE),aarch64)
	GORACE := -race
endif

deps:
	go mod tidy && go mod download

test: deps
    # to run a single test inside a stretchr suite (e.g.):
    # go test -v ./... -run ^TestHandlerTestSuite$ -testify.m TestRoundTrip_ReplaysResponse
	go test -timeout 20s -cover ./...

	# note: -race significantly degrades performance hence a high "timeout" value and reduced parallelism
	go test -timeout 60s -cover $(GORACE) -parallel 2 ./...

lint: deps
	./golangci-lint.sh || :

mutate: deps
	./gomutesting.sh || :

