.PHONY: all build test clean install run smoke-test validate-configs

# Variables
BINARY_NAME=umcp
GO=go
GOFLAGS=-v
LDFLAGS=
CONFIGS_DIR=configs

# Default target
all: clean build test

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) main.go
	@echo "Build complete: ./$(BINARY_NAME)"

# Run unit tests
test:
	@echo "Running unit tests..."
	$(GO) test $(GOFLAGS) ./...
	@echo "Unit tests complete"

# Run smoke tests
smoke-test: build
	@echo "Running smoke tests..."
	@./test/smoke_test.sh
	@echo "Smoke tests complete"

# Validate all config files
validate-configs: build
	@echo "Validating configuration files..."
	@for config in $(CONFIGS_DIR)/*.yaml; do \
		echo "  Validating $$config..."; \
		./$(BINARY_NAME) --config $$config --validate || exit 1; \
	done
	@echo "All configs are valid"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf coverage.out
	@echo "Clean complete"

# Install binary to ~/bin
install: build
	@echo "Installing $(BINARY_NAME) to ~/bin..."
	@cp $(BINARY_NAME) ~/bin/
	@echo "Installation complete"

# Install via Homebrew formula (local testing)
brew-install: build
	@echo "Installing via Homebrew formula..."
	@brew install --formula ./umcp.rb
	@echo "Homebrew installation complete"

# Test release process locally
test-release:
	@echo "Testing release process..."
	@git tag -d v$(VERSION) 2>/dev/null || true
	@git tag v$(VERSION)
	@echo "Created local tag v$(VERSION)"
	@echo "To push: git push origin v$(VERSION)"

# Run the application with a sample config
run: build
	@echo "Running $(BINARY_NAME) with ls config..."
	./$(BINARY_NAME) --config $(CONFIGS_DIR)/ls.yaml --test

# Generate Claude Desktop configuration
claude-config: build
	@echo "Generating Claude Desktop configuration..."
	@./$(BINARY_NAME) --config $(CONFIGS_DIR)/git.yaml --generate-claude-config

# Development mode - watch and rebuild
dev:
	@echo "Running in development mode..."
	@while true; do \
		make build test; \
		echo "Watching for changes... (Ctrl+C to stop)"; \
		fswatch -1 -r --exclude '.git' --exclude '$(BINARY_NAME)' .; \
	done

# Get dependencies
deps:
	@echo "Getting dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Dependencies installed"

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Formatting complete"

# Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run
	@echo "Linting complete"

# Run security scan
security:
	@echo "Running security scan..."
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	gosec ./...
	@echo "Security scan complete"

# Show help
help:
	@echo "Universal MCP Bridge - Makefile targets:"
	@echo ""
	@echo "  make build          - Build the binary"
	@echo "  make test           - Run unit tests"
	@echo "  make smoke-test     - Run smoke tests"
	@echo "  make validate-configs - Validate all config files"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make install        - Install binary to /usr/local/bin"
	@echo "  make run            - Run with sample config"
	@echo "  make claude-config  - Generate Claude Desktop config"
	@echo "  make dev            - Development mode with auto-rebuild"
	@echo "  make deps           - Get Go dependencies"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Run linter"
	@echo "  make security       - Run security scan"
	@echo "  make help           - Show this help message"
	@echo ""
	@echo "Default target (make): clean, build, test"