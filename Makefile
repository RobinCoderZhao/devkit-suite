.PHONY: all build-newsbot build-devkit build-watchbot test lint clean

GO=go
GOFLAGS=-trimpath -ldflags="-s -w"

all: build-newsbot build-devkit build-watchbot

build-newsbot:
	$(GO) build $(GOFLAGS) -o bin/newsbot ./cmd/newsbot

build-devkit:
	$(GO) build $(GOFLAGS) -o bin/devkit ./cmd/devkit

build-watchbot:
	$(GO) build $(GOFLAGS) -o bin/watchbot ./cmd/watchbot

test:
	$(GO) test ./... -v -count=1

test-pkg:
	$(GO) test ./pkg/... -v -count=1

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

tidy:
	$(GO) mod tidy
