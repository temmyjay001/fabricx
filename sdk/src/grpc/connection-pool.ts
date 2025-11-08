// sdk/src/grpc/connection-pool.ts
import { GrpcClient, GrpcClientConfig } from './client';
import { FabricXError } from '../types';

/**
 * Connection pool configuration
 */
export interface ConnectionPoolConfig extends GrpcClientConfig {
  /** Minimum number of connections to maintain (default: 1) */
  minConnections?: number;
  /** Maximum number of connections allowed (default: 10) */
  maxConnections?: number;
  /** Connection idle timeout in milliseconds (default: 300000) */
  idleTimeout?: number;
  /** Health check interval in milliseconds (default: 30000) */
  healthCheckInterval?: number;
}

/**
 * Connection wrapper with metadata
 */
interface PooledConnection {
  id: string;
  client: GrpcClient;
  createdAt: Date;
  lastUsed: Date;
  inUse: boolean;
  requestCount: number;
}

/**
 * Connection pool statistics
 */
export interface PoolStats {
  totalConnections: number;
  activeConnections: number;
  idleConnections: number;
  totalRequests: number;
  averageRequestsPerConnection: number;
  oldestConnection: Date | null;
  newestConnection: Date | null;
}

/**
 * Connection Pool Manager for FabricX gRPC clients
 * Manages a pool of reusable gRPC connections for optimal performance
 */
export class ConnectionPool {
  private config: Required<ConnectionPoolConfig>;
  private pool: Map<string, PooledConnection>;
  private waitQueue: Array<(client: GrpcClient) => void>;
  private healthCheckTimer?: NodeJS.Timeout;
  private cleanupTimer?: NodeJS.Timeout;
  private closed: boolean = false;

  constructor(config?: ConnectionPoolConfig) {
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
    };

    this.pool = new Map();
    this.waitQueue = [];

    // Validate configuration
    if (this.config.minConnections > this.config.maxConnections) {
      throw new FabricXError(
        'minConnections cannot be greater than maxConnections',
        'INVALID_CONFIG'
      );
    }

