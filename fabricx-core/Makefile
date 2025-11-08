# Makefile
.PHONY: all build proto test clean run install deps client

# Variables
BINARY_NAME=fabricx-runtime
CLIENT_NAME=fabricx-client
PROTO_DIR=protos
GO_FILES=$(shell find . -name '*.go' -type f)
PROTO_FILES=$(shell find $(PROTO_DIR) -name '*.proto' -type f)

all: deps proto build client

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Generate protobuf files
proto:
	@echo "Generating protobuf files..."
	@mkdir -p pkg/grpcserver
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_FILES)
	@mv protos/*.go pkg/grpcserver/

# Build the binary
build: proto
	@echo "Building $(BINARY_NAME)..."
	go build -o bin/$(BINARY_NAME) main.go

# Build the client
client: proto
	@echo "Building $(CLIENT_NAME)..."
	go build -o bin/$(CLIENT_NAME) cmd/client/main.go

# Build for multiple platforms
build-all: proto
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_NAME)-linux-amd64 main.go
	GOOS=darwin GOARCH=amd64 go build -o bin/$(BINARY_NAME)-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -o bin/$(BINARY_NAME)-darwin-arm64 main.go
	GOOS=windows GOARCH=amd64 go build -o bin/$(BINARY_NAME)-windows-amd64.exe main.go
	@echo "Building client for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build -o bin/$(CLIENT_NAME)-linux-amd64 cmd/client/main.go
	GOOS=darwin GOARCH=amd64 go build -o bin/$(CLIENT_NAME)-darwin-amd64 cmd/client/main.go
	GOOS=darwin GOARCH=arm64 go build -o bin/$(CLIENT_NAME)-darwin-arm64 cmd/client/main.go
	GOOS=windows GOARCH=amd64 go build -o bin/$(CLIENT_NAME)-windows-amd64.exe cmd/client/main.go

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run the binary
run: build
	@echo "Starting FabricX Runtime..."
	./bin/$(BINARY_NAME)

# Install binary to $GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	go install

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -rf /tmp/fabricx/

# Development mode with auto-reload
dev:
	@echo "Starting development mode..."
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	air

# Format code
fmt:
	@echo "Formatting code..."
	gofmt -s -w .

# Lint code
lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

# Check for security issues
security:
	@echo "Running security checks..."
	@which gosec > /dev/null || go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec ./...

# Generate mock files for testing
mocks:
	@echo "Generating mocks..."
	@which mockgen > /dev/null || go install github.com/golang/mock/mockgen@latest
	go generate ./...

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t fabricx-runtime:latest .

# Docker run
docker-run:
	@echo "Running Docker container..."
	docker run -p 50051:50051 -v /var/run/docker.sock:/var/run/docker.sock fabricx-runtime:latest

# Help
help:
	@echo "FabricX Runtime - Available targets:"
	@echo "  make deps          - Install Go dependencies"
	@echo "  make proto         - Generate protobuf files"
	@echo "  make build         - Build the binary"
	@echo "  make build-all     - Build for all platforms"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make run           - Build and run"
	@echo "  make install       - Install to GOPATH"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make dev           - Start development mode"
	@echo "  make fmt           - Format code"
	@echo "  make lint          - Lint code"
	@echo "  make security      - Run security checks"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-run    - Run Docker container"