.PHONY: build clean install test help

# Variables
BINARY_NAME=pg-create
BUILD_DIR=./bin
CMD_DIR=./cmd/pgcreate
GO=go
GOFLAGS=-v

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "✓ Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "✓ Clean complete"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "✓ Dependencies installed"

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Install binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "✓ Installed to $(GOPATH)/bin/$(BINARY_NAME)"

# Run the application
run: build
	@$(BUILD_DIR)/$(BINARY_NAME)

# Show help
help:
	@echo "Available targets:"
	@echo "  build    - Build the binary to ./bin/$(BINARY_NAME)"
	@echo "  clean    - Remove build artifacts"
	@echo "  deps     - Download and tidy dependencies"
	@echo "  test     - Run tests"
	@echo "  install  - Install binary to GOPATH/bin"
	@echo "  run      - Build and run the application"
	@echo "  help     - Show this help message"

# Default target
.DEFAULT_GOAL := build
