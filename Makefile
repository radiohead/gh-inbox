.PHONY: build test lint run

BIN_DIR := .bin
LINTER_VERSION := 2.10.1
LINTER_BINARY := $(BIN_DIR)/golangci-lint-$(LINTER_VERSION)

build:
	go build -o gh-inbox .

test:
	go test ./...

lint: $(LINTER_BINARY)
	$(LINTER_BINARY) run

run: build
	./gh-inbox

$(LINTER_BINARY):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BIN_DIR) v$(LINTER_VERSION)
	@mv $(BIN_DIR)/golangci-lint $@
