# FabricX Chaincode Templates

Production-ready chaincode templates for common use cases.

## Available Templates

### Asset Transfer

Basic CRUD operations for assets with ownership tracking.

**Languages:** Go, TypeScript  
**Use Cases:** Supply chain, inventory management, document tracking

```bash
npx fabricx scaffold asset-transfer ./my-chaincode --lang go
```

### ERC-20 Token

Fungible token standard with minting, transfers, and allowances.

**Languages:** Go, TypeScript  
**Use Cases:** Loyalty points, digital currencies, voting tokens

```bash
npx fabricx scaffold erc20 ./my-token --lang go
```

### Escrow

Multi-party escrow with time locks and arbitration.

**Languages:** Go, TypeScript  
**Use Cases:** Payment escrow, conditional transfers, dispute resolution

```bash
npx fabricx scaffold escrow ./my-escrow --lang go
```

### Supply Chain

Track products through supply chain with provenance.

**Languages:** Go, TypeScript  
**Use Cases:** Product tracking, authenticity verification, compliance

```bash
npx fabricx scaffold supply-chain ./my-supply --lang go
```

## Template Structure

Each template follows this structure:

```
template-name/
├── go/
│   ├── main.go              # Chaincode implementation
│   ├── main_test.go         # Unit tests
│   ├── go.mod              # Go module definition
│   ├── go.sum              # Dependency checksums
│   └── README.md           # Template-specific docs
└── typescript/
    ├── src/
    │   └── index.ts        # Chaincode implementation
    ├── test/
    │   └── index.test.ts   # Unit tests
    ├── package.json        # NPM dependencies
    ├── tsconfig.json       # TypeScript config
    └── README.md           # Template-specific docs
```

## Using Templates

### Via CLI (Recommended)

```bash
# Initialize a network
npx fabricx init

# Scaffold a template
npx fabricx scaffold erc20 ./my-token --lang go

# Deploy it
npx fabricx deploy my-token ./my-token
```

### Via SDK

```typescript
import { FabricX, TemplateManager } from '@fabricx/sdk';

const templateMgr = new TemplateManager();

// List available templates
const templates = templateMgr.list();

// Scaffold a template
await templateMgr.scaffold('erc20', './my-token', { language: 'go' });

// Deploy it
const fx = new FabricX();
await fx.deployChaincode('my-token', {
  path: './my-token',
  version: '1.0',
  language: 'golang',
});
```

## Creating Custom Templates

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines on adding new templates.

Template requirements:

- ✅ Must compile without errors
- ✅ Must include unit tests
- ✅ Must have clear documentation
- ✅ Must follow Fabric best practices
- ✅ Must be production-ready
