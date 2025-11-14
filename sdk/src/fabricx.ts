// sdk/src/fabricx.ts
import { GrpcClient } from './grpc/client';
import { ConnectionPool, ConnectionPoolConfig, PoolStats } from './grpc/connection-pool';
import {
  InitNetworkOptions,
  InitNetworkResult,
  DeployChaincodeOptions,
  DeployChaincodeResult,
  InvokeTransactionOptions,
  InvokeTransactionResult,
  QueryLedgerOptions,
  QueryLedgerResult,
  NetworkStatusResult,
  StopNetworkOptions,
  LogStreamHandler,
  FabricXError,
} from './types';
import { Logger, LogLevel } from './utils/logger';
import { RetryManager, RetryOptions } from './utils/retry';

/**
 * FabricX SDK Configuration
 */
export interface FabricXConfig extends ConnectionPoolConfig {
  /** Enable connection pooling (default: true) */
  useConnectionPool?: boolean;
  /** Logger configuration */
  logger?: {
    level?: LogLevel;
    enableConsole?: boolean;
  };
  /** Retry configuration for operations */
  retry?: RetryOptions;
  /** Enable automatic reconnection (default: true) */
  autoReconnect?: boolean;
}

/**
 * Production-grade FabricX SDK
 * Main class for interacting with Hyperledger Fabric networks
 */
export class FabricX {
  private client?: GrpcClient;
  private pool?: ConnectionPool;
  private config: Required<FabricXConfig>;
  private networkId?: string;
  private logger: Logger;
  private retryManager: RetryManager;
  private connectionWatcher?: () => void;

  constructor(config?: FabricXConfig) {
    this.config = {
      serverAddr: config?.serverAddr || 'localhost:50051',
      timeout: config?.timeout || 120000,
      useTls: config?.useTls || false,
      tlsConfig: config?.tlsConfig || {},
      maxRetries: config?.maxRetries || 3,
      retryDelay: config?.retryDelay || 1000,
      keepAliveTime: config?.keepAliveTime || 30000,
      keepAliveTimeout: config?.keepAliveTimeout || 10000,
      minConnections: config?.minConnections || 1,
      maxConnections: config?.maxConnections || 10,
      idleTimeout: config?.idleTimeout || 300000,
      healthCheckInterval: config?.healthCheckInterval || 30000,
      useConnectionPool: config?.useConnectionPool !== false,
      logger: {
        level: config?.logger?.level || ('info' as LogLevel),
        enableConsole: config?.logger?.enableConsole !== false,
      },
      retry: config?.retry || {},
      autoReconnect: config?.autoReconnect !== false,
    };

    // Initialize logger
    this.logger = new Logger(this.config.logger.level, this.config.logger.enableConsole);

    // Initialize retry manager
    this.retryManager = new RetryManager(this.config.retry);

    // Initialize connection
    if (this.config.useConnectionPool) {
      this.pool = new ConnectionPool(this.config);
      this.logger.info('Connection pool initialized');
    } else {
      this.client = new GrpcClient(this.config);
      this.logger.info('Single client initialized');
    }

    // Setup connection monitoring
    if (this.config.autoReconnect && this.client) {
      this.setupConnectionMonitoring();
    }
  }

  /**
   * Initialize a new Hyperledger Fabric network
   */
  async initNetwork(options?: InitNetworkOptions): Promise<InitNetworkResult> {
    this.logger.info('Initializing Fabric network', options);

    const result = await this.executeWithRetry(async (client) => {
      return client.initNetwork({
        network_name: options?.name || 'fabricx-network',
        num_orgs: options?.numOrgs || 2,
        channel_name: options?.channelName || 'mychannel',
        config: options?.config || {},
      });
    });

    if (result.success) {
      this.networkId = result.network_id;
      this.logger.info(`Network initialized: ${result.network_id}`, {
        endpoints: result.endpoints,
      });
    } else {
      this.logger.error('Network initialization failed', result.message);
    }

    return {
      success: result.success,
      message: result.message,
      networkId: result.network_id,
      endpoints: result.endpoints,
    };
  }

