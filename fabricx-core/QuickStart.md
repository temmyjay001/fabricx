# FabricX Go Runtime - Quick Start Guide

## 🚀 Prerequisites

**Only 3 things needed:**

1. **Go 1.23+**

   ```bash
   go version  # Should show go1.23 or higher
   ```

2. **Docker**

   ```bash
   docker --version  # Should be running
   ```

3. **Docker Compose**

   ```bash
   docker-compose --version
   ```

**That's it!** No Fabric binaries needed.

## 📦 Project Structure

```
core/
├── main.go                          # Entry point
├── go.mod                           # Go dependencies
├── go.sum                           # Dependency checksums
├── Makefile                         # Build automation
├── setup.sh                         # Automated setup script
├── protos/
│   └── fabricx.proto               # gRPC service definition
├── pkg/
│   ├── grpcserver/
│   │   ├── server.go               # gRPC server implementation
│   │   ├── fabricx_grpc.pb.go     # Generated gRPC stubs
│   │   └── fabricx.pb.go          # Generated message types
│   ├── network/
│   │   ├── network.go              # Network bootstrapping
│   │   ├── crypto.go               # Crypto generation (dockerized)
│   │   └── docker-compose.go      # Docker Compose generation
│   ├── chaincode/
│   │   ├── deployer.go             # Chaincode deployment
│   │   └── invoker.go              # Transaction invocation
│   ├── docker/
│   │   └── manager.go              # Docker orchestration
│   └── utils/
│       └── file.go                 # File utilities
└── bin/
    └── fabricx-runtime             # Compiled binary
```

## 🎯 Method 1: Automated Setup (Recommended)

```bash
# Clone the repository
git clone https://github.com/your-org/core
cd core

# Run the setup script
chmod +x setup.sh
./setup.sh
```

This will:

- ✅ Check prerequisites
- ✅ Install Go dependencies
- ✅ Generate protobuf files (if protoc available)
- ✅ Pull Fabric Docker images
- ✅ Build the binary
- ✅ Run tests (optional)

## 🔧 Method 2: Manual Setup

### Step 1: Install Dependencies

```bash
go mod download
go mod tidy
```

### Step 2: Generate Protobuf Files (Optional)

If you have `protoc` installed:

```bash
# Install protoc-gen-go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate from proto
make proto
# OR manually:
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  api/fabricx.proto
```

**Note:** If you don't have protoc, the manual implementations in `pkg/grpcserver/` will work.

### Step 3: Build the Binary

```bash
make build
# OR manually:
go build -o bin/fabricx-runtime main.go
```

### Step 4: Pull Fabric Images (First Time Only)

```bash
docker pull hyperledger/fabric-peer:2.5
docker pull hyperledger/fabric-orderer:2.5
docker pull hyperledger/fabric-ca:1.5
docker pull hyperledger/fabric-tools:2.5
docker pull couchdb:3.3
```

## 🚀 Running the Runtime

```bash
# Start the gRPC server
./bin/fabricx-runtime

# With custom port
./bin/fabricx-runtime --port=50052

# Check version
./bin/fabricx-runtime --version
```

**Output:**

```
🚀 FabricX Runtime v0.1.0 starting on port 50051
📦 All Fabric operations will run in Docker containers
✅ No local Fabric binaries required!
```

## 🧪 Testing the Runtime

### Option 1: Using grpcurl

```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List available services
grpcurl -plaintext localhost:50051 list

# Initialize a test network
grpcurl -plaintext -d '{
  "network_name": "test-network",
  "num_orgs": 2,
  "channel_name": "mychannel"
}' localhost:50051 fabricx.FabricXService/InitNetwork

# Check network status
grpcurl -plaintext -d '{
  "network_id": "NETWORK_ID_FROM_ABOVE"
}' localhost:50051 fabricx.FabricXService/GetNetworkStatus

# Stop network
grpcurl -plaintext -d '{
  "network_id": "NETWORK_ID",
  "cleanup": true
}' localhost:50051 fabricx.FabricXService/StopNetwork
```

### Option 2: Using Go Client

Create a test file `test_client.go`:

```go
package main

import (
    "context"
    "log"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    pb "core/pkg/grpcserver"
)

func main() {
    conn, err := grpc.Dial("localhost:50051", 
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewFabricXServiceClient(conn)

    // Initialize network
    resp, err := client.InitNetwork(context.Background(), &pb.InitNetworkRequest{
        NetworkName: "test-network",
        NumOrgs:     2,
        ChannelName: "mychannel",
    })
    if err != nil {
        log.Fatalf("InitNetwork failed: %v", err)
    }

    log.Printf("Network initialized: %s", resp.NetworkId)
    log.Printf("Endpoints: %v", resp.Endpoints)
}
```

Run it:

```bash
go run test_client.go
```

## 🐛 Troubleshooting

### Issue: "Docker not available"

```bash
# Check if Docker is running
docker ps

# If not running, start Docker Desktop (Mac/Windows)
# Or start Docker daemon (Linux)
sudo systemctl start docker
```

### Issue: "Permission denied" for Docker socket

```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Log out and back in, or:
newgrp docker
```

### Issue: Build fails with "undefined: RegisterFabricXServiceServer"

This means the protobuf files aren't generated. You have two options:

1. **Install protoc and regenerate:**

   ```bash
   # macOS
   brew install protobuf
   
   # Linux
   apt-get install -y protobuf-compiler
   
   # Then regenerate
   make proto
   ```

2. **Use the manual implementations:**
   The files `pkg/grpcserver/fabricx_grpc.pb.go` and `pkg/grpcserver/fabricx.pb.go` are provided with manual implementations.

### Issue: Port 50051 already in use

```bash
# Find what's using the port
lsof -i :50051

# Kill it or use a different port
./bin/fabricx-runtime --port=50052
```

### Issue: "Failed to pull Fabric images"

```bash
# Pull images manually
docker pull hyperledger/fabric-peer:2.5
docker pull hyperledger/fabric-orderer:2.5

# Or check Docker Hub status
# Images might be temporarily unavailable
```

## 📊 Verifying Everything Works

Run this complete test:

```bash
# 1. Start the runtime (in one terminal)
./bin/fabricx-runtime

# 2. In another terminal, test initialization
grpcurl -plaintext -d '{
  "network_name": "test",
  "num_orgs": 2,
  "channel_name": "testchannel"
}' localhost:50051 fabricx.FabricXService/InitNetwork

# 3. Check Docker containers
docker ps

# You should see:
# - orderer.example.com
# - peer0.org1.example.com
# - peer0.org2.example.com
# - ca.org1.example.com
# - ca.org2.example.com
# - couchdb containers
# - cli container

# 4. Check logs
docker logs peer0.org1.example.com

# 5. Clean up
grpcurl -plaintext -d '{
  "network_id": "YOUR_NETWORK_ID",
  "cleanup": true
}' localhost:50051 fabricx.FabricXService/StopNetwork
```

## 🎯 Next Steps

Once the runtime is working:

1. **Build the TypeScript SDK** to consume this runtime
2. **Create the CLI** for developer-friendly commands
3. **Add chaincode templates** for quick deployment
4. **Test full lifecycle** (init → deploy → invoke → query)

## 💡 Development Tips

```bash
# Watch mode (requires 'air')
make dev

# Run tests
make test

# Code coverage
make test-coverage

# Format code
make fmt

# Lint
make lint

# Build for all platforms
make build-all
```

## 🐳 Docker Deployment

```bash
# Build Docker image
make docker-build

# Run in Docker
docker run -d \
  -p 50051:50051 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  fabricx-runtime:latest
```

## 📝 Environment Variables

```bash
# Change gRPC port
export FABRICX_PORT=50052

# Change network data location
export FABRICX_NETWORK_PATH=/var/fabricx

# Then run
./bin/fabricx-runtime
```

## ✅ Success Checklist

- [ ] Go 1.23+ installed
- [ ] Docker running
- [ ] Dependencies installed (`go mod download`)
- [ ] Binary built (`./bin/fabricx-runtime` exists)
- [ ] Fabric images pulled
- [ ] Runtime starts without errors
- [ ] Can initialize test network via grpcurl
- [ ] Docker containers appear after init
- [ ] Can stop network and cleanup
