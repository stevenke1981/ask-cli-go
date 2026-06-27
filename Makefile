.PHONY: all build test fmt vet lint clean run help

APP_NAME := ask
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")
COMMIT_SHA := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/ask-cli/ask-cli/internal/cli.Version=$(VERSION) -X github.com/ask-cli/ask-cli/internal/cli.CommitSHA=$(COMMIT_SHA)"

GO := go
GOFLAGS := -trimpath

# Detect OS
UNAME_S := $(shell uname -s 2>/dev/null || echo Windows)
ifeq ($(UNAME_S),Windows)
	BINARY_EXT := .exe
endif

all: fmt vet build test

build:
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o dist/$(APP_NAME)$(BINARY_EXT) ./cmd/ask

build-all:
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o dist/ask-windows-amd64.exe ./cmd/ask
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o dist/ask-linux-amd64 ./cmd/ask
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o dist/ask-darwin-arm64 ./cmd/ask

test:
	$(GO) test ./... -v -count=1

test-race:
	$(GO) test ./... -race -count=1

test-short:
	$(GO) test ./... -short -count=1

test-coverage:
	$(GO) test ./... -coverprofile=coverage.out -count=1
	$(GO) tool cover -func=coverage.out

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint:
ifeq (, $(shell which golangci-lint))
	$(warning "golangci-lint not installed, skipping")
else
	golangci-lint run ./...
endif

clean:
	rm -rf dist/
	rm -f go.work

run:
	$(GO) run $(GOFLAGS) ./cmd/ask $(ARGS)

help:
	$(GO) run $(GOFLAGS) ./cmd/ask --help

login:
	$(GO) run $(GOFLAGS) ./cmd/ask login
