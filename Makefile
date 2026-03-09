.PHONY: all build clean install test test-unit test-e2e test-all test-deps help release release-dry-run fmt fmt-check vet lint verify

BINARY_NAME=rosactl
BUILD_DIR=./bin
INSTALL_DIR=/usr/local/bin
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
# Get version from latest git tag, fallback to 0.1.0 if no tags exist
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.1.0")
LDFLAGS=-ldflags "-X github.com/openshift-online/rosa-regional-platform-cli/internal/commands/version.GitCommit=$(GIT_COMMIT) -X github.com/openshift-online/rosa-regional-platform-cli/internal/commands/version.Version=$(VERSION)"

all: fmt vet lint test build
	@echo ""
	@echo "✓ All checks and build completed successfully!"

help:
	@echo "Available targets:"
	@echo "  all             - Run all checks (fmt, vet, lint, test, build)"
	@echo "  build           - Build the rosactl binary"
	@echo "  clean           - Remove built binaries"
	@echo "  install         - Install rosactl to $(INSTALL_DIR)"
	@echo "  test            - Run unit tests (default, no AWS required)"
	@echo "  test-e2e        - Run e2e tests (requires AWS_PROFILE)"
	@echo "  test-deps       - Install test dependencies"
	@echo "  tidy            - Tidy go modules"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt             - Format Go code with gofmt"
	@echo "  fmt-check       - Check if Go code is formatted (non-destructive)"
	@echo "  vet             - Run go vet for suspicious code"
	@echo "  lint            - Run golangci-lint"
	@echo "  verify          - Run all checks (fmt-check, vet, lint)"
	@echo ""
	@echo "Version Management (Semantic Versioning):"
	@echo "  release-dry-run - Show what next version would be (dry-run)"
	@echo "  release         - Create semantic version release (uses conventional commits)"
	@echo ""
	@echo "  help            - Show this help message"

build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/rosactl
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

clean:
	@echo "Cleaning up..."
	@rm -f $(BUILD_DIR)/$(BINARY_NAME)
	@echo "Clean complete"

install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installation complete"

test:
	@echo "Running unit tests..."
	@go test -v -race -coverprofile=coverage.out ./internal/... ./cmd/...
	@echo "✓ Unit tests passed"

test-e2e: build
	@echo "Running e2e tests..."
	@if [ -z "$$AWS_PROFILE" ]; then \
		echo "Error: AWS_PROFILE environment variable must be set"; \
		echo "Example: export AWS_PROFILE=your-profile-name"; \
		exit 1; \
	fi
	@E2E_BINARY_PATH=$$(pwd)/$(BUILD_DIR)/$(BINARY_NAME) \
		ginkgo -v --trace --timeout=15m ./test/e2e
	@echo "✓ E2E tests passed"

test-deps:
	@echo "Installing test dependencies..."
	@go get github.com/onsi/ginkgo/v2
	@go get github.com/onsi/gomega
	@go install github.com/onsi/ginkgo/v2/ginkgo@latest
	@echo "Test dependencies installed"

tidy:
	@echo "Tidying go modules..."
	@go mod tidy
	@echo "Tidy complete"

# Code Quality Targets
fmt:
	@echo "Formatting Go code..."
	@gofmt -w -s ./cmd ./internal ./test
	@echo "Formatting complete"

fmt-check:
	@echo "Checking code formatting..."
	@files=$$(gofmt -l ./cmd ./internal ./test 2>&1); \
	if [ -n "$$files" ]; then \
		echo "The following files are not formatted:"; \
		echo "$$files"; \
		echo ""; \
		echo "Run 'make fmt' to format them"; \
		exit 1; \
	fi
	@echo "✓ All files are properly formatted"

vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ go vet passed"

lint:
	@echo "Running golangci-lint..."
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "Error: golangci-lint not found"; \
		echo "Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		echo "Or visit: https://golangci-lint.run/welcome/install/"; \
		exit 1; \
	fi
	@golangci-lint run ./...
	@echo "✓ golangci-lint passed"

verify: fmt-check vet lint
	@echo ""
	@echo "✓ All verification checks passed!"

# Semantic versioning using go-semver-release
# Requires: go install github.com/s0ders/go-semver-release@latest
release-dry-run:
	@echo "Checking what next version would be..."
	@if ! command -v go-semver-release &> /dev/null; then \
		echo "Error: go-semver-release not found"; \
		echo "Install: go install github.com/s0ders/go-semver-release@latest"; \
		exit 1; \
	fi
	@go-semver-release release . --dry-run --config .semver.yaml

release:
	@echo "Creating semantic version release..."
	@if ! command -v go-semver-release &> /dev/null; then \
		echo "Error: go-semver-release not found"; \
		echo "Install: go install github.com/s0ders/go-semver-release@latest"; \
		exit 1; \
	fi
	@echo "Analyzing commits using conventional commit messages..."
	@go-semver-release release . --config .semver.yaml
	@echo ""
	@echo "✓ Release created! New version:"
	@git describe --tags --abbrev=0
	@echo ""
	@echo "To push the tag to remote, run:"
	@echo "  git push origin $$(git describe --tags --abbrev=0)"
	@echo ""
	@echo "⚠️  Don't forget to update the version badge in README.md:"
	@echo "  ![Version](https://img.shields.io/badge/version-$$(git describe --tags --abbrev=0 | sed 's/^v//')-blue.svg)"