  /**
   * Deploy a chaincode to the network
   */
  async deployChaincode(
    chaincodeName: string,
    options?: DeployChaincodeOptions
  ): Promise<DeployChaincodeResult> {
    this.ensureNetworkId();
    this.logger.info(`Deploying chaincode: ${chaincodeName}`, options);

    const result = await this.executeWithRetry(async (client) => {
      return client.deployChaincode({
        network_id: this.networkId!,
        chaincode_name: chaincodeName,
        chaincode_path: options?.path || `./${chaincodeName}`,
        version: options?.version || '1.0',
        language: options?.language || 'golang',
        endorsement_policy_orgs: options?.endorsementPolicyOrgs || [],
      });
    });

    if (result.success) {
      this.logger.info(`Chaincode deployed: ${result.chaincode_id}`);
    } else {
      this.logger.error('Chaincode deployment failed', result.message);
    }

    return {
      success: result.success,
      message: result.message,
      chaincodeId: result.chaincode_id,
    };
  }

  /**
   * Invoke a transaction on the chaincode
   */
  async invoke(
    chaincode: string,
    func: string,
    args: string[],
    options?: InvokeTransactionOptions
  ): Promise<InvokeTransactionResult> {
    this.ensureNetworkId();
    this.logger.info(`Invoking transaction: ${chaincode}.${func}`, { args });

    const result = await this.executeWithRetry(async (client) => {
      return client.invokeTransaction({
        network_id: options?.networkId || this.networkId!,
        chaincode_name: chaincode,
        function_name: func,
        args,
        transient: options?.transient || false,
      });
    });

    if (result.success) {
      this.logger.info(`Transaction successful: ${result.transaction_id}`);
    } else {
      this.logger.error('Transaction failed', result.message);
    }

    return {
      success: result.success,
      message: result.message,
      transactionId: result.transaction_id,
      payload: result.payload ? new Uint8Array(result.payload) : undefined,
    };
  }

  /**
   * Query the ledger (read-only operation)
   */
  async query(
    chaincode: string,
    func: string,
    args: string[],
    options?: QueryLedgerOptions
  ): Promise<QueryLedgerResult> {
    this.ensureNetworkId();
    this.logger.info(`Querying ledger: ${chaincode}.${func}`, { args });

    const result = await this.executeWithRetry(async (client) => {
      return client.queryLedger({
        network_id: options?.networkId || this.networkId!,
        chaincode_name: chaincode,
        function_name: func,
        args,
      });
    });

    if (result.success) {
      this.logger.info('Query successful');
    } else {
      this.logger.error('Query failed', result.message);
    }

    return {
      success: result.success,
      message: result.message,
      payload: result.payload ? new Uint8Array(result.payload) : undefined,
    };
  }

  /**
   * Get the status of a network
   */
  async getNetworkStatus(networkId?: string): Promise<NetworkStatusResult> {
    const id = networkId || this.networkId;
    if (!id) {
      throw new FabricXError(
        'No network ID available. Initialize a network first or provide a network ID.',
        'NO_NETWORK_ID'
      );
    }

    this.logger.info(`Getting network status: ${id}`);

    const result = await this.executeWithRetry(async (client) => {
      return client.getNetworkStatus({ network_id: id });
    });

    this.logger.info('Network status retrieved', {
      running: result.running,
      peers: result.peers.length,
      orderers: result.orderers.length,
    });

    return {
      running: result.running,
      status: result.status,
      peers: result.peers,
      orderers: result.orderers,
    };
  }

  /**
   * Stream logs from network containers
   */
  async streamLogs(
    handler: LogStreamHandler,
    containerName?: string,
    networkId?: string
  ): Promise<() => void> {
    const id = networkId || this.networkId;
    if (!id) {
      throw new FabricXError(
        'No network ID available. Initialize a network first or provide a network ID.',
        'NO_NETWORK_ID'
      );
    }

    this.logger.info(`Streaming logs from network: ${id}`, { containerName });

    const client = await this.getClient();

    return client.streamLogs(
      { network_id: id, container_name: containerName || '' },
      (message) => {
        handler({
          timestamp: message.timestamp,
          container: message.container,
          message: message.message,
        });
      },
      (error) => {
        this.logger.error('Log stream error', error.message);
      },
      () => {
        this.logger.info('Log stream ended');
      }
    );
  }

