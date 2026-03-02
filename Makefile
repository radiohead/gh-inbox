.PHONY: build test lint run install help

.DEFAULT_GOAL := help

BIN_DIR := .bin
LINTER_VERSION := 2.10.1
LINTER_BINARY := $(BIN_DIR)/golangci-lint-$(LINTER_VERSION)

CYAN  := \033[36m
BOLD  := \033[1m
RESET := \033[0m

help: ## Show this help
	@printf "$(BOLD)gh-inbox - Available Targets$(RESET)\n\n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-12s$(RESET) %s\n", $$1, $$2}'
	@printf "\n$(BOLD)Common workflows:$(RESET)\n"
	@printf "  Install extension:   make install\n"
	@printf "  Run tests:           make test\n"
	@printf "  Build only:          make build\n\n"

build: ## Build the gh-inbox binary
	go build -o gh-inbox .

test: ## Run all tests
	go test ./...

lint: $(LINTER_BINARY) ## Run golangci-lint
	$(LINTER_BINARY) run

run: build ## Build and run the binary
	./gh-inbox

EXT_DIR := $(HOME)/.local/gh-inbox

install: build ## Install (or reinstall) as a gh extension
	@gh extension remove inbox 2>/dev/null || true
	@mkdir -p $(EXT_DIR)
	@ln -sf $(CURDIR)/gh-inbox $(EXT_DIR)/gh-inbox
	@cd $(EXT_DIR) && gh extension install .

$(LINTER_BINARY):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BIN_DIR) v$(LINTER_VERSION)
	@mv $(BIN_DIR)/golangci-lint $@
