# Readwise TUI Makefile

BINARY_NAME=readwise-tui
CMD_PATH=./cmd/readwise-tui

.PHONY: all build test run clean install

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
