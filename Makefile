.PHONY: build run clean test fmt vet install

# Build variables
BINARY_NAME=mcp-server
GO=go

# Default target
all: build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@$(GO) build -o $(BINARY_NAME) .
	@echo "Build complete: ./$(BINARY_NAME)"

# Run the application
run: build
	@echo "Starting MCP server..."
	@./$(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@$(GO) clean
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	@$(GO) test -v ./...

# Format code
fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	@$(GO) vet ./...

# Install dependencies
install:
	@echo "Installing dependencies..."
	@$(GO) mod download
	@$(GO) mod tidy
	@echo "Dependencies installed"

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@GOOS=linux GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-linux-amd64 .
	@GOOS=darwin GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 $(GO) build -o $(BINARY_NAME)-darwin-arm64 .
	@GOOS=windows GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-windows-amd64.exe .
	@echo "Multi-platform build complete"

# Show help
help:
	@echo "Available targets:"
	@echo "  build      - Build the application (default)"
	@echo "  run        - Build and run the application"
	@echo "  clean      - Remove build artifacts"
	@echo "  test       - Run tests"
	@echo "  fmt        - Format code"
	@echo "  vet        - Run go vet"
	@echo "  install    - Install dependencies"
	@echo "  build-all  - Build for multiple platforms"
	@echo "  help       - Show this help message"
