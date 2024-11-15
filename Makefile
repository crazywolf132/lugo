.PHONY: all build test clean lint coverage

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=lugo

all: lint test build

build:
	$(GOBUILD) -v ./...

test:
	$(GOTEST) -v -race ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out

lint:
	golangci-lint run

coverage:
	$(GOTEST) -v -covermode=count -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

deps:
	$(GOMOD) download
	$(GOMOD) verify

update-deps:
	$(GOMOD) tidy

benchmark:
	$(GOTEST) -bench=. -benchmem ./...

check: lint test coverage

# CI/CD targets
ci: deps lint test coverage

# Install development tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest