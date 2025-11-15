# Makefile
.PHONY: all build proto test test-unit test-integration test-coverage clean run install deps client lint security docker-build help

# Variables
BINARY_NAME=fabricx-runtime
CLIENT_NAME=fabricx-client
PROTO_DIR=protos
GO_FILES=$(shell find . -name '*.go' -type f -not -path "./vendor/*")
PROTO_FILES=$(shell find $(PROTO_DIR) -name '*.proto' -type f)
COVERAGE_DIR=coverage

VERSION ?= 0.1.0
GIT_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.gitCommit=$(GIT_COMMIT) -X main.buildDate=$(BUILD_DATE)"

all: deps proto build client

# Install dependencies
deps:
	@echo "📦 Installing dependencies..."
	go mod download
	go mod tidy

# Generate protobuf files
proto:
	@echo "🔨 Generating protobuf files..."
	@mkdir -p pkg/grpcserver
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_FILES)
	@mv protos/*.go pkg/grpcserver/

# Build the binary
build: proto
	@echo "🏗️  Building $(BINARY_NAME) v$(VERSION)..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) main.go

# Build the client
client: proto
	@echo "🏗️  Building $(CLIENT_NAME)..."
	go build -o bin/$(CLIENT_NAME) cmd/client/main.go

# Build for all platforms with compression
build-release: proto
	@echo "🌍 Building release binaries for v$(VERSION)..."
	@mkdir -p releases/$(VERSION)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 main.go
	tar -czf releases/$(VERSION)/fabricx-runtime-$(VERSION)-linux-amd64.tar.gz -C bin $(BINARY_NAME)-linux-amd64
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 main.go
	tar -czf releases/$(VERSION)/fabricx-runtime-$(VERSION)-linux-arm64.tar.gz -C bin $(BINARY_NAME)-linux-arm64
	
	# macOS AMD64 (Intel)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 main.go
	tar -czf releases/$(VERSION)/fabricx-runtime-$(VERSION)-darwin-amd64.tar.gz -C bin $(BINARY_NAME)-darwin-amd64
	
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 main.go
	tar -czf releases/$(VERSION)/fabricx-runtime-$(VERSION)-darwin-arm64.tar.gz -C bin $(BINARY_NAME)-darwin-arm64
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe main.go
	cd bin && zip ../releases/$(VERSION)/fabricx-runtime-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	
	@echo "✅ Release binaries created in releases/$(VERSION)/"
	@ls -lh releases/$(VERSION)/

# Generate checksums for release
checksums:
	@echo "🔐 Generating checksums..."
	cd releases/$(VERSION) && shasum -a 256 * > SHA256SUMS
	@echo "✅ Checksums generated"

# Complete release build
release: deps proto build-release checksums
	@echo "🎉 Release v$(VERSION) ready!"
	@echo "📦 Files:"
	@ls -lh releases/$(VERSION)/

# Run all tests
test: test-unit

# Run unit tests
test-unit:
	@echo "🧪 Running unit tests..."
	go test -v -race ./pkg/...

# Run unit tests with short flag (skip slow tests)
test-short:
	@echo "⚡ Running short unit tests..."
	go test -v -short -race ./pkg/...

# Run integration tests
test-integration:
	@echo "🔗 Running integration tests..."
	go test -v -tags=integration -race ./tests/integration/...

# Run tests with coverage
test-coverage:
	@echo "📊 Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	go test -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./pkg/...
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "📈 Coverage report: $(COVERAGE_DIR)/coverage.html"

# Run tests and generate coverage report
test-coverage-full:
	@echo "📊 Running full test coverage..."
	@mkdir -p $(COVERAGE_DIR)
	go test -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	go tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -n 1
	@echo "📈 Coverage report: $(COVERAGE_DIR)/coverage.html"

# Run specific package tests
test-pkg:
	@echo "🎯 Testing specific package..."
	@read -p "Package name (e.g., ./pkg/docker): " pkg; \
	go test -v -race $$pkg

# Run tests with verbose output
test-verbose:
	@echo "📢 Running tests with verbose output..."
	go test -v -race -count=1 ./pkg/...

# Run benchmarks
bench:
	@echo "⏱️  Running benchmarks..."
	go test -bench=. -benchmem ./pkg/...

# Run benchmarks for specific package
bench-pkg:
	@read -p "Package name (e.g., ./pkg/docker): " pkg; \
	go test -bench=. -benchmem $$pkg

# Run the binary
run: build
	@echo "▶️  Starting FabricX Runtime..."
	./bin/$(BINARY_NAME)

# Install binary to $GOPATH/bin
install: build
	@echo "📦 Installing $(BINARY_NAME)..."
	go install

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	@if command -v docker-compose > /dev/null; then \
		docker-compose -f config/docker-compose-fabric.yaml down -v --remove-orphans; \
	fi
	rm -rf bin/
	rm -rf $(COVERAGE_DIR)/
	rm -f coverage.out coverage.html
	rm -rf /tmp/fabricx/
	go clean -cache -testcache

# Development mode with auto-reload
dev:
	@echo "🔄 Starting development mode..."
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	air

# Format code
fmt:
	@echo "✨ Formatting code..."
	gofmt -s -w .
	@which goimports > /dev/null || go install golang.org/x/tools/cmd/goimports@latest
	goimports -w $(GO_FILES)

# Lint code
lint:
	@echo "🔍 Linting code..."
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run --timeout=5m

# Lint with autofix
lint-fix:
	@echo "🔧 Linting code with autofix..."
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run --fix --timeout=5m

# Check for security issues
security:
	@echo "🔒 Running security checks..."
	@which gosec > /dev/null || go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -exclude=G204,G304 ./...

# Run static analysis
vet:
	@echo "🔬 Running go vet..."
	go vet ./...

# Generate mock files for testing
mocks:
	@echo "🎭 Generating mocks..."
	@which mockgen > /dev/null || go install github.com/golang/mock/mockgen@latest
	go generate ./...

# Check test coverage threshold
check-coverage:
	@echo "📊 Checking coverage threshold..."
	@go test -coverprofile=$(COVERAGE_DIR)/coverage.out ./pkg/... > /dev/null
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out | \
		awk '/^total:/ {print "Total coverage: " $$3; if ($$3+0 < 70) {print "❌ Coverage below 70%"; exit 1} else {print "✅ Coverage above 70%"}}'

# Run all quality checks
quality: fmt vet lint test-coverage check-coverage
	@echo "✅ All quality checks passed!"

# Docker build
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t fabricx-runtime:latest .

# Docker run
docker-run:
	@echo "🐳 Running Docker container..."
	docker run -p 50051:50051 -v /var/run/docker.sock:/var/run/docker.sock fabricx-runtime:latest

# CI pipeline simulation
ci: deps proto lint vet test-coverage check-coverage
	@echo "✅ CI pipeline completed successfully!"

# Pre-commit checks
pre-commit: fmt lint test-unit
	@echo "✅ Pre-commit checks passed!"

# Generate test report
test-report:
	@echo "📄 Generating test report..."
	@mkdir -p $(COVERAGE_DIR)
	go test -v -json ./pkg/... > $(COVERAGE_DIR)/test-report.json
	@echo "📄 Test report: $(COVERAGE_DIR)/test-report.json"

# View coverage in browser
coverage-html: test-coverage
	@echo "🌐 Opening coverage report in browser..."
	@which open > /dev/null && open $(COVERAGE_DIR)/coverage.html || \
	 which xdg-open > /dev/null && xdg-open $(COVERAGE_DIR)/coverage.html || \
	 echo "Please open $(COVERAGE_DIR)/coverage.html manually"

# List all test functions
list-tests:
	@echo "📝 Listing all tests..."
	@grep -r "^func Test" pkg/ | sed 's/.*func //' | sed 's/(t.*//' | sort

