.PHONY: build install clean test benchmark coverage

# Binary name
BINARY=blink
# Build directory
BUILD_DIR=build

# Build the binary
build:
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY) ./cmd/blink

# Install the binary
install:
	@echo "Installing $(BINARY)..."
	@go install ./cmd/blink

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with short flag (skips integration tests)
test-short:
	@echo "Running short tests..."
	@go test -v -short ./...

# Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	@go test -v -bench=. -benchmem ./...

# Generate test coverage
coverage:
	@echo "Generating test coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

# Default target
all: build

# Help
help:
	@echo "Available targets:"
	@echo "  build      - Build the binary"
	@echo "  install    - Install the binary"
	@echo "  clean      - Clean build artifacts"
	@echo "  test       - Run tests"
	@echo "  test-short - Run tests without integration tests"
	@echo "  benchmark  - Run benchmarks"
	@echo "  coverage   - Generate test coverage report"
	@echo "  help       - Show this help" 