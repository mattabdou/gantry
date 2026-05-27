# GANTRY Makefile
# Cross-platform build targets for Windows, macOS, and Linux

BINARY_NAME=gantry
VERSION=1.2.1-beta.5
BUILD_DIR=build
LDFLAGS=-ldflags "-s -w"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build targets
.PHONY: all build clean test fmt vet deps help
.PHONY: build-linux build-linux-arm64 build-darwin build-darwin-arm64 build-windows
.PHONY: build-all install

# Default target
all: clean deps test build

# Build for current platform
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

# Install locally
install: build
	cp $(BINARY_NAME) $(GOPATH)/bin/

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	$(GOFMT) ./...

# Vet code
vet:
	$(GOVET) ./...

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# =============================================================================
# Cross-compilation targets
# =============================================================================

# Linux AMD64
build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .

# Linux ARM64
build-linux-arm64:
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .

# macOS AMD64 (Intel)
build-darwin:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .

# macOS ARM64 (Apple Silicon)
build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .

# Windows AMD64
build-windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

# Windows ARM64
build-windows-arm64:
	GOOS=windows GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe .

# Build all platforms
build-all: clean deps
	@mkdir -p $(BUILD_DIR)
	@echo "Building for Linux AMD64..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	@echo "Building for Linux ARM64..."
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	@echo "Building for macOS AMD64..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	@echo "Building for macOS ARM64..."
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	@echo "Building for Windows AMD64..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Building for Windows ARM64..."
	GOOS=windows GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe .
	@echo "Build complete! Binaries are in $(BUILD_DIR)/"
	@ls -la $(BUILD_DIR)/

# Create release archives
release: build-all
	@echo "Creating release archives..."
	@cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	@cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	@cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	@cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	@cd $(BUILD_DIR) && zip $(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	@cd $(BUILD_DIR) && zip $(BINARY_NAME)-$(VERSION)-windows-arm64.zip $(BINARY_NAME)-windows-arm64.exe
	@echo "Release archives created!"
	@ls -la $(BUILD_DIR)/*.tar.gz $(BUILD_DIR)/*.zip

# Help
help:
	@echo "GANTRY Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make              - Clean, download deps, test, and build"
	@echo "  make build        - Build for current platform"
	@echo "  make install      - Build and install to GOPATH/bin"
	@echo "  make test         - Run tests"
	@echo "  make test-coverage- Run tests with coverage report"
	@echo "  make fmt          - Format code"
	@echo "  make vet          - Vet code"
	@echo "  make deps         - Download dependencies"
	@echo "  make clean        - Remove build artifacts"
	@echo ""
	@echo "Cross-compilation:"
	@echo "  make build-linux        - Build for Linux AMD64"
	@echo "  make build-linux-arm64  - Build for Linux ARM64"
	@echo "  make build-darwin       - Build for macOS AMD64 (Intel)"
	@echo "  make build-darwin-arm64 - Build for macOS ARM64 (Apple Silicon)"
	@echo "  make build-windows      - Build for Windows AMD64"
	@echo "  make build-windows-arm64- Build for Windows ARM64"
	@echo "  make build-all          - Build for all platforms"
	@echo "  make release            - Build all and create release archives"
