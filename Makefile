MACHINE := $(shell uname -m)
ifneq ($(MACHINE),aarch64)
	GORACE := -race
endif

deps:
	go mod tidy && go mod download

test: deps
    # to run a single test inside a stretchr suite (e.g.):
    # go test -run ^TestGoVCRTestSuite$ -testify.m TestRoundTrip_ReplaysResponse -v ./...
	go test -timeout 20s -cover ./... -coverprofile coverage.out -coverpkg ./...
	go tool cover -func coverage.out -o coverage.out
	cat coverage.out

	# note: -race significantly degrades performance hence a high "timeout" value and reduced parallelism
	go test -timeout 120s $(GORACE) -parallel 2 ./...

lint: deps
	./golangci-lint.sh || :
