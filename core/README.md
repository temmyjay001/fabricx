# FabricX Runtime

**Zero-Installation Hyperledger Fabric Development**

## Quick Start

### Prerequisites

- **Docker** (that's it!)

### Installation

#### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/temmyjay001/fabricx/main/core/install.sh | bash
```

#### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/temmyjay001/fabricx/main/core/install.ps1 | iex
```

#### Manual Download

Download the appropriate binary for your system from the [releases page](https://github.com/temmyjay001/fabricx/releases):

- **macOS Intel**: `fabricx-runtime-0.1.0-darwin-amd64.tar.gz`
- **macOS Apple Silicon**: `fabricx-runtime-0.1.0-darwin-arm64.tar.gz`
- **Linux x64**: `fabricx-runtime-0.1.0-linux-amd64.tar.gz`
- **Linux ARM64**: `fabricx-runtime-0.1.0-linux-arm64.tar.gz`
- **Windows**: `fabricx-runtime-0.1.0-windows-amd64.zip`

Extract and run:

```bash
tar -xzf fabricx-runtime-*.tar.gz
./fabricx-runtime-*/fabricx-runtime
```

### Run

```bash
fabricx-runtime
```

That's it! The runtime will:

- ✅ Check Docker availability
- ✅ Start gRPC server on port 50051
- ✅ Manage Fabric networks automatically in Docker

## Building from Source

```bash
git clone https://github.com/temmyjay001/fabricx
cd fabricx/core
make release
```
