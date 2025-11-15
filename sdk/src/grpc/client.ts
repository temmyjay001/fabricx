// sdk/src/grpc/client.ts
import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import { promisify } from 'util';
import path from 'path';

import { FabricXError } from '../types';

/**
 * gRPC client configuration
 */
export interface GrpcClientConfig {
  /** Server address (default: "localhost:50051") */
  serverAddr?: string;
  /** Request timeout in milliseconds (default: 120000) */
  timeout?: number;
  /** Whether to use TLS (default: false) */
  useTls?: boolean;
  /** TLS credentials configuration */
  tlsConfig?: TlsConfig;
  /** Maximum retry attempts (default: 3) */
  maxRetries?: number;
  /** Retry delay in milliseconds (default: 1000) */
  retryDelay?: number;
  /** Keep alive time in milliseconds (default: 30000) */
  keepAliveTime?: number;
  /** Keep alive timeout in milliseconds (default: 10000) */
  keepAliveTimeout?: number;
}

/**
 * TLS configuration
 */
export interface TlsConfig {
  /** Root CA certificate */
  rootCert?: Buffer;
  /** Client certificate */
  clientCert?: Buffer;
  /** Client private key */
  clientKey?: Buffer;
  /** Server name override */
  serverNameOverride?: string;
}

/**
 * Internal gRPC request/response types
 */
interface InitNetworkRequest {
  network_name: string;
  num_orgs: number;
  channel_name: string;
  config: { [key: string]: string };
}

interface InitNetworkResponse {
  success: boolean;
  message: string;
  network_id: string;
  endpoints: string[];
}

interface DeployChaincodeRequest {
  network_id: string;
  chaincode_name: string;
  chaincode_path: string;
  version: string;
  language: string;
  endorsement_policy_orgs: string[];
}

interface DeployChaincodeResponse {
  success: boolean;
  message: string;
  chaincode_id: string;
}

interface InvokeTransactionRequest {
  network_id: string;
  chaincode_name: string;
  function_name: string;
  args: string[];
  transient: boolean;
}

interface InvokeTransactionResponse {
  success: boolean;
  message: string;
  transaction_id: string;
  payload: Buffer;
}

interface QueryLedgerRequest {
  network_id: string;
  chaincode_name: string;
  function_name: string;
  args: string[];
}

interface QueryLedgerResponse {
  success: boolean;
  message: string;
  payload: Buffer;
}

interface NetworkStatusRequest {
  network_id: string;
}

interface NetworkStatusResponse {
  running: boolean;
  status: string;
  peers: Array<{
    name: string;
    org: string;
    status: string;
    endpoint: string;
  }>;
  orderers: Array<{
    name: string;
    status: string;
    endpoint: string;
  }>;
}

interface StopNetworkRequest {
  network_id: string;
  cleanup: boolean;
}

interface StopNetworkResponse {
  success: boolean;
  message: string;
}

interface StreamLogsRequest {
  network_id: string;
  container_name: string;
}

interface LogMessage {
  timestamp: string;
  container: string;
  message: string;
}

/**
 * Production-grade gRPC Client for FabricX
 */
export class GrpcClient {
  private client: any;
  private credentials: grpc.ChannelCredentials;
  private options: grpc.ChannelOptions;
  private config: Required<GrpcClientConfig>;
  private connected: boolean = false;

  constructor(config?: GrpcClientConfig) {
    this.config = {
      serverAddr: config?.serverAddr || 'localhost:50051',
      timeout: config?.timeout || 120000,
      useTls: config?.useTls || false,
      tlsConfig: config?.tlsConfig || {},
      maxRetries: config?.maxRetries || 3,
      retryDelay: config?.retryDelay || 1000,
      keepAliveTime: config?.keepAliveTime || 30000,
      keepAliveTimeout: config?.keepAliveTimeout || 10000,
    };

    // Setup credentials
    this.credentials = this.createCredentials();

    // Setup channel options
    this.options = {
      'grpc.keepalive_time_ms': this.config.keepAliveTime,
      'grpc.keepalive_timeout_ms': this.config.keepAliveTimeout,
      'grpc.keepalive_permit_without_calls': 1,
      'grpc.http2.max_pings_without_data': 0,
      'grpc.http2.min_time_between_pings_ms': 10000,
      'grpc.http2.min_ping_interval_without_data_ms': 5000,
      'grpc.max_receive_message_length': 100 * 1024 * 1024, // 100MB
      'grpc.max_send_message_length': 100 * 1024 * 1024, // 100MB
    };

    this.client = null;
  }

