// sdk/src/__tests__/fabricx.test.ts
import { FabricX, LogLevel } from '../fabricx';
import { FabricXError } from '../types';
import { GrpcClient } from '../grpc/client';

// Mock the gRPC client
jest.mock('../grpc/client');

describe('FabricX SDK', () => {
  let fabricx: FabricX;
  let mockClient: jest.Mocked<GrpcClient>;

  beforeEach(() => {
    // Clear all mocks
    jest.clearAllMocks();

    // Create mock client
    mockClient = {
      initialize: jest.fn().mockResolvedValue(undefined),
      initNetwork: jest.fn(),
      deployChaincode: jest.fn(),
      invokeTransaction: jest.fn(),
      queryLedger: jest.fn(),
      getNetworkStatus: jest.fn(),
      stopNetwork: jest.fn(),
      streamLogs: jest.fn(),
      close: jest.fn().mockResolvedValue(undefined),
      isConnected: jest.fn().mockReturnValue(true),
      getConnectionState: jest.fn().mockReturnValue('READY'),
      watchConnectionState: jest.fn(),
    } as any;

    // Mock the GrpcClient constructor
    (GrpcClient as jest.MockedClass<typeof GrpcClient>).mockImplementation(() => mockClient);

    // Create FabricX instance
    fabricx = new FabricX({
      serverAddr: 'localhost:50051',
      useConnectionPool: false,
      logger: { level: 'silent' as LogLevel },
    });
  });

  afterEach(async () => {
    await fabricx.close();
  });

  describe('initNetwork', () => {
    it('should initialize a network successfully', async () => {
      const mockResponse = {
        success: true,
        message: 'Network initialized',
        network_id: 'test-network-123',
        endpoints: ['localhost:7051', 'localhost:8051'],
      };

      mockClient.initNetwork.mockResolvedValue(mockResponse);

      const result = await fabricx.initNetwork({
        name: 'test-network',
        numOrgs: 2,
        channelName: 'mychannel',
      });

      expect(result.success).toBe(true);
      expect(result.networkId).toBe('test-network-123');
      expect(result.endpoints).toHaveLength(2);
      expect(mockClient.initNetwork).toHaveBeenCalledWith({
        network_name: 'test-network',
        num_orgs: 2,
        channel_name: 'mychannel',
        config: {},
      });
    });

    it('should use default values when options not provided', async () => {
      const mockResponse = {
        success: true,
        message: 'Network initialized',
        network_id: 'test-network-123',
        endpoints: ['localhost:7051'],
      };

      mockClient.initNetwork.mockResolvedValue(mockResponse);

      await fabricx.initNetwork();

      expect(mockClient.initNetwork).toHaveBeenCalledWith({
        network_name: 'fabricx-network',
        num_orgs: 2,
        channel_name: 'mychannel',
        config: {},
      });
    });

    it('should handle initialization failure', async () => {
      const mockResponse = {
        success: false,
        message: 'Failed to initialize network',
        network_id: '',
        endpoints: [],
      };

      mockClient.initNetwork.mockResolvedValue(mockResponse);

      const result = await fabricx.initNetwork();

      expect(result.success).toBe(false);
      expect(result.message).toBe('Failed to initialize network');
    });
  });

  describe('deployChaincode', () => {
    beforeEach(async () => {
      // Initialize network first
      mockClient.initNetwork.mockResolvedValue({
        success: true,
        message: 'Network initialized',
        network_id: 'test-network-123',
        endpoints: ['localhost:7051'],
      });
      await fabricx.initNetwork();
    });

    it('should deploy chaincode successfully', async () => {
      const mockResponse = {
        success: true,
        message: 'Chaincode deployed',
        chaincode_id: 'mycc-abc123',
      };

      mockClient.deployChaincode.mockResolvedValue(mockResponse);

      const result = await fabricx.deployChaincode('mycc', {
        path: './chaincode/mycc',
        version: '1.0',
        language: 'golang',
      });

      expect(result.success).toBe(true);
      expect(result.chaincodeId).toBe('mycc-abc123');
      expect(mockClient.deployChaincode).toHaveBeenCalledWith({
        network_id: 'test-network-123',
        chaincode_name: 'mycc',
        chaincode_path: './chaincode/mycc',
        version: '1.0',
        language: 'golang',
        endorsement_policy_orgs: [],
      });
    });

    it('should throw error if network not initialized', async () => {
      const uninitializedFabricx = new FabricX({
        useConnectionPool: false,
        logger: { level: 'silent' as LogLevel },
      });

      await expect(uninitializedFabricx.deployChaincode('mycc')).rejects.toThrow(FabricXError);
      await expect(uninitializedFabricx.deployChaincode('mycc')).rejects.toThrow(
        'No network initialized'
      );
    });
  });

  describe('invoke', () => {
    beforeEach(async () => {
      mockClient.initNetwork.mockResolvedValue({
        success: true,
        message: 'Network initialized',
        network_id: 'test-network-123',
        endpoints: ['localhost:7051'],
      });
      await fabricx.initNetwork();
    });

    it('should invoke transaction successfully', async () => {
      const mockResponse = {
        success: true,
        message: 'Transaction invoked',
        transaction_id: 'tx-123',
        payload: Buffer.from(JSON.stringify({ result: 'success' })),
      };

      mockClient.invokeTransaction.mockResolvedValue(mockResponse);

      const result = await fabricx.invoke('mycc', 'createAsset', ['asset1', 'value1']);

      expect(result.success).toBe(true);
      expect(result.transactionId).toBe('tx-123');
      expect(result.payload).toBeDefined();
      expect(mockClient.invokeTransaction).toHaveBeenCalledWith({
        network_id: 'test-network-123',
        chaincode_name: 'mycc',
        function_name: 'createAsset',
        args: ['asset1', 'value1'],
        transient: false,
      });
    });

    it('should handle transient data option', async () => {
      const mockResponse = {
        success: true,
        message: 'Transaction invoked',
        transaction_id: 'tx-123',
        payload: Buffer.from(''),
      };

      mockClient.invokeTransaction.mockResolvedValue(mockResponse);

      await fabricx.invoke('mycc', 'privateFunc', ['arg1'], { transient: true });

      expect(mockClient.invokeTransaction).toHaveBeenCalledWith(
        expect.objectContaining({
          transient: true,
        })
      );
    });
  });

  describe('query', () => {
    beforeEach(async () => {
      mockClient.initNetwork.mockResolvedValue({
        success: true,
        message: 'Network initialized',
        network_id: 'test-network-123',
        endpoints: ['localhost:7051'],
      });
      await fabricx.initNetwork();
    });

    it('should query ledger successfully', async () => {
      const mockResponse = {
        success: true,
        message: 'Query successful',
        payload: Buffer.from(JSON.stringify({ id: 'asset1', value: 'value1' })),
      };

      mockClient.queryLedger.mockResolvedValue(mockResponse);

      const result = await fabricx.query('mycc', 'getAsset', ['asset1']);

      expect(result.success).toBe(true);
      expect(result.payload).toBeDefined();
      expect(mockClient.queryLedger).toHaveBeenCalledWith({
        network_id: 'test-network-123',
        chaincode_name: 'mycc',
        function_name: 'getAsset',
        args: ['asset1'],
      });
    });
  });

  describe('getNetworkStatus', () => {
    it('should get network status successfully', async () => {
      fabricx.setNetworkId('test-network-123');

      const mockResponse = {
        running: true,
        status: 'Running',
        peers: [{ name: 'peer0.org1', org: 'Org1', status: 'running', endpoint: 'localhost:7051' }],
        orderers: [{ name: 'orderer', status: 'running', endpoint: 'localhost:7050' }],
      };

      mockClient.getNetworkStatus.mockResolvedValue(mockResponse);

      const result = await fabricx.getNetworkStatus();

      expect(result.running).toBe(true);
      expect(result.peers).toHaveLength(1);
      expect(result.orderers).toHaveLength(1);
    });

    it('should throw error if network ID not set', async () => {
      await expect(fabricx.getNetworkStatus()).rejects.toThrow(FabricXError);
      await expect(fabricx.getNetworkStatus()).rejects.toThrow('No network ID available');
    });
  });

  describe('stopNetwork', () => {
    it('should stop network successfully', async () => {
      fabricx.setNetworkId('test-network-123');

      const mockResponse = {
        success: true,
        message: 'Network stopped',
      };

      mockClient.stopNetwork.mockResolvedValue(mockResponse);

      await fabricx.stopNetwork({ cleanup: true });

      expect(mockClient.stopNetwork).toHaveBeenCalledWith({
        network_id: 'test-network-123',
        cleanup: true,
      });
      expect(fabricx.getNetworkId()).toBeUndefined();
    });

    it('should throw error on stop failure', async () => {
      fabricx.setNetworkId('test-network-123');

      const mockResponse = {
        success: false,
        message: 'Failed to stop network',
      };

      mockClient.stopNetwork.mockResolvedValue(mockResponse);

      await expect(fabricx.stopNetwork()).rejects.toThrow(FabricXError);
      await expect(fabricx.stopNetwork()).rejects.toThrow('Failed to stop network');
    });
  });

  describe('Network ID management', () => {
    it('should get and set network ID', () => {
      expect(fabricx.getNetworkId()).toBeUndefined();

      fabricx.setNetworkId('test-network-123');
      expect(fabricx.getNetworkId()).toBe('test-network-123');
    });

    it('should set network ID after initialization', async () => {
      mockClient.initNetwork.mockResolvedValue({
        success: true,
        message: 'Network initialized',
        network_id: 'auto-network-456',
        endpoints: ['localhost:7051'],
      });

      await fabricx.initNetwork();
      expect(fabricx.getNetworkId()).toBe('auto-network-456');
    });
  });

  describe('Connection state', () => {
    it('should get connection state', () => {
      const state = fabricx.getConnectionState();
      expect(state).toBe('READY');
    });

    it('should check if connected', () => {
      const isConnected = fabricx.isConnected();
      expect(isConnected).toBe(true);
    });
  });

  describe('Close', () => {
    it('should close connection properly', async () => {
      await fabricx.close();
      expect(mockClient.close).toHaveBeenCalled();
    });
  });
});
