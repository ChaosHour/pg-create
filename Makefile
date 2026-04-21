.PHONY: build build-validate build-pgstress build-pgwatch build-all clean install test help run

# Variables
BINARY_NAME=pg-create
VALIDATE_BINARY_NAME=pg-validate
PGSTRESS_BINARY_NAME=pgstress
BUILD_DIR=./bin
CMD_DIR=./cmd/pgcreate
VALIDATE_CMD_DIR=./cmd/pgvalidate
PGSTRESS_CMD_DIR=./cmd/pgstress
PGWATCH_CMD_DIR=./cmd/pgwatch
PGWATCH_BINARY_NAME=pg-watch
GO=go
GOFLAGS=-v

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "✓ Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

# Build validation CLI
build-validate:
	@echo "Building $(VALIDATE_BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(VALIDATE_BINARY_NAME) $(VALIDATE_CMD_DIR)
	@echo "✓ Binary built: $(BUILD_DIR)/$(VALIDATE_BINARY_NAME)"

# Build pgstress CLI
build-pgstress:
	@echo "Building $(PGSTRESS_BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(PGSTRESS_BINARY_NAME) $(PGSTRESS_CMD_DIR)
	@echo "✓ Binary built: $(BUILD_DIR)/$(PGSTRESS_BINARY_NAME)"

# Build pg-watch CLI
build-pgwatch:
	@echo "Building $(PGWATCH_BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(PGWATCH_BINARY_NAME) $(PGWATCH_CMD_DIR)
	@echo "✓ Binary built: $(BUILD_DIR)/$(PGWATCH_BINARY_NAME)"

# Build all CLIs
build-all: build build-validate build-pgstress build-pgwatch
	@echo "✓ Built $(BINARY_NAME), $(VALIDATE_BINARY_NAME), $(PGSTRESS_BINARY_NAME), and $(PGWATCH_BINARY_NAME)"

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
	@echo "  build-validate - Build the validator binary to ./bin/$(VALIDATE_BINARY_NAME)"
	@echo "  build-pgstress - Build the pgstress binary to ./bin/$(PGSTRESS_BINARY_NAME)"
	@echo "  build-pgwatch - Build the pg-watch binary to ./bin/$(PGWATCH_BINARY_NAME)"
	@echo "  build-all - Build all CLIs"
	@echo "  clean    - Remove build artifacts"
	@echo "  deps     - Download and tidy dependencies"
	@echo "  test     - Run tests"
	@echo "  install  - Install binary to GOPATH/bin"
	@echo "  run      - Build and run the application"
	@echo "  help     - Show this help message"

# Default target
.DEFAULT_GOAL := build
