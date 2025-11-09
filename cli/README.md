# @fabricx/cli

> Command-line interface for FabricX Developer Toolkit

A beautiful, production-ready CLI for managing Hyperledger Fabric networks with the FabricX toolkit.

## 🚀 Features

- ✅ **Beautiful Output** - Colored, formatted output with spinners and progress indicators
- ✅ **Smart Defaults** - Remembers last network ID and configuration
- ✅ **Production SDK** - Uses the production-grade @fabricx/sdk
- ✅ **Configuration Management** - Persistent config stored in `~/.fabricxrc`
- ✅ **Error Handling** - Clear, helpful error messages
- ✅ **Connection Pooling** - Efficient connection management
- ✅ **Real-time Logs** - Stream container logs with formatting
- ✅ **Progress Tracking** - Shows operation duration and status

## 📦 Installation

### Global Installation (Recommended)

```bash
npm install -g @fabricx/cli
```

### Use with npx (No Installation)

```bash
npx @fabricx/cli init
```

### Local Installation

```bash
npm install @fabricx/cli
```

## 🎯 Quick Start

```bash
# Initialize a network
fabricx init

# Deploy chaincode
fabricx deploy mycc ./chaincode/mycc

# Invoke a transaction
fabricx invoke mycc createAsset asset1 owner1 100

# Query the ledger
fabricx query mycc getAsset asset1

# Get network status
fabricx status

# Stop the network
fabricx stop --cleanup
```

## 📚 Commands

### `init` - Initialize Network

Create a new Hyperledger Fabric network.

```bash
fabricx init [options]
```

**Options:**
- `-n, --name <name>` - Network name (default: "fabricx-network")
- `-o, --orgs <number>` - Number of organizations (default: 2)
- `-c, --channel <name>` - Channel name (default: "mychannel")

**Examples:**

```bash
# Basic initialization
fabricx init

# Custom configuration
fabricx init --name supply-chain --orgs 3 --channel logistics

# With global options
fabricx init --server myserver.com:50051 --log-level debug
```

**Output:**

```
✔ Network initialized in 45.2s

📋 Network Details:
  Network ID: abc12345
  Name: supply-chain
  Organizations: 3
  Channel: logistics
  Endpoints:
    • localhost:7051
    • localhost:8051
    • localhost:9051

✓ Network ready for chaincode deployment

💡 Tip: Save this network ID or use it automatically in future commands
```

---

### `status` - Network Status

Get detailed status of a network.

```bash
fabricx status [network-id]
```

**Arguments:**
- `network-id` - Network ID (optional, uses last network if not provided)

**Examples:**

```bash
# Status of last network
fabricx status

# Status of specific network
fabricx status abc12345
```

**Output:**

```
✔ Network status retrieved

📊 Network Status:
  Network ID: abc12345
  Running: Yes
  Status: 6 containers running

👥 Peers:
  • peer0.org1.example.com
    Organization: Org1
    Status: running
    Endpoint: localhost:7051
  • peer0.org2.example.com
    Organization: Org2
    Status: running
    Endpoint: localhost:8051

⚙️  Orderers:
  • orderer.example.com
    Status: running
    Endpoint: localhost:7050

🔗 Connection Pool:
  Total Connections: 2
  Active: 0
  Idle: 2
  Total Requests: 15
```

---

### `deploy` - Deploy Chaincode

Deploy chaincode to the network.

```bash
fabricx deploy <chaincode> [path] [options]
```

**Arguments:**
- `chaincode` - Chaincode name (required)
- `path` - Path to chaincode directory (optional)

**Options:**
- `-v, --version <version>` - Chaincode version (default: "1.0")
- `-l, --language <lang>` - Language: golang, node, java (default: "golang")
- `-n, --network <id>` - Network ID (uses last network if not provided)
- `-e, --endorsement <orgs>` - Endorsement policy organizations (comma-separated)

**Examples:**

```bash
# Basic deployment
fabricx deploy mycc

# With custom path and version
fabricx deploy mycc ./chaincode/mycc --version 2.0

# Node.js chaincode
fabricx deploy mycc ./chaincode --language node

# With endorsement policy
fabricx deploy mycc --endorsement "Org1,Org2"

# To specific network
fabricx deploy mycc --network abc12345
```

**Output:**

```
✔ Chaincode deployed in 38.7s

📦 Deployment Details:
  Chaincode ID: mycc-a1b2c3d4
  Name: mycc
  Version: 1.0
  Language: golang
  Path: ./mycc

✓ Chaincode ready for transactions
```