  /**
   * Stop and optionally cleanup the network
   */
  async stopNetwork(options?: StopNetworkOptions, networkId?: string): Promise<void> {
    const id = networkId || this.networkId;
    if (!id) {
      throw new FabricXError(
        'No network ID available. Initialize a network first or provide a network ID.',
        'NO_NETWORK_ID'
      );
    }

    this.logger.info(`Stopping network: ${id}`, { cleanup: options?.cleanup });

    const result = await this.executeWithRetry(async (client) => {
      return client.stopNetwork({
        network_id: id,
        cleanup: options?.cleanup || false,
      });
    });

    if (result.success) {
      this.logger.info('Network stopped successfully');
      if (this.networkId === id) {
        this.networkId = undefined;
      }
    } else {
      this.logger.error('Failed to stop network', result.message);
      throw new FabricXError(result.message, 'STOP_FAILED');
    }
  }

  /**
   * Get the current network ID
   */
  getNetworkId(): string | undefined {
    return this.networkId;
  }

  /**
   * Set the network ID (useful for connecting to existing networks)
   */
  setNetworkId(networkId: string): void {
    this.networkId = networkId;
    this.logger.info(`Network ID set: ${networkId}`);
  }

  /**
   * Get connection pool statistics (if using connection pool)
   */
  getPoolStats(): PoolStats | null {
    if (!this.pool) {
      return null;
    }
    return this.pool.getStats();
  }

  /**
   * Get connection state
   */
  getConnectionState(): string {
    if (this.client) {
      return this.client.getConnectionState();
    }
    if (this.pool) {
      return this.pool.isClosed() ? 'CLOSED' : 'POOLED';
    }
    return 'NOT_INITIALIZED';
  }

  /**
   * Check if connected
   */
  isConnected(): boolean {
    if (this.client) {
      return this.client.isConnected();
    }
    if (this.pool) {
      return !this.pool.isClosed();
    }
    return false;
  }

  /**
   * Set log level
   */
  setLogLevel(level: LogLevel): void {
    this.logger.setLevel(level);
  }

  /**
   * Get logger instance
   */
  getLogger(): Logger {
    return this.logger;
  }

  /**
   * Close the client connection
   */
  async close(): Promise<void> {
    this.logger.info('Closing FabricX client');

    // Stop connection monitoring
    if (this.connectionWatcher) {
      this.connectionWatcher();
      this.connectionWatcher = undefined;
    }

    if (this.pool) {
      await this.pool.close();
      this.pool = undefined;
    }

    if (this.client) {
      await this.client.close();
      this.client = undefined;
    }

    this.logger.info('FabricX client closed');
  }

  /**
   * Execute operation with connection pool or single client
   */
  private async executeWithRetry<T>(operation: (client: GrpcClient) => Promise<T>): Promise<T> {
    return this.retryManager.execute(async () => {
      if (this.pool) {
        return this.pool.execute(operation);
      } else {
        const client = await this.getClient();
        return operation(client);
      }
    });
  }

  /**
   * Get a client instance
   */
  private async getClient(): Promise<GrpcClient> {
    if (this.client) {
      if (!this.client.isConnected()) {
        await this.client.initialize();
      }
      return this.client;
    }
    throw new FabricXError('No client available', 'NO_CLIENT');
  }

  /**
   * Ensure network ID is set
   */
  private ensureNetworkId(): void {
    if (!this.networkId) {
      throw new FabricXError(
        'No network initialized. Call initNetwork() first or use setNetworkId() to connect to an existing network.',
        'NO_NETWORK_ID'
      );
    }
  }

  /**
   * Setup connection monitoring and auto-reconnect
   */
  private setupConnectionMonitoring(): void {
    if (!this.client) {
      return;
    }

    this.connectionWatcher = this.client.watchConnectionState(async (state) => {
      this.logger.debug(`Connection state changed: ${state}`);

      if (state === 'TRANSIENT_FAILURE' && this.config.autoReconnect) {
        this.logger.warn('Connection lost, attempting to reconnect...');
        try {
          await this.client!.initialize();
          this.logger.info('Reconnected successfully');
        } catch (error) {
          this.logger.error('Reconnection failed', (error as Error).message);
        }
      }
    });
  }
}

// Re-export types for convenience
export * from './types';
export { GrpcClientConfig } from './grpc/client';
export { ConnectionPoolConfig, PoolStats } from './grpc/connection-pool';
export { Logger, LogLevel } from './utils/logger';
export { RetryOptions } from './utils/retry';