  /**
   * Initialize the gRPC client
   */
  async initialize(): Promise<void> {
    if (this.connected) {
      return;
    }

    try {
      // Load proto file
      const protoPath = path.join(__dirname, 'protos/fabricx.proto');
      const packageDefinition = await protoLoader.load(protoPath, {
        keepCase: true,
        longs: String,
        enums: String,
        defaults: true,
        oneofs: true,
      });

      const protoDescriptor = grpc.loadPackageDefinition(packageDefinition) as any;
      const FabricXService = protoDescriptor.fabricx.FabricXService;

      // Create client
      this.client = new FabricXService(this.config.serverAddr, this.credentials, this.options);

      // Wait for connection
      await this.waitForReady();
      this.connected = true;
    } catch (error) {
      throw new FabricXError(
        `Failed to initialize gRPC client: ${(error as Error).message}`,
        'INITIALIZATION_ERROR',
        { error, serverAddr: this.config.serverAddr }
      );
    }
  }

  /**
   * Wait for gRPC client to be ready
   */
  private waitForReady(): Promise<void> {
    return new Promise((resolve, reject) => {
      const deadline = new Date();
      deadline.setSeconds(deadline.getSeconds() + 10);

      this.client.waitForReady(deadline, (error: Error | undefined) => {
        if (error) {
          reject(
            new FabricXError('Failed to connect to FabricX runtime', 'CONNECTION_ERROR', {
              error,
              serverAddr: this.config.serverAddr,
            })
          );
        } else {
          resolve();
        }
      });
    });
  }

  /**
   * Create gRPC credentials
   */
  private createCredentials(): grpc.ChannelCredentials {
    if (!this.config.useTls) {
      return grpc.credentials.createInsecure();
    }

    const { tlsConfig } = this.config;

    if (tlsConfig.rootCert && tlsConfig.clientCert && tlsConfig.clientKey) {
      // Mutual TLS
      return grpc.credentials.createSsl(
        tlsConfig.rootCert,
        tlsConfig.clientKey,
        tlsConfig.clientCert
      );
    } else if (tlsConfig.rootCert) {
      // Server TLS only
      return grpc.credentials.createSsl(tlsConfig.rootCert);
    } else {
      // Default TLS
      return grpc.credentials.createSsl();
    }
  }

  /**
   * Initialize a network
   */
  async initNetwork(request: InitNetworkRequest): Promise<InitNetworkResponse> {
    await this.ensureConnected();
    return this.makeUnaryCall<InitNetworkRequest, InitNetworkResponse>('InitNetwork', request);
  }

  /**
   * Deploy chaincode
   */
  async deployChaincode(request: DeployChaincodeRequest): Promise<DeployChaincodeResponse> {
    await this.ensureConnected();
    return this.makeUnaryCall<DeployChaincodeRequest, DeployChaincodeResponse>(
      'DeployChaincode',
      request
    );
  }

  /**
   * Invoke a transaction
   */
  async invokeTransaction(request: InvokeTransactionRequest): Promise<InvokeTransactionResponse> {
    await this.ensureConnected();
    return this.makeUnaryCall<InvokeTransactionRequest, InvokeTransactionResponse>(
      'InvokeTransaction',
      request
    );
  }

  /**
   * Query the ledger
   */
  async queryLedger(request: QueryLedgerRequest): Promise<QueryLedgerResponse> {
    await this.ensureConnected();
    return this.makeUnaryCall<QueryLedgerRequest, QueryLedgerResponse>('QueryLedger', request);
  }

  /**
   * Get network status
   */
  async getNetworkStatus(request: NetworkStatusRequest): Promise<NetworkStatusResponse> {
    await this.ensureConnected();
    return this.makeUnaryCall<NetworkStatusRequest, NetworkStatusResponse>(
      'GetNetworkStatus',
      request
    );
  }

  /**
   * Stop a network
   */
  async stopNetwork(request: StopNetworkRequest): Promise<StopNetworkResponse> {
    await this.ensureConnected();
    return this.makeUnaryCall<StopNetworkRequest, StopNetworkResponse>('StopNetwork', request);
  }

  /**
   * Stream logs from containers
   */
  async streamLogs(
    request: StreamLogsRequest,
    onData: (message: LogMessage) => void,
    onError?: (error: Error) => void,
    onEnd?: () => void
  ): Promise<() => void> {
    await this.ensureConnected();

    return new Promise((resolve, reject) => {
      const call = this.client.StreamLogs(request);

      call.on('data', (message: LogMessage) => {
        onData(message);
      });

      call.on('error', (error: grpc.ServiceError) => {
        const fabricxError = this.convertGrpcError(error, 'StreamLogs');
        if (onError) {
          onError(fabricxError);
        } else {
          reject(fabricxError);
        }
      });

      call.on('end', () => {
        if (onEnd) {
          onEnd();
        }
      });

      // Return cancel function
      resolve(() => call.cancel());
    });
  }

  /**
   * Close the client connection
   */
  async close(): Promise<void> {
    if (this.client) {
      this.client.close();
      this.connected = false;
      this.client = null;
    }
  }

  /**
   * Check if client is connected
   */
  isConnected(): boolean {
    return this.connected;
  }

