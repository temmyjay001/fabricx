# FabricX CLI Client Guide

The FabricX CLI client is a command-line tool for interacting with the FabricX runtime. It provides an easy way to manage Fabric networks, deploy chaincode, and execute transactions.

## 📦 Building the Client

```bash
# Build both runtime and client
make all

# Or build just the client
make client

# Result: bin/fabricx-client
```

## 🚀 Quick Start

### 1. Start the Runtime Server

In one terminal:

```bash
./bin/fabricx-runtime
```

### 2. Use the Client

In another terminal:

```bash
# Initialize a network
./bin/fabricx-client init

# Example output:
# 🚀 Initializing Fabric network...
#    Name: fabricx-network
#    Organizations: 2
#    Channel: mychannel
# 
# ✅ Network initialized successfully!
#    Network ID: abc12345
#    Endpoints: localhost:7051, localhost:8051
# 
# 💡 Save this network ID for future commands
```

## 📖 Commands Reference

### `init` - Initialize Network

Creates a new Hyperledger Fabric network.

**Usage:**

```bash
fabricx-client init [options]
```

**Options:**

- `--name <name>` - Network name (default: "fabricx-network")
- `--orgs <num>` - Number of organizations (default: 2)
- `--channel <name>` - Channel name (default: "mychannel")

**Examples:**

```bash
# Basic initialization (2 orgs, default settings)
./bin/fabricx-client init

# Custom configuration
./bin/fabricx-client init --name my-network --orgs 3 --channel supply-chain

# Quick test network
./bin/fabricx-client init --name test --orgs 2
```

**Output:**

```
🚀 Initializing Fabric network...
   Name: my-network
   Organizations: 3
   Channel: supply-chain

✅ Network initialized successfully!
   Network ID: f3a8b2c1
   Endpoints: localhost:7051, localhost:8051, localhost:9051

💡 Save this network ID for future commands
```

---

### `status` - Get Network Status

Check if a network is running and get details about peers and orderers.

**Usage:**

```bash
fabricx-client status <network-id>
```

**Example:**

```bash
./bin/fabricx-client status f3a8b2c1
```

**Output:**

```
Network Status: f3a8b2c1
  Running: true
  Status: 6 containers running

Peers:
  - peer0.org1.example.com (localhost:7051)
    Organization: Org1
    Status: running
  - peer0.org2.example.com (localhost:8051)
    Organization: Org2
    Status: running

Orderers:
  - orderer.example.com (localhost:7050)
    Status: running
```

---

### `deploy` - Deploy Chaincode

Package, install, approve, and commit chaincode to the network.

**Usage:**

```bash
fabricx-client deploy <network-id> <chaincode-name> <chaincode-path> [options]
```

**Options:**

- `--version <v>` - Chaincode version (default: "1.0")
- `--lang <language>` - Language: go, node, java (default: "golang")

**Examples:**

```bash
# Deploy Go chaincode
./bin/fabricx-client deploy f3a8b2c1 mycc ./chaincode/mycc

# Deploy with specific version
./bin/fabricx-client deploy f3a8b2c1 mycc ./chaincode/mycc --version 2.0

# Deploy Node.js chaincode
./bin/fabricx-client deploy f3a8b2c1 mycc ./chaincode/mycc --lang node
```

**Output:**

```
📦 Deploying chaincode...
   Network: f3a8b2c1
   Chaincode: mycc
   Path: ./chaincode/mycc
   Version: 1.0
   Language: golang

✅ Chaincode deployed successfully!
   Chaincode ID: mycc-a1b2c3d4
```

---

### `invoke` - Invoke Transaction

Submit a transaction to the ledger.

**Usage:**

```bash
fabricx-client invoke <network-id> <chaincode> <function> [args...]
```

**Examples:**

```bash
# Create an asset
./bin/fabricx-client invoke f3a8b2c1 mycc CreateAsset asset1 blue 20 tom 100

# Transfer asset
./bin/fabricx-client invoke f3a8b2c1 mycc TransferAsset asset1 jerry

# Update asset
./bin/fabricx-client invoke f3a8b2c1 mycc UpdateAsset asset1 red 30
```