# Run specific test by name
test-name:
	@read -p "Test name pattern: " pattern; \
	go test -v -race -run=$$pattern ./pkg/...

# Help
help:
	@echo "FabricX Runtime - Available targets:"
	@echo ""
	@echo "📦 Build & Dependencies:"
	@echo "  make deps           - Install Go dependencies"
	@echo "  make proto          - Generate protobuf files"
	@echo "  make build          - Build the runtime binary"
	@echo "  make client         - Build the client binary"
	@echo "  make build-all      - Build for all platforms"
	@echo ""
	@echo "🧪 Testing:"
	@echo "  make test           - Run all tests"
	@echo "  make test-unit      - Run unit tests only"
	@echo "  make test-short     - Run short tests"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-coverage  - Run tests with coverage"
	@echo "  make test-verbose   - Run tests with verbose output"
	@echo "  make test-pkg       - Test specific package"
	@echo "  make test-name      - Run tests matching pattern"
	@echo "  make bench          - Run benchmarks"
	@echo "  make bench-pkg      - Benchmark specific package"
	@echo ""
	@echo "📊 Coverage & Reports:"
	@echo "  make coverage-html  - Generate and open HTML coverage report"
	@echo "  make check-coverage - Verify coverage threshold (70%)"
	@echo "  make test-report    - Generate JSON test report"
	@echo ""
	@echo "✨ Code Quality:"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Lint code"
	@echo "  make lint-fix       - Lint with autofix"
	@echo "  make vet            - Run go vet"
	@echo "  make security       - Run security checks"
	@echo "  make quality        - Run all quality checks"
	@echo "  make pre-commit     - Run pre-commit checks"
	@echo ""
	@echo "🔄 Development:"
	@echo "  make run            - Build and run"
	@echo "  make dev            - Start dev mode with auto-reload"
	@echo "  make install        - Install to GOPATH"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make mocks          - Generate mock files"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-run     - Run Docker container"
	@echo ""
	@echo "🚀 CI/CD:"
	@echo "  make ci             - Simulate full CI pipeline"
	@echo ""
	@echo "💡 Other:"
	@echo "  make list-tests     - List all test functions"
	@echo "  make help           - Show this help message"