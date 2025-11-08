# FabricX Go Runtime Core

**🎯 Zero-Installation Hyperledger Fabric Development**

The Go runtime is the heart of FabricX, providing a **fully containerized** Hyperledger Fabric experience. Developers don't need to install Fabric binaries—everything runs in Docker containers automatically.

## 🚀 The FabricX Promise

```bash
# All you need is Docker installed
docker --version

# Everything else is handled automatically
npx fabricx init
```

**No manual downloads. No complex setup. Just Docker + FabricX.**

## Architecture

```
┌─────────────────────────────────────────┐
│   TypeScript CLI/SDK (via gRPC)         │
│   "npx fabricx init"                    │
└─────────────────┬───────────────────────┘
                  │ gRPC (port 50051)
                  ▼
┌─────────────────────────────────────────┐
│      FabricX Go Runtime Core            │
│  ┌─────────────────────────────────┐   │
│  │  gRPC Server                    │   │
│  └─────────────────────────────────┘   │
│  ┌─────────────────────────────────┐   │
│  │  Network Module                 │   │
│  │  • Bootstrap via Docker         │   │
│  │  • Generate crypto in container │   │
│  │  • No local binaries needed     │   │
│  └─────────────────────────────────┘   │
│  ┌─────────────────────────────────┐   │
│  │  Chaincode Module               │   │
│  │  • Package in container         │   │
│  │  • Install via docker exec      │   │
│  │  • Lifecycle in container       │   │
│  └─────────────────────────────────┘   │
│  ┌─────────────────────────────────┐   │
│  │  Docker Module                  │   │
│  │  • Pull Fabric images           │   │
│  │  • Orchestrate containers       │   │
│  │  • Execute in containers        │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────┐
│ Hyperledger Fabric Network (Docker)     │
│ ┌─────────────────────────────────────┐ │
│ │ hyperledger/fabric-tools:2.5        │ │
│ │ • cryptogen, configtxgen            │ │
│ │ • peer lifecycle commands           │ │
│ └─────────────────────────────────────┘ │
│ ┌─────────────────────────────────────┐ │
│ │ hyperledger/fabric-peer:2.5         │ │
│ │ hyperledger/fabric-orderer:2.5      │ │
│ │ hyperledger/fabric-ca:1.5           │ │
│ │ couchdb:3.3                         │ │
│ └─────────────────────────────────────┘ │
└─────────────────────────────────────────┘
```

## Key Difference: Containerized Operations

### ❌ Traditional Approach

```bash
# Developer has to:
1. Download Fabric binaries manually
2. Set PATH variables
3. Install cryptogen, configtxgen, peer CLI
4. Manage versions manually
5. Deal with OS-specific issues
```

### ✅ FabricX Approach

```bash
# FabricX handles everything:
1. Pulls Docker images automatically
2. Runs cryptogen in container: docker run fabric-tools cryptogen
3. Runs configtxgen in container: docker run fabric-tools configtxgen
4. Executes peer commands: docker exec peer0.org1 peer lifecycle chaincode...
5. Zero manual installation required
```

## Prerequisites

**Only Docker is required!**

```bash
# Check if Docker is installed
docker --version
# Docker version 20.10+ required

docker-compose --version
# Docker Compose 1.29+ required
```

That's it. No Fabric binaries, no Go SDKs, no cryptogen downloads needed.

## Quick Start

### 1. Build the Runtime

```bash
# Install Go dependencies
make deps

# Generate protobuf files
make proto

# Build the runtime binary
make build
```

### 2. Run the Runtime

```bash
# Start the gRPC server
./bin/fabricx-runtime

# Output:
# 🚀 FabricX Runtime v0.1.0 starting on port 50051
# 📦 All Fabric operations will run in Docker containers
# ✅ No local Fabric binaries required!
```

### 3. The Runtime Will Automatically

1. **Pull Fabric Docker images** when first initializing a network
2. **Generate crypto material** inside `fabric-tools` container
3. **Start Fabric network** (orderers, peers, CAs, CouchDB)
4. **Execute all commands** inside peer containers

## How It Works: Containerized Operations

### Crypto Generation

**Traditional:**

```bash
# Requires cryptogen binary installed locally
cryptogen generate --config=crypto-config.yaml
```

**FabricX:**

```go
// Runs cryptogen inside Docker container
docker run --rm \
  -v /config:/config \
  -v /crypto-config:/crypto-config \
  hyperledger/fabric-tools:2.5 \
  cryptogen generate --config=/config/crypto-config.yaml --output=/crypto-config
```

### Chaincode Deployment

**Traditional:**

```bash
# Requires peer CLI installed locally
peer lifecycle chaincode install chaincode.tar.gz
```

**FabricX:**

```go
// Executes inside running peer container
docker exec peer0.org1.example.com \
  peer lifecycle chaincode install /tmp/chaincode.tar.gz
```

### Transaction Invocation

**Traditional:**

```bash
# Requires peer CLI and proper environment setup
export CORE_PEER_ADDRESS=localhost:7051
peer chaincode invoke -n mycc -c '{"Args":["invoke"]}'
```

**FabricX:**

```go
// Executes inside peer container with proper environment
docker exec \
  -e CORE_PEER_LOCALMSPID=Org1MSP \
  -e CORE_PEER_ADDRESS=peer0.org1.example.com:7051 \
  peer0.org1.example.com \
  peer chaincode invoke -n mycc -c '{"Args":["invoke"]}'
```