**Output:**

```
📝 Invoking transaction...
   Network: f3a8b2c1
   Chaincode: mycc
   Function: CreateAsset
   Args: [asset1 blue 20 tom 100]

✅ Transaction invoked successfully!
   Transaction ID: a1b2c3d4e5f6...
   Payload: {"asset":"asset1","created":true}
```

---

### `query` - Query Ledger

Read data from the ledger without creating a transaction.

**Usage:**

```bash
fabricx-client query <network-id> <chaincode> <function> [args...]
```

**Examples:**

```bash
# Read an asset
./bin/fabricx-client query f3a8b2c1 mycc ReadAsset asset1

# Get all assets
./bin/fabricx-client query f3a8b2c1 mycc GetAllAssets

# Check asset exists
./bin/fabricx-client query f3a8b2c1 mycc AssetExists asset1
```

**Output:**

```
🔍 Querying ledger...
   Network: f3a8b2c1
   Chaincode: mycc
   Function: ReadAsset
   Args: [asset1]

✅ Query successful!
   Result:
   {
     "ID": "asset1",
     "Color": "blue",
     "Size": 20,
     "Owner": "tom",
     "AppraisedValue": 100
   }
```

---

### `logs` - Stream Container Logs

View real-time logs from network containers.

**Usage:**

```bash
fabricx-client logs <network-id> [container-name]
```

**Examples:**

```bash
# Stream all logs
./bin/fabricx-client logs f3a8b2c1

# Stream specific container
./bin/fabricx-client logs f3a8b2c1 peer0.org1.example.com

# Stream orderer logs
./bin/fabricx-client logs f3a8b2c1 orderer.example.com
```

**Output:**

```
📜 Streaming logs from network f3a8b2c1
   Press Ctrl+C to stop

[peer0.org1.example.com] 2025-01-15 10:30:15.123 UTC [gossip.privdata] StoreBlock -> INFO Received block [5] from buffer
[peer0.org1.example.com] 2025-01-15 10:30:15.124 UTC [committer.txvalidator] Validate -> INFO [mychannel] Validated block [5] in 12ms
[orderer.example.com] 2025-01-15 10:30:15.125 UTC [orderer.consensus.solo] main -> INFO Ordering block [6]
```

**Tip:** Press `Ctrl+C` to stop streaming.

---

### `stop` - Stop Network

Stop and optionally cleanup a network.

**Usage:**

```bash
fabricx-client stop <network-id> [--cleanup]
```

**Options:**

- `--cleanup` - Remove containers, volumes, and network files

**Examples:**

```bash
# Stop network (containers remain)
./bin/fabricx-client stop f3a8b2c1

# Stop and cleanup everything
./bin/fabricx-client stop f3a8b2c1 --cleanup
```

**Output:**

```
🛑 Stopping network f3a8b2c1 (with cleanup)

✅ Network stopped successfully!
   All containers and volumes removed
```

---

## 🎯 Complete Workflow Example

Here's a complete example from network initialization to transaction execution:

```bash
# 1. Initialize network
./bin/fabricx-client init --name demo --orgs 2 --channel demochannel
# Output: Network ID: abc12345

# 2. Check status
./bin/fabricx-client status abc12345

# 3. Deploy chaincode
./bin/fabricx-client deploy abc12345 asset-transfer ./chaincode/asset-transfer

# 4. Create an asset
./bin/fabricx-client invoke abc12345 asset-transfer CreateAsset \
  asset1 blue 20 tom 100

# 5. Query the asset
./bin/fabricx-client query abc12345 asset-transfer ReadAsset asset1

# 6. Transfer asset
./bin/fabricx-client invoke abc12345 asset-transfer TransferAsset \
  asset1 jerry

# 7. Verify transfer
./bin/fabricx-client query abc12345 asset-transfer ReadAsset asset1

# 8. View logs
./bin/fabricx-client logs abc12345 peer0.org1.example.com

# 9. Stop network
./bin/fabricx-client stop abc12345 --cleanup
```