  /**
   * Ensure client is connected
   */
  private async ensureConnected(): Promise<void> {
    if (!this.connected) {
      await this.initialize();
    }
  }

  /**
   * Make a unary gRPC call with retry logic
   */
  private async makeUnaryCall<TReq, TRes>(
    method: string,
    request: TReq,
    attempt: number = 1
  ): Promise<TRes> {
    const deadline = new Date();
    deadline.setMilliseconds(deadline.getMilliseconds() + this.config.timeout);

    const metadata = new grpc.Metadata();
    metadata.set('request-id', this.generateRequestId());

    const callOptions: grpc.CallOptions = {
      deadline,
    };

    try {
      const promisifiedCall = promisify(this.client[method]).bind(this.client);
      const response = await promisifiedCall(request, metadata, callOptions);
      return response as TRes;
    } catch (error) {
      const grpcError = error as grpc.ServiceError;

      // Check if we should retry
      if (this.shouldRetry(grpcError) && attempt < this.config.maxRetries) {
        // Wait before retry
        await this.sleep(this.config.retryDelay * attempt);

        // Retry with exponential backoff
        return this.makeUnaryCall<TReq, TRes>(method, request, attempt + 1);
      }

      // Convert and throw error
      throw this.convertGrpcError(grpcError, method);
    }
  }

  /**
   * Check if error is retryable
   */
  private shouldRetry(error: grpc.ServiceError): boolean {
    const retryableCodes = [
      grpc.status.UNAVAILABLE,
      grpc.status.DEADLINE_EXCEEDED,
      grpc.status.RESOURCE_EXHAUSTED,
      grpc.status.ABORTED,
    ];

    return retryableCodes.includes(error.code);
  }

  /**
   * Convert gRPC error to FabricXError
   */
  private convertGrpcError(error: grpc.ServiceError, method: string): FabricXError {
    let code: string;
    let message: string;

    switch (error.code) {
      case grpc.status.UNAVAILABLE:
        code = 'CONNECTION_ERROR';
        message = `FabricX runtime is unavailable at ${this.config.serverAddr}`;
        break;
      case grpc.status.DEADLINE_EXCEEDED:
        code = 'TIMEOUT';
        message = `Request timeout after ${this.config.timeout}ms`;
        break;
      case grpc.status.UNAUTHENTICATED:
        code = 'AUTHENTICATION_ERROR';
        message = 'Authentication failed';
        break;
      case grpc.status.PERMISSION_DENIED:
        code = 'PERMISSION_DENIED';
        message = 'Permission denied';
        break;
      case grpc.status.INVALID_ARGUMENT:
        code = 'INVALID_ARGUMENT';
        message = 'Invalid request parameters';
        break;
      case grpc.status.NOT_FOUND:
        code = 'NOT_FOUND';
        message = 'Resource not found';
        break;
      case grpc.status.ALREADY_EXISTS:
        code = 'ALREADY_EXISTS';
        message = 'Resource already exists';
        break;
      case grpc.status.INTERNAL:
        code = 'INTERNAL_ERROR';
        message = 'Internal server error';
        break;
      default:
        code = 'GRPC_ERROR';
        message = error.message || 'Unknown gRPC error';
    }

    return new FabricXError(message, code, {
      method,
      grpcCode: error.code,
      grpcMessage: error.message,
      details: error.details,
      metadata: error.metadata?.getMap(),
    });
  }

  /**
   * Generate unique request ID
   */
  private generateRequestId(): string {
    return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Sleep for specified milliseconds
   */
  private sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  /**
   * Get connection state
   */
  getConnectionState(): string {
    if (!this.client) {
      return 'NOT_INITIALIZED';
    }

    const state = this.client.getChannel().getConnectivityState(false);

    switch (state) {
      case grpc.connectivityState.IDLE:
        return 'IDLE';
      case grpc.connectivityState.CONNECTING:
        return 'CONNECTING';
      case grpc.connectivityState.READY:
        return 'READY';
      case grpc.connectivityState.TRANSIENT_FAILURE:
        return 'TRANSIENT_FAILURE';
      case grpc.connectivityState.SHUTDOWN:
        return 'SHUTDOWN';
      default:
        return 'UNKNOWN';
    }
  }

  /**
   * Watch connection state changes
   */
  watchConnectionState(callback: (state: string) => void): () => void {
    if (!this.client) {
      throw new FabricXError('Client not initialized', 'NOT_INITIALIZED');
    }

    const channel = this.client.getChannel();
    let watching = true;

    const watch = () => {
      if (!watching) return;

      const currentState = channel.getConnectivityState(true);
      callback(this.getConnectionState());

      const deadline = new Date();
      deadline.setSeconds(deadline.getSeconds() + 5);

      channel.watchConnectivityState(currentState, deadline, (error: Error | undefined) => {
        if (!error && watching) {
          watch();
        }
      });
    };

    watch();

    // Return stop function
    return () => {
      watching = false;
    };
  }
}