---

### `invoke` - Invoke Transaction

Submit a transaction to the ledger.

```bash
fabricx invoke <chaincode> <function> [args...] [options]
```

**Arguments:**
- `chaincode` - Chaincode name (required)
- `function` - Function name (required)
- `args...` - Function arguments (optional)

**Options:**
- `-n, --network <id>` - Network ID (uses last network if not provided)
- `--transient` - Use transient data

**Examples:**

```bash
# Simple invocation
fabricx invoke mycc createAsset asset1 owner1 100

# Multiple arguments
fabricx invoke mycc transferAsset asset1 owner2

# With network ID
fabricx invoke mycc createAsset asset1 owner1 100 --network abc12345

# With transient data
fabricx invoke mycc privateFunction arg1 --transient
```

**Output:**

```
✔ Transaction completed in 2.3s

📝 Transaction Details:
  Transaction ID: tx-a1b2c3d4e5f6
  Chaincode: mycc
  Function: createAsset
  Arguments: ["asset1","owner1","100"]

📄 Response:
{
  "status": "success",
  "assetId": "asset1"
}
```

---

### `query` - Query Ledger

Read data from the ledger.

```bash
fabricx query <chaincode> <function> [args...] [options]
```

**Arguments:**
- `chaincode` - Chaincode name (required)
- `function` - Function name (required)
- `args...` - Function arguments (optional)

**Options:**
- `-n, --network <id>` - Network ID (uses last network if not provided)

**Examples:**

```bash
# Query single asset
fabricx query mycc getAsset asset1

# Query all assets
fabricx query mycc getAllAssets

# Query with multiple arguments
fabricx query mycc getAssetsByOwner owner1

# Query specific network
fabricx query mycc getAsset asset1 --network abc12345
```

**Output:**

```
✔ Query completed in 0.8s

🔍 Query Details:
  Chaincode: mycc
  Function: getAsset
  Arguments: ["asset1"]

📄 Result:
{
  "ID": "asset1",
  "Owner": "owner1",
  "Value": 100,
  "CreatedAt": "2025-01-15T10:30:00Z"
}
```

---

### `logs` - Stream Logs

Stream real-time logs from containers.

```bash
fabricx logs [container] [options]
```

**Arguments:**
- `container` - Container name (optional, streams all containers if not provided)

**Options:**
- `-n, --network <id>` - Network ID (uses last network if not provided)

**Examples:**

```bash
# Stream all logs
fabricx logs

# Stream specific container
fabricx logs peer0.org1.example.com

# Stream from specific network
fabricx logs --network abc12345

# Stream orderer logs
fabricx logs orderer.example.com
```

**Output:**

```
📜 Streaming logs from network: abc12345
   Press Ctrl+C to stop

[10:30:15] [peer0.org1.example.com] Received block [5] from orderer
[10:30:15] [peer0.org1.example.com] Validated block [5] in 12ms
[10:30:16] [orderer.example.com] Ordering block [6]
[10:30:16] [peer0.org2.example.com] Committed block [5] to ledger
```

---

### `stop` - Stop Network

Stop and cleanup a network.

```bash
fabricx stop [network-id] [options]
```

**Arguments:**
- `network-id` - Network ID (optional, uses last network if not provided)

**Options:**
- `--cleanup` - Remove containers and volumes

**Examples:**

```bash
# Stop last network (keep containers)
fabricx stop

# Stop and cleanup
fabricx stop --cleanup

# Stop specific network
fabricx stop abc12345 --cleanup
```

**Output:**

```
✔ Network stopped successfully
  All containers and volumes removed
```

---

### `config` - Configuration

Manage CLI configuration.

```bash
fabricx config [options]
```

**Options:**
- `--show` - Show current configuration
- `--reset` - Reset configuration to defaults
- `--set-server <address>` - Set server address
- `--set-timeout <ms>` - Set timeout
- `--set-log-level <level>` - Set log level

**Examples:**

```bash
# Show configuration
fabricx config --show

# Set server address
fabricx config --set-server myserver.com:50051

# Set timeout
fabricx config --set-timeout 180000

# Set log level
fabricx config --set-log-level debug

# Reset to defaults
fabricx config --reset
```

**Output (--show):**

```
⚙️  Current Configuration:
  Server: localhost:50051
  Timeout: 120000ms
  TLS: Disabled
  Log Level: info
  Last Network: abc12345
```

