# GANTRY Makefile
# Cross-platform build targets for Windows, macOS, and Linux

BINARY_NAME=gantry
VERSION=1.2.1-beta.11
BUILD_DIR=build
LDFLAGS=-ldflags "-s -w"

# macOS code signing / notarization (override on the command line if needed)
# SIGN_IDENTITY is matched by name so it keeps working when the certificate is
# renewed and its hash changes.
SIGN_IDENTITY=Developer ID Application: Matthew Abdou (EDV43PJULJ)
NOTARY_PROFILE=gantry-notarize

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
.PHONY: build-all install archives release release-signed
.PHONY: sign-darwin notarize-darwin verify-darwin

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

# Create release archives from whatever is currently in $(BUILD_DIR).
# Kept separate from build-all so it can run *after* the macOS binaries are
# signed - see release-signed.
archives:
	@echo "Creating release archives..."
	@cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	@cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	@cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	@cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	@cd $(BUILD_DIR) && zip $(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	@cd $(BUILD_DIR) && zip $(BINARY_NAME)-$(VERSION)-windows-arm64.zip $(BINARY_NAME)-windows-arm64.exe
	@echo "Release archives created!"
	@ls -la $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-*

# Create release archives with UNSIGNED macOS binaries.
# For a real macOS release use release-signed instead.
release: build-all
	@$(MAKE) archives
	@echo ""
	@echo "NOTE: the macOS binaries in these archives are NOT signed or notarized."
	@echo "      Use 'make release-signed' on macOS to produce a distributable release."

# =============================================================================
# macOS code signing and notarization
# =============================================================================

# Sign the macOS binaries with a Developer ID certificate.
# --options runtime enables the hardened runtime and --timestamp adds a secure
# timestamp; notarization rejects binaries without both.
sign-darwin:
	@if [ "$$(uname)" != "Darwin" ]; then \
		echo "Error: sign-darwin requires macOS (codesign is unavailable here)."; \
		exit 1; \
	fi
	@if ! security find-identity -v -p codesigning | grep -q "$(SIGN_IDENTITY)"; then \
		echo "Error: signing identity not found in the keychain:"; \
		echo "  $(SIGN_IDENTITY)"; \
		echo ""; \
		echo "Available identities:"; \
		security find-identity -v -p codesigning; \
		echo ""; \
		echo "Override with: make sign-darwin SIGN_IDENTITY=\"...\""; \
		exit 1; \
	fi
	@for arch in amd64 arm64; do \
		bin="$(BUILD_DIR)/$(BINARY_NAME)-darwin-$$arch"; \
		if [ ! -f "$$bin" ]; then \
			echo "Error: $$bin not found. Run 'make build-all' first."; \
			exit 1; \
		fi; \
		echo "Signing $$bin..."; \
		codesign --force --options runtime --timestamp \
			--sign "$(SIGN_IDENTITY)" "$$bin" || exit 1; \
	done
	@$(MAKE) verify-darwin
	@echo "macOS binaries signed."

# Submit the signed macOS binaries to Apple's notary service and wait.
#
# Credentials come from a notarytool keychain profile. Create one once with:
#   xcrun notarytool store-credentials "$(NOTARY_PROFILE)" \
#     --apple-id "<apple-id>" --team-id "<team-id>" --password "<app-specific-password>"
#
# Note: the resulting ticket is deliberately NOT stapled. A notarization ticket
# can only be stapled into a bundle/dmg/pkg, not a bare Mach-O binary
# ('xcrun stapler staple' fails with Error 73). Gatekeeper looks the ticket up
# online instead, so a notarized CLI binary needs network access on first run.
notarize-darwin:
	@if [ "$$(uname)" != "Darwin" ]; then \
		echo "Error: notarize-darwin requires macOS."; \
		exit 1; \
	fi
	@if ! xcrun notarytool history --keychain-profile "$(NOTARY_PROFILE)" >/dev/null 2>&1; then \
		echo "Error: notarytool keychain profile \"$(NOTARY_PROFILE)\" is missing or invalid."; \
		echo ""; \
		echo "Create it with:"; \
		echo "  xcrun notarytool store-credentials \"$(NOTARY_PROFILE)\" \\"; \
		echo "    --apple-id \"<apple-id>\" --team-id \"<team-id>\" --password \"<app-specific-password>\""; \
		exit 1; \
	fi
	@for arch in amd64 arm64; do \
		bin="$(BINARY_NAME)-darwin-$$arch"; \
		echo "Submitting $$bin for notarization..."; \
		( cd $(BUILD_DIR) && \
		  ditto -c -k --keepParent "$$bin" "$$bin-notarize.zip" && \
		  xcrun notarytool submit "$$bin-notarize.zip" \
			--keychain-profile "$(NOTARY_PROFILE)" --wait ) || exit 1; \
	done
	@echo ""
	@echo "Notarization complete. Verifying with Gatekeeper..."
	@echo "(a freshly issued ticket can take a minute or two to propagate;"
	@echo " 'Unnotarized Developer ID' immediately after acceptance is usually lag)"
	@for arch in amd64 arm64; do \
		spctl -a -vvv -t install "$(BUILD_DIR)/$(BINARY_NAME)-darwin-$$arch" 2>&1 | head -3; \
	done

# Verify the macOS signatures without contacting Apple
verify-darwin:
	@for arch in amd64 arm64; do \
		bin="$(BUILD_DIR)/$(BINARY_NAME)-darwin-$$arch"; \
		echo "--- $$bin ---"; \
		codesign --verify --strict --verbose=2 "$$bin" 2>&1 | tail -2; \
		codesign -dvvv "$$bin" 2>&1 | grep -E "^(Authority=Developer|TeamIdentifier|CDHash)"; \
	done

# Full distributable release: build, sign, notarize, then archive.
# Ordering matters - the archives must be created from the signed binaries, so
# the steps are invoked sequentially rather than as prerequisites.
release-signed:
	@if [ "$$(uname)" != "Darwin" ]; then \
		echo "Error: release-signed requires macOS to sign and notarize."; \
		exit 1; \
	fi
	$(MAKE) build-all
	$(MAKE) sign-darwin
	$(MAKE) notarize-darwin
	$(MAKE) archives
	@echo ""
	@echo "Signed release $(VERSION) ready in $(BUILD_DIR)/"
	@echo "Upload the 6 versioned archives AND the 6 raw binaries"
	@echo "(the raw binaries are required by 'gantry update')."

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
	@echo "  make archives           - Archive whatever is already in build/"
	@echo "  make release            - Build all and archive (macOS binaries UNSIGNED)"
	@echo ""
	@echo "macOS signing (requires macOS):"
	@echo "  make sign-darwin        - Codesign the macOS binaries with a Developer ID cert"
	@echo "  make verify-darwin      - Verify the macOS signatures locally"
	@echo "  make notarize-darwin    - Submit the signed macOS binaries to Apple and wait"
	@echo "  make release-signed     - build-all + sign + notarize + archive (use this to ship)"
	@echo ""
	@echo "  Override the identity or notary profile if needed:"
	@echo "    make release-signed SIGN_IDENTITY=\"Developer ID Application: ...\" NOTARY_PROFILE=my-profile"