    // Initialize pool
    this.initialize();
  }

  /**
   * Initialize the connection pool
   */
  private async initialize(): Promise<void> {
    // Create minimum connections
    const promises = [];
    for (let i = 0; i < this.config.minConnections; i++) {
      promises.push(this.createConnection());
    }

    await Promise.all(promises);

    // Start health check timer
    this.startHealthCheck();

    // Start cleanup timer
    this.startCleanup();
  }

  /**
   * Acquire a connection from the pool
   */
  async acquire(): Promise<GrpcClient> {
    if (this.closed) {
      throw new FabricXError('Connection pool is closed', 'POOL_CLOSED');
    }

    // Try to find an idle connection
    for (const [id, conn] of this.pool.entries()) {
      if (!conn.inUse) {
        conn.inUse = true;
        conn.lastUsed = new Date();
        conn.requestCount++;
        return conn.client;
      }
    }

    // If pool is not at max capacity, create a new connection
    if (this.pool.size < this.config.maxConnections) {
      const id = await this.createConnection();
      const conn = this.pool.get(id)!;
      conn.inUse = true;
      conn.lastUsed = new Date();
      conn.requestCount++;
      return conn.client;
    }

    // Wait for a connection to become available
    return new Promise((resolve) => {
      this.waitQueue.push(resolve);
    });
  }

  /**
   * Release a connection back to the pool
   */
  release(client: GrpcClient): void {
    // Find the connection in the pool
    for (const [id, conn] of this.pool.entries()) {
      if (conn.client === client) {
        conn.inUse = false;
        conn.lastUsed = new Date();

        // If there are waiting requests, give it to them
        if (this.waitQueue.length > 0) {
          const waitingCallback = this.waitQueue.shift()!;
          conn.inUse = true;
          conn.requestCount++;
          waitingCallback(client);
        }

        return;
      }
    }

    throw new FabricXError('Connection not found in pool', 'INVALID_CONNECTION');
  }

  /**
   * Execute a function with a pooled connection
   */
  async execute<T>(fn: (client: GrpcClient) => Promise<T>): Promise<T> {
    const client = await this.acquire();
    try {
      return await fn(client);
    } finally {
      this.release(client);
    }
  }

  /**
   * Create a new connection
   */
  private async createConnection(): Promise<string> {
    const id = this.generateConnectionId();

    const client = new GrpcClient({
      serverAddr: this.config.serverAddr,
      timeout: this.config.timeout,
      useTls: this.config.useTls,
      tlsConfig: this.config.tlsConfig,
      maxRetries: this.config.maxRetries,
      retryDelay: this.config.retryDelay,
      keepAliveTime: this.config.keepAliveTime,
      keepAliveTimeout: this.config.keepAliveTimeout,
    });

    await client.initialize();

    const conn: PooledConnection = {
      id,
      client,
      createdAt: new Date(),
      lastUsed: new Date(),
      inUse: false,
      requestCount: 0,
    };

    this.pool.set(id, conn);
    return id;
  }

  /**
   * Remove a connection from the pool
   */
  private async removeConnection(id: string): Promise<void> {
    const conn = this.pool.get(id);
    if (!conn) {
      return;
    }

    // Don't remove if in use
    if (conn.inUse) {
      return;
    }

    await conn.client.close();
    this.pool.delete(id);
  }

  /**
   * Start health check timer
   */
  private startHealthCheck(): void {
    this.healthCheckTimer = setInterval(async () => {
      if (this.closed) {
        return;
      }

      // Check each connection
      for (const [id, conn] of this.pool.entries()) {
        if (conn.inUse) {
          continue;
        }

        try {
          const state = conn.client.getConnectionState();
          if (state !== 'READY' && state !== 'IDLE') {
            // Connection is not healthy, remove it
            await this.removeConnection(id);

            // Create a new connection if below minimum
            if (this.pool.size < this.config.minConnections) {
              await this.createConnection();
            }
          }
        } catch (error) {
          // Error checking connection, remove it
          await this.removeConnection(id);

          // Create a new connection if below minimum
          if (this.pool.size < this.config.minConnections) {
            await this.createConnection();
          }
        }
      }
    }, this.config.healthCheckInterval);
  }

  /**
   * Start cleanup timer
   */
  private startCleanup(): void {
    this.cleanupTimer = setInterval(async () => {
      if (this.closed) {
        return;
      }

      const now = new Date();

      // Remove idle connections that have exceeded the timeout
      for (const [id, conn] of this.pool.entries()) {
        if (conn.inUse) {
          continue;
        }

        const idleTime = now.getTime() - conn.lastUsed.getTime();

        if (idleTime > this.config.idleTimeout) {
          // Only remove if above minimum connections
          if (this.pool.size > this.config.minConnections) {
            await this.removeConnection(id);
          }
        }
      }
    }, 60000); // Run cleanup every minute
  }

  /**
   * Get pool statistics
   */
  getStats(): PoolStats {
    let activeConnections = 0;
    let totalRequests = 0;
    let oldest: Date | null = null;
    let newest: Date | null = null;

    for (const conn of this.pool.values()) {
      if (conn.inUse) {
        activeConnections++;
      }

      totalRequests += conn.requestCount;

      if (!oldest || conn.createdAt < oldest) {
        oldest = conn.createdAt;
      }

      if (!newest || conn.createdAt > newest) {
        newest = conn.createdAt;
      }
    }

    return {
      totalConnections: this.pool.size,
      activeConnections,
      idleConnections: this.pool.size - activeConnections,
      totalRequests,
      averageRequestsPerConnection: this.pool.size > 0 ? totalRequests / this.pool.size : 0,
      oldestConnection: oldest,
      newestConnection: newest,
    };
  }

  /**
   * Drain the pool (wait for all connections to become idle)
   */
  async drain(timeoutMs: number = 30000): Promise<void> {
    const startTime = Date.now();

    while (this.hasActiveConnections()) {
      if (Date.now() - startTime > timeoutMs) {
        throw new FabricXError('Drain timeout exceeded', 'DRAIN_TIMEOUT', {
          activeConnections: this.getStats().activeConnections,
        });
      }

      await this.sleep(100);
    }
  }

  /**
   * Check if there are active connections
   */
  private hasActiveConnections(): boolean {
    for (const conn of this.pool.values()) {
      if (conn.inUse) {
        return true;
      }
    }
    return false;
  }

  /**
   * Close the connection pool
   */
  async close(): Promise<void> {
    if (this.closed) {
      return;
    }

    this.closed = true;

    // Stop timers
    if (this.healthCheckTimer) {
      clearInterval(this.healthCheckTimer);
    }

    if (this.cleanupTimer) {
      clearInterval(this.cleanupTimer);
    }

    // Wait for active connections to finish (with timeout)
    try {
      await this.drain(30000);
    } catch (error) {
      // Log but don't throw - we're closing anyway
      console.warn('Some connections did not drain gracefully');
    }

    // Close all connections
    const closePromises: Promise<void>[] = [];
    for (const conn of this.pool.values()) {
      closePromises.push(conn.client.close());
    }

    await Promise.all(closePromises);
    this.pool.clear();

    // Clear wait queue
    this.waitQueue = [];
  }

  /**
   * Check if pool is closed
   */
  isClosed(): boolean {
    return this.closed;
  }

  /**
   * Generate unique connection ID
   */
  private generateConnectionId(): string {
    return `conn-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Sleep for specified milliseconds
   */
  private sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  /**
   * Get the underlying pool (for debugging)
   */
  getPool(): Map<string, PooledConnection> {
    return this.pool;
  }

  /**
   * Force resize the pool
   */
  async resize(newMin: number, newMax: number): Promise<void> {
    if (newMin > newMax) {
      throw new FabricXError(
        'minConnections cannot be greater than maxConnections',
        'INVALID_CONFIG'
      );
    }

    this.config.minConnections = newMin;
    this.config.maxConnections = newMax;

    // Add connections if below minimum
    while (this.pool.size < newMin) {
      await this.createConnection();
    }

    // Remove excess idle connections if above maximum
    if (this.pool.size > newMax) {
      let removed = 0;
      for (const [id, conn] of this.pool.entries()) {
        if (conn.inUse) {
          continue;
        }

        await this.removeConnection(id);
        removed++;

        if (this.pool.size <= newMax) {
          break;
        }
      }
    }
  }
}
