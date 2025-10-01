.PHONY: build run clean test install help

BINARY_NAME=probe-lbcdn
BUILD_DIR=build
GO=go
GOFLAGS=-v

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the probe binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Binary created at $(BUILD_DIR)/$(BINARY_NAME)"

run: build ## Build and run the probe
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

clean: ## Remove build artifacts
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

test: ## Run tests
	$(GO) test -v ./...

install: build ## Install the binary to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install .
	@echo "Installation complete"

dev: ## Run in development mode with auto-restart (requires air)
	air

fmt: ## Format Go code
	$(GO) fmt ./...

vet: ## Run go vet
	$(GO) vet ./...

lint: fmt vet ## Run formatters and linters

all: clean lint test build ## Run all checks and build
