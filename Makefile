.PHONY: build test lint run

build:
	go build -o gh-inbox .

test:
	go test ./...

lint:
	golangci-lint run

run: build
	./gh-inbox