---

## ⚙️ Advanced Usage

### Custom Server Address

Connect to a remote runtime server:

```bash
./bin/fabricx-client -server myserver.example.com:50051 init
```

### Custom Timeout

Increase timeout for slow operations:

```bash
./bin/fabricx-client -timeout 300s deploy abc12345 mycc ./chaincode
```

### Combining Flags

```bash
./bin/fabricx-client -server localhost:50052 -timeout 180s \
  invoke abc12345 mycc MyFunction arg1 arg2
```

---

## 🐛 Troubleshooting

### "Failed to connect to server"

**Problem:** Runtime server is not running.

**Solution:**

```bash
# Start the runtime in another terminal
./bin/fabricx-runtime

# Verify it's running
curl localhost:50051
```

### "Network not found"

**Problem:** Network ID is incorrect or network was stopped.

**Solution:**

```bash
# List Docker networks
docker network ls | grep fabricx

# Reinitialize if needed
./bin/fabricx-client init
```

### "Timeout exceeded"

**Problem:** Operation took too long (e.g., deploying large chaincode).

**Solution:**

```bash
# Increase timeout
./bin/fabricx-client -timeout 300s deploy abc12345 mycc ./chaincode
```

### "Chaincode not found"

**Problem:** Chaincode path is incorrect or chaincode not deployed.

**Solution:**

```bash
# Check path exists
ls -la ./chaincode/mycc

# Deploy chaincode first
./bin/fabricx-client deploy abc12345 mycc ./chaincode/mycc

# Then invoke
./bin/fabricx-client invoke abc12345 mycc MyFunction
```

---

## 📝 Tips & Best Practices

### 1. Save Network IDs

```bash
# Save to file
./bin/fabricx-client init | tee network.txt

# Or use environment variable
export NETWORK_ID=$(./bin/fabricx-client init | grep "Network ID" | awk '{print $3}')
echo $NETWORK_ID
```

### 2. Create Aliases

```bash
# Add to ~/.bashrc or ~/.zshrc
alias fx='./bin/fabricx-client'

# Then use
fx init
fx status abc12345
fx invoke abc12345 mycc CreateAsset ...
```

### 3. Scripting

```bash
#!/bin/bash
# setup-network.sh

NETWORK_ID=$(./bin/fabricx-client init --name test --orgs 2 | grep "Network ID" | awk '{print $3}')
echo "Network created: $NETWORK_ID"

./bin/fabricx-client deploy $NETWORK_ID mycc ./chaincode/mycc
echo "Chaincode deployed"

./bin/fabricx-client invoke $NETWORK_ID mycc Init
echo "Chaincode initialized"
```

### 4. JSON Output Parsing

```bash
# Query and parse JSON
./bin/fabricx-client query abc12345 mycc ReadAsset asset1 | jq '.Owner'
```

---

## 🔗 Integration with TypeScript SDK

The CLI client is a standalone tool, but for programmatic access, use the TypeScript SDK:

```typescript
import { FabricX } from '@fabricx/sdk';

const fx = new FabricX({ serverAddr: 'localhost:50051' });
await fx.initNetwork({ name: 'test', numOrgs: 2 });
```

The CLI client is perfect for:

- ✅ Quick testing
- ✅ DevOps scripts
- ✅ Manual network management
- ✅ Debugging

The TypeScript SDK is better for:

- ✅ Application integration
- ✅ Automated workflows
- ✅ Complex orchestration
- ✅ Production deployments

---

## 🎉 Summary

The FabricX CLI client provides a user-friendly interface to the FabricX runtime:

| Command | Purpose | Time |
|---------|---------|------|
| `init` | Create network | 30-60s |
| `status` | Check health | <1s |
| `deploy` | Deploy chaincode | 20-40s |
| `invoke` | Submit transaction | 1-2s |
| `query` | Read data | <1s |
| `logs` | Stream logs | Real-time |
| `stop` | Cleanup | 5-10s |
