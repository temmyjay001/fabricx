# Dockerfile
# Multi-stage build for FabricX Runtime

# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make protobuf-dev protobuf

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Install protoc-gen-go plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate protobuf files
RUN make proto

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o fabricx-runtime main.go

# Stage 2: Runtime image with Fabric binaries
FROM hyperledger/fabric-tools:2.5

# Install required packages
RUN apt-get update && apt-get install -y \
    docker.io \
    docker-compose \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy the binary from builder
COPY --from=builder /app/fabricx-runtime /usr/local/bin/fabricx-runtime

# Create directory for network data
RUN mkdir -p /var/fabricx

# Expose gRPC port
EXPOSE 50051

# Set working directory
WORKDIR /var/fabricx

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:50051/health || exit 1

# Run the binary
ENTRYPOINT ["/usr/local/bin/fabricx-runtime"]
CMD ["--port=50051"]