## Docker Images Used

FabricX automatically pulls and uses these official Hyperledger images:

| Image | Version | Purpose |
|-------|---------|---------|
| `hyperledger/fabric-peer` | 2.5 | Peer nodes |
| `hyperledger/fabric-orderer` | 2.5 | Ordering service |
| `hyperledger/fabric-ca` | 1.5 | Certificate Authority |
| `hyperledger/fabric-tools` | 2.5 | CLI tools (cryptogen, configtxgen, peer) |
| `couchdb` | 3.3 | State database |

These images contain **all Fabric binaries**, so developers never need to install anything manually.

## Development

### Running in Development Mode

```bash
# Auto-reload on file changes
make dev
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

## gRPC Service API

The runtime exposes these containerized operations via gRPC:

### Network Operations

- **InitNetwork**: Bootstrap network using Docker containers
  - Pulls Fabric images if not present
  - Generates crypto in `fabric-tools` container
  - Starts orderers, peers, CAs via docker-compose

- **StopNetwork**: Stop and cleanup containers
  - Stops all containers via docker-compose down
  - Optionally removes volumes

- **GetNetworkStatus**: Check container health
  - Lists running containers
  - Returns peer/orderer status

### Chaincode Operations

- **DeployChaincode**: Full lifecycle in containers
  - Package in `fabric-tools` container
  - Install via `docker exec` into peer containers
  - Approve/commit using containerized peer CLI

- **InvokeTransaction**: Execute transaction
  - Runs `docker exec` into peer container
  - Submits transaction to orderer
  - Returns transaction ID

- **QueryLedger**: Query state
  - Executes query in peer container
  - Returns query results

### Monitoring

- **StreamLogs**: Real-time container logs
  - Streams via `docker-compose logs -f`

## Configuration

### Environment Variables

```bash
# gRPC server port (default: 50051)
export FABRICX_PORT=50051

# Base path for network data (default: /tmp/fabricx)
export FABRICX_NETWORK_PATH=/tmp/fabricx
```

### Network Configuration

Configure via gRPC `InitNetworkRequest`:

```protobuf
message InitNetworkRequest {
  string network_name = 1;      // Network ID
  int32 num_orgs = 2;            // Number of orgs (default: 2)
  string channel_name = 3;       // Channel name (default: "mychannel")
  map<string, string> config = 4; // Custom config
}
```

## Docker Deployment

### Build Runtime Image

```bash
make docker-build
```

### Run Runtime Container

```bash
docker run -d \
  -p 50051:50051 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd)/networks:/var/fabricx \
  fabricx-runtime:latest
```

**Important:** The runtime container needs access to the Docker socket to manage Fabric containers.

## Troubleshooting

### Common Issues

**1. "Docker not available"**

```bash
# Ensure Docker is running
docker ps

# Check Docker socket permissions
ls -la /var/run/docker.sock
```

**2. "Failed to pull Fabric images"**

```bash
# Manually pull images
docker pull hyperledger/fabric-peer:2.5
docker pull hyperledger/fabric-orderer:2.5
docker pull hyperledger/fabric-ca:1.5
docker pull hyperledger/fabric-tools:2.5
docker pull couchdb:3.3
```

**3. "Container failed to start"**

```bash
# Check container logs
docker-compose -f <network-path>/config/docker-compose.yaml logs

# Check if ports are available
netstat -tulpn | grep -E '7050|7051|7054'
```

**4. "Permission denied" when accessing Docker socket**

```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Log out and back in
```

### Debugging

```bash
# View runtime logs
./bin/fabricx-runtime

# View network container logs
docker-compose -f /tmp/fabricx/<network-id>/config/docker-compose.yaml logs -f

# Execute commands in peer container manually
docker exec -it peer0.org1.example.com bash
peer channel list
```

## Testing with grpcurl

```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List services
grpcurl -plaintext localhost:50051 list

# Initialize a network
grpcurl -plaintext -d '{
  "network_name": "test-net",
  "num_orgs": 2,
  "channel_name": "mychannel"
}' localhost:50051 fabricx.FabricXService/InitNetwork
```

## Performance

- **Network Init**: 30-60s (includes image pulls on first run, crypto generation)
- **Subsequent Inits**: 15-30s (images cached)
- **Chaincode Deploy**: 20-40s (packaging, installation, approval, commit)
- **Transactions**: 1-2s (endorsement + ordering + commit)

## Security Notes

⚠️ **This runtime is designed for local development only.**

For production:

- Enable TLS for gRPC server
- Enable TLS between Fabric components
- Implement authentication/authorization
- Use secure credential management
- Run in isolated network namespace
- Restrict Docker socket access

## Advantages of Containerized Approach

✅ **Zero Manual Installation**: No Fabric binaries to download  
✅ **Version Control**: Fabric version pinned in Docker images  
✅ **Cross-Platform**: Works on Mac, Linux, Windows (with WSL)  
✅ **Reproducible**: Same images, same behavior everywhere  
✅ **Easy Updates**: Update Fabric by changing image tags  
✅ **Isolation**: Fabric tools don't pollute host system  
✅ **Cleanup**: Remove containers = clean slate  

## License

Apache 2.0

## Support

- GitHub: [github.com/temmyjay001/fabricx](https://github.com/temmyjay001/fabricx)
- Docs: [docs.fabricx.dev](https://docs.fabricx.dev)
