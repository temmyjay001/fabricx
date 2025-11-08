# @fabricx/sdk - Production Setup

## Installation

```bash
npm install @fabricx/sdk
```

## Quick Start

```typescript
import { FabricX } from '@fabricx/sdk';

const fabricx = new FabricX({
  serverAddr: 'localhost:50051',
  useConnectionPool: true,
  minConnections: 2,
  maxConnections: 10,
  logger: {
    level: 'info',
  },
});

// Initialize network
const network = await fabricx.initNetwork();

// Deploy chaincode
await fabricx.deployChaincode('mycc', {
  path: './chaincode/mycc',
});

// Invoke transaction
await fabricx.invoke('mycc', 'createAsset', ['asset1', 'value1']);

// Query ledger
const result = await fabricx.query('mycc', 'getAsset', ['asset1']);

// Cleanup
await fabricx.close();
```

## Production Features

### 1. Connection Pooling

Efficient connection management with automatic pooling:

```typescript
const fabricx = new FabricX({
  useConnectionPool: true,
  minConnections: 2,
  maxConnections: 10,
  idleTimeout: 300000,
  healthCheckInterval: 30000,
});

// Get pool statistics
const stats = fabricx.getPoolStats();
console.log(`Active: ${stats.activeConnections}, Idle: ${stats.idleConnections}`);
```

### 2. Automatic Retry with Multiple Strategies

Built-in retry logic with exponential backoff:

```typescript
const fabricx = new FabricX({
  retry: {
    maxAttempts: 3,
    initialDelay: 1000,
    maxDelay: 30000,
    strategy: 'exponential',
    backoffMultiplier: 2,
    jitter: true,
  },
});
```

### 3. Advanced Logging

Production-grade logging with multiple levels:

```typescript
const fabricx = new FabricX({
  logger: {
    level: 'debug', // 'debug' | 'info' | 'warn' | 'error' | 'silent'
    enableConsole: true,
  },
});

// Access logger
const logger = fabricx.getLogger();
logger.info('Custom log message', { context: 'data' });

// Get logs
const logs = logger.getLogs();
```

### 4. TLS Support

Secure connections with mutual TLS:

```typescript
import fs from 'fs';

const fabricx = new FabricX({
  useTls: true,
  tlsConfig: {
    rootCert: fs.readFileSync('./certs/ca.crt'),
    clientCert: fs.readFileSync('./certs/client.crt'),
    clientKey: fs.readFileSync('./certs/client.key'),
  },
});
```

### 5. Connection Monitoring

Real-time connection state monitoring:

```typescript
// Get connection state
const state = fabricx.getConnectionState();
console.log(`Connection state: ${state}`);

// Check if connected
const isConnected = fabricx.isConnected();
```

### 6. Graceful Shutdown

Proper resource cleanup:

```typescript
// Handle shutdown
process.on('SIGTERM', async () => {
  await fabricx.close();
  process.exit(0);
});
```

## Error Handling

All errors are wrapped in `FabricXError` with specific error codes:

```typescript
import { FabricXError } from '@fabricx/sdk';

try {
  await fabricx.invoke('mycc', 'myFunc', ['arg1']);
} catch (error) {
  if (error instanceof FabricXError) {
    console.error(`Error Code: ${error.code}`);
    console.error(`Message: ${error.message}`);
    console.error(`Details:`, error.details);
  }
}
```

### Common Error Codes

- `CONNECTION_ERROR` - Cannot connect to runtime
- `TIMEOUT` - Operation timed out
- `NO_NETWORK_ID` - No network initialized
- `MAX_RETRIES_EXCEEDED` - All retry attempts failed
- `AUTHENTICATION_ERROR` - Auth failed
- `PERMISSION_DENIED` - Insufficient permissions
- `NOT_FOUND` - Resource not found

## Testing

```bash
# Run tests
npm test

# Watch mode
npm run test:watch

# Coverage
npm run test:coverage
```

## Development

```bash
# Build
npm run build

# Watch mode
npm run watch

# Lint
npm run lint

# Format
npm run format

# Type check
npm run type-check
```

## Environment Variables

```bash
# Server address
FABRICX_SERVER_ADDR=localhost:50051

# Timeout (ms)
FABRICX_TIMEOUT=120000

# Log level
FABRICX_LOG_LEVEL=info

# Enable TLS
FABRICX_USE_TLS=false
```

## Performance Tips

1. **Use Connection Pooling** - Enable for high-throughput applications
2. **Tune Timeouts** - Adjust based on network latency
3. **Configure Retry** - Use exponential backoff for network operations
4. **Monitor Logs** - Use structured logging in production
5. **Graceful Shutdown** - Always call `close()` before exit

## License

Apache 2.0