---

## 🌐 Global Options

These options can be used with any command:

- `-s, --server <address>` - FabricX runtime server address
- `-t, --timeout <ms>` - Request timeout in milliseconds
- `--tls` - Enable TLS
- `--log-level <level>` - Log level (debug, info, warn, error, silent)

**Examples:**

```bash
# Use custom server
fabricx init --server myserver.com:50051

# With TLS
fabricx status --server myserver.com:50051 --tls

# Debug mode
fabricx deploy mycc --log-level debug

# Custom timeout
fabricx invoke mycc func1 --timeout 300000
```

## 🔧 Configuration File

The CLI stores configuration in `~/.fabricxrc`:

```json
{
  "serverAddr": "localhost:50051",
  "timeout": 120000,
  "useTls": false,
  "logLevel": "info",
  "lastNetworkId": "abc12345"
}
```

## 💡 Tips & Best Practices

### 1. Use Last Network Shortcut

The CLI automatically remembers your last network:

```bash
fabricx init                    # Creates network abc12345
fabricx deploy mycc             # Uses abc12345 automatically
fabricx invoke mycc func1       # Uses abc12345 automatically
```

### 2. Save Network IDs

For multiple networks, save the network ID:

```bash
# Create networks
fabricx init --name dev  > dev-network.txt
fabricx init --name prod > prod-network.txt

# Use specific network
DEV_ID=$(cat dev-network.txt | grep "Network ID" | awk '{print $3}')
fabricx invoke mycc func1 --network $DEV_ID
```

### 3. Create Aliases

Add to your shell profile:

```bash
alias fxi='fabricx init'
alias fxd='fabricx deploy'
alias fxq='fabricx query'
alias fxs='fabricx status'
```

### 4. Use Environment Variables

```bash
export FABRICX_SERVER=myserver.com:50051
export FABRICX_TIMEOUT=180000

fabricx init  # Uses environment variables
```

### 5. Combine with Scripts

```bash
#!/bin/bash
# setup.sh

# Initialize network
NETWORK_ID=$(fabricx init --name test | grep "Network ID" | awk '{print $3}')

# Deploy chaincode
fabricx deploy mycc --network $NETWORK_ID

# Run tests
fabricx invoke mycc test1 --network $NETWORK_ID
fabricx invoke mycc test2 --network $NETWORK_ID

# Verify
fabricx query mycc getResults --network $NETWORK_ID

# Cleanup
fabricx stop $NETWORK_ID --cleanup
```

## 🐛 Troubleshooting

### Connection Issues

```bash
# Check if runtime is running
curl localhost:50051

# Use debug mode
fabricx status --log-level debug

# Check configuration
fabricx config --show
```

### Network Not Found

```bash
# List Docker networks
docker network ls | grep fabricx

# Check last network ID
fabricx config --show

# Re-initialize
fabricx init
```

### Timeout Errors

```bash
# Increase timeout
fabricx deploy mycc --timeout 300000

# Or set globally
fabricx config --set-timeout 300000
```

## 📊 Output Formats

The CLI uses colored output:
- 🟢 Green - Success messages
- 🔴 Red - Error messages
- 🟡 Yellow - Warnings and tips
- 🔵 Cyan - Headers and labels
- ⚪ Gray - Details and metadata

## 🔗 Integration

### With Scripts

```javascript
// Node.js script
const { execSync } = require('child_process');

const networkId = execSync('fabricx init', { encoding: 'utf-8' })
  .match(/Network ID: (\w+)/)[1];

console.log(`Network created: ${networkId}`);
```

### With CI/CD

```yaml
# .github/workflows/test.yml
- name: Initialize Fabric Network
  run: fabricx init --name ci-network

- name: Deploy Chaincode
  run: fabricx deploy mycc ./chaincode

- name: Run Tests
  run: |
    fabricx invoke mycc test1
    fabricx query mycc getResults

- name: Cleanup
  run: fabricx stop --cleanup
```

## 📄 License

Apache 2.0

## 🤝 Contributing

Contributions welcome! Please read our contributing guidelines.

## 🔗 Links

- [SDK Documentation](https://github.com/temmyjay001/fabricx/tree/main/sdk)
- [Runtime Documentation](https://github.com/temmyjay001/fabricx/tree/main/fabricx-core)
- [Examples](https://github.com/temmyjay001/fabricx/tree/main/examples)