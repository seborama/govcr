MACHINE := $(shell uname -m)
ifneq ($(MACHINE),aarch64)
	GORACE := -race
endif

deps:
	go get -d -t -v ./...

test: deps
	go test -timeout 15s -cover $(GORACE) -parallel 100 ./...

examples: deps
	go run ./examples/*.go

lint: deps

	@echo "=~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~="
	./golangci-lint.sh ||:

	@echo "=~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~="
	./gomutesting.sh ||:

