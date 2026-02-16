# Readwise TUI Makefile

BINARY_NAME=readwise-triage
CMD_PATH=./cmd/readwise-triage

.PHONY: all build test run clean install setup

all: build

build:
	go build -o $(BINARY_NAME) $(CMD_PATH)

test:
	go test ./... -v

test-coverage:
	go test ./... -cover

run:
	go run $(CMD_PATH)

clean:
	rm -f $(BINARY_NAME)

install: build
	cp $(BINARY_NAME) $(GOPATH)/bin/

fmt:
	go fmt ./...

vet:
	go vet ./...

mod-tidy:
	go mod tidy

setup:
	git config core.hooksPath .githooks
	@echo "Git hooks configured."
