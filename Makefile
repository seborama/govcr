MACHINE := $(shell uname -m)
ifneq ($(MACHINE),aarch64)
	GORACE := -race
endif

deps:
	go mod tidy && go mod download

cover:
	go test -timeout 20s -count 1 -cover ./... -coverprofile coverage.out -coverpkg ./...
	go tool cover -func coverage.out -o coverage.out

	@echo -e "\n"
	@cat coverage.out

test: deps cover
    # to run a single test inside a stretchr suite (e.g.):
    # go test -run ^TestGoVCRTestSuite$ -testify.m TestRoundTrip_ReplaysResponse -v ./...

	# note: -race significantly degrades performance hence a high "timeout" value and reduced parallelism
	go test -timeout 120s $(GORACE) -count 1 -parallel 2 ./...

lint: deps
	./golangci-lint.sh || :

bin: deps
	go build -o bin/govcr ./cmd/govcr/...

