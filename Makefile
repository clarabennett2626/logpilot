.PHONY: build test lint clean

BINARY=logpilot
VERSION?=dev

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) ./cmd/logpilot/

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)

install:
	go install ./cmd/logpilot/
