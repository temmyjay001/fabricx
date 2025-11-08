// sdk/src/types.ts

/**
 * Options for initializing a Fabric network
 */
export interface InitNetworkOptions {
  /** Network name (default: "fabricx-network") */
  name?: string;
  /** Number of organizations (default: 2) */
  numOrgs?: number;
  /** Channel name (default: "mychannel") */
  channelName?: string;
  /** Custom configuration parameters */
  config?: Record<string, string>;
}

/**
 * Result of network initialization
 */
export interface InitNetworkResult {
  /** Whether the operation succeeded */
  success: boolean;
  /** Status message */
  message: string;
  /** Unique network identifier */
  networkId: string;
  /** List of peer endpoints */
  endpoints: string[];
}

/**
 * Options for deploying chaincode
 */
export interface DeployChaincodeOptions {
  /** Path to chaincode directory */
  path?: string;
  /** Chaincode version (default: "1.0") */
  version?: string;
  /** Chaincode language: "golang", "node", "java" (default: "golang") */
  language?: string;
  /** Organizations required for endorsement */
  endorsementPolicyOrgs?: string[];
}

/**
 * Result of chaincode deployment
 */
export interface DeployChaincodeResult {
  /** Whether the operation succeeded */
  success: boolean;
  /** Status message */
  message: string;
  /** Unique chaincode identifier */
  chaincodeId: string;
}

/**
 * Options for invoking transactions
 */
export interface InvokeTransactionOptions {
  /** Network ID (uses current network if not provided) */
  networkId?: string;
  /** Whether to use transient data */
  transient?: boolean;
}

/**
 * Result of transaction invocation
 */
export interface InvokeTransactionResult {
  /** Whether the operation succeeded */
  success: boolean;
  /** Status message */
  message: string;
  /** Transaction ID */
  transactionId: string;
  /** Transaction response payload */
  payload?: Uint8Array;
}

/**
 * Options for querying the ledger
 */
export interface QueryLedgerOptions {
  /** Network ID (uses current network if not provided) */
  networkId?: string;
}

/**
 * Result of ledger query
 */
export interface QueryLedgerResult {
  /** Whether the operation succeeded */
  success: boolean;
  /** Status message */
  message: string;
  /** Query result payload */
  payload?: Uint8Array;
}

/**
 * Peer status information
 */
export interface PeerStatus {
  /** Peer name */
  name: string;
  /** Organization name */
  org: string;
  /** Current status */
  status: string;
  /** Peer endpoint */
  endpoint: string;
}

/**
 * Orderer status information
 */
export interface OrdererStatus {
  /** Orderer name */
  name: string;
  /** Current status */
  status: string;
  /** Orderer endpoint */
  endpoint: string;
}

/**
 * Network status result
 */
export interface NetworkStatusResult {
  /** Whether the network is running */
  running: boolean;
  /** Overall status description */
  status: string;
  /** List of peer statuses */
  peers: PeerStatus[];
  /** List of orderer statuses */
  orderers: OrdererStatus[];
}

/**
 * Options for stopping a network
 */
export interface StopNetworkOptions {
  /** Whether to cleanup containers and volumes */
  cleanup?: boolean;
}

/**
 * Log message from container
 */
export interface LogMessage {
  /** Timestamp of the log */
  timestamp: string;
  /** Container name */
  container: string;
  /** Log message content */
  message: string;
}

/**
 * Callback handler for log streaming
 */
export type LogStreamHandler = (log: LogMessage) => void;

/**
 * Error thrown by FabricX SDK
 */
export class FabricXError extends Error {
  constructor(
    message: string,
    public readonly code: string,
    public readonly details?: any
  ) {
    super(message);
    this.name = "FabricXError";
  }
}

/**
 * Network configuration for advanced use cases
 */
export interface NetworkConfig {
  /** Network ID */
  id: string;
  /** Network name */
  name: string;
  /** Channel name */
  channel: string;
  /** List of organizations */
  organizations: OrganizationConfig[];
  /** List of orderers */
  orderers: OrdererConfig[];
}

/**
 * Organization configuration
 */
export interface OrganizationConfig {
  /** Organization name */
  name: string;
  /** MSP ID */
  mspId: string;
  /** Organization domain */
  domain: string;
  /** List of peers */
  peers: PeerConfig[];
}

/**
 * Peer configuration
 */
export interface PeerConfig {
  /** Peer name */
  name: string;
  /** Peer port */
  port: number;
  /** Whether CouchDB is enabled */
  couchdb?: boolean;
}

/**
 * Orderer configuration
 */
export interface OrdererConfig {
  /** Orderer name */
  name: string;
  /** Orderer port */
  port: number;
  /** Orderer domain */
  domain: string;
}

/**
 * Transaction proposal
 */
export interface TransactionProposal {
  /** Chaincode name */
  chaincode: string;
  /** Function name */
  function: string;
  /** Function arguments */
  args: string[];
  /** Transient data (optional) */
  transient?: Record<string, Uint8Array>;
}

/**
 * Query proposal
 */
export interface QueryProposal {
  /** Chaincode name */
  chaincode: string;
  /** Function name */
  function: string;
  /** Function arguments */
  args: string[];
}

/**
 * Event listener callback
 */
export type EventCallback = (event: BlockEvent | TransactionEvent) => void;

/**
 * Block event
 */
export interface BlockEvent {
  /** Event type */
  type: "block";
  /** Block number */
  blockNumber: number;
  /** Block hash */
  blockHash: string;
  /** Number of transactions in block */
  transactionCount: number;
}

/**
 * Transaction event
 */
export interface TransactionEvent {
  /** Event type */
  type: "transaction";
  /** Transaction ID */
  transactionId: string;
  /** Block number */
  blockNumber: number;
  /** Transaction status */
  status: "VALID" | "INVALID";
}
