// examples/basic-usage.ts
/**
 * FabricX SDK - Basic Usage Example
 *
 * This example demonstrates the core functionality of the FabricX SDK:
 * 1. Initializing a Fabric network
 * 2. Deploying chaincode
 * 3. Invoking transactions
 * 4. Querying the ledger
 * 5. Getting network status
 * 6. Stopping the network
 */

import { FabricX, FabricXError } from '../src/index';

async function main() {
  console.log('🚀 FabricX SDK Example\n');

  // Create a new FabricX instance
  const fabricx = new FabricX({
    serverAddr: 'localhost:50051',
    timeout: 120000,
  });

  try {
    // Step 1: Initialize a Fabric network
    console.log('📡 Step 1: Initializing Fabric network...');
    const network = await fabricx.initNetwork({
      name: 'example-network',
      numOrgs: 2,
      channelName: 'mychannel',
    });

    console.log(`✅ Network initialized!`);
    console.log(`   Network ID: ${network.networkId}`);
    console.log(`   Endpoints: ${network.endpoints.join(', ')}\n`);

    // Wait a bit for network to stabilize
    await sleep(5000);

    // Step 2: Get network status
    console.log('📊 Step 2: Checking network status...');
    const status = await fabricx.getNetworkStatus();
    console.log(`   Running: ${status.running}`);
    console.log(`   Status: ${status.status}`);
    console.log(`   Peers: ${status.peers.length}`);
    console.log(`   Orderers: ${status.orderers.length}\n`);

    // Step 3: Deploy chaincode
    console.log('📦 Step 3: Deploying chaincode...');

    // Note: In a real scenario, you would have chaincode in a directory
    // For this example, we'll simulate it
    const chaincodePath = './chaincode/asset-transfer';

    try {
      const deployment = await fabricx.deployChaincode('asset-transfer', {
        path: chaincodePath,
        version: '1.0',
        language: 'golang',
      });

      console.log(`✅ Chaincode deployed!`);
      console.log(`   Chaincode ID: ${deployment.chaincodeId}\n`);

      // Wait for chaincode to be ready
      await sleep(5000);

      // Step 4: Invoke a transaction (create an asset)
      console.log('📝 Step 4: Creating an asset...');
      const createResult = await fabricx.invoke('asset-transfer', 'CreateAsset', [
        'asset1', // ID
        'blue', // Color
        '20', // Size
        'Tom', // Owner
        '100', // AppraisedValue
      ]);

      console.log(`✅ Asset created!`);
      console.log(`   Transaction ID: ${createResult.transactionId}`);
      if (createResult.payload) {
        console.log(`   Response: ${new TextDecoder().decode(createResult.payload)}\n`);
      }

      // Step 5: Query the asset
      console.log('🔍 Step 5: Querying the asset...');
      const queryResult = await fabricx.query('asset-transfer', 'ReadAsset', ['asset1']);

      if (queryResult.payload) {
        const asset = JSON.parse(new TextDecoder().decode(queryResult.payload));
        console.log(`✅ Asset retrieved:`);
        console.log(`   ID: ${asset.ID}`);
        console.log(`   Color: ${asset.Color}`);
        console.log(`   Size: ${asset.Size}`);
        console.log(`   Owner: ${asset.Owner}`);
        console.log(`   Value: ${asset.AppraisedValue}\n`);
      }

      // Step 6: Transfer the asset
      console.log('🔄 Step 6: Transferring asset to Jerry...');
      const transferResult = await fabricx.invoke('asset-transfer', 'TransferAsset', [
        'asset1',
        'Jerry',
      ]);

      console.log(`✅ Asset transferred!`);
      console.log(`   Transaction ID: ${transferResult.transactionId}\n`);

      // Step 7: Verify the transfer
      console.log('✓ Step 7: Verifying transfer...');
      const verifyResult = await fabricx.query('asset-transfer', 'ReadAsset', ['asset1']);

      if (verifyResult.payload) {
        const updatedAsset = JSON.parse(new TextDecoder().decode(verifyResult.payload));
        console.log(`✅ Transfer verified!`);
        console.log(`   New Owner: ${updatedAsset.Owner}\n`);
      }

      // Step 8: Get all assets
      console.log('📋 Step 8: Getting all assets...');
      const allAssetsResult = await fabricx.query('asset-transfer', 'GetAllAssets', []);

      if (allAssetsResult.payload) {
        const assets = JSON.parse(new TextDecoder().decode(allAssetsResult.payload));
        console.log(`✅ Found ${assets.length} asset(s):`);
        assets.forEach((asset: any, index: number) => {
          console.log(
            `   ${index + 1}. ${asset.ID} - Owner: ${asset.Owner}, Value: ${asset.AppraisedValue}`
          );
        });
        console.log();
      }
    } catch (chaincodeError) {
      console.error('⚠️  Chaincode operations failed (chaincode may not exist)');
      console.error("   This is expected if you don't have the asset-transfer chaincode\n");
    }

    // Step 9: Stop the network
    console.log('🛑 Step 9: Stopping network...');
    const cleanup = true; // Set to false to keep containers running
    await fabricx.stopNetwork({ cleanup });

    console.log(`✅ Network stopped ${cleanup ? 'and cleaned up' : ''}!\n`);

    console.log('🎉 Example completed successfully!');
  } catch (error: any) {
    if (error instanceof FabricXError) {
      console.error('\n❌ FabricX Error:');
      console.error(`   Code: ${error.code}`);
      console.error(`   Message: ${error.message}`);
      if (error.details) {
        console.error(`   Details:`, error.details);
      }
    } else {
      console.error('\n❌ Unexpected error:', error);
    }
    process.exit(1);
  } finally {
    // Always close the client
    await fabricx.close();
  }
}

// Helper function to sleep
function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// Run the example
main().catch(console.error);

// ===================================
// Additional Examples
// ===================================

/**
 * Example: Multiple Networks
 */
async function multipleNetworksExample() {
  const fabricx1 = new FabricX();
  const fabricx2 = new FabricX();

  // Initialize two separate networks
  const net1 = await fabricx1.initNetwork({ name: 'network-1' });
  const net2 = await fabricx2.initNetwork({ name: 'network-2' });

  console.log('Network 1:', net1.networkId);
  console.log('Network 2:', net2.networkId);

  // Work with both independently
  // ...

  // Cleanup
  await fabricx1.stopNetwork({ cleanup: true });
  await fabricx2.stopNetwork({ cleanup: true });
}

/**
 * Example: Connecting to Existing Network
 */
async function connectToExistingNetwork() {
  const fabricx = new FabricX();

  // Set the network ID of an existing network
  fabricx.setNetworkId('abc12345');

  // Now you can interact with it
  const status = await fabricx.getNetworkStatus();
  console.log('Network status:', status);

  // Invoke transactions, query, etc.
  await fabricx.invoke('mycc', 'myFunction', ['arg1', 'arg2']);
}

/**
 * Example: Error Handling
 */
async function errorHandlingExample() {
  const fabricx = new FabricX();

  try {
    // This will fail if the network doesn't exist
    await fabricx.invoke('mycc', 'myFunc', ['arg1']);
  } catch (error: any) {
    if (error instanceof FabricXError) {
      switch (error.code) {
        case 'TIMEOUT':
          console.error('Operation timed out');
          break;
        case 'CONNECTION_ERROR':
          console.error('Could not connect to FabricX runtime');
          break;
        case 'GRPC_ERROR':
          console.error('gRPC error:', error.message);
          break;
        default:
          console.error('Unknown error:', error.message);
      }
    }
  }
}

/**
 * Example: Custom Configuration
 */
async function customConfigExample() {
  const fabricx = new FabricX({
    serverAddr: 'my-server.example.com:50051',
    timeout: 300000, // 5 minutes
    useTls: true,
  });

  await fabricx.initNetwork({
    name: 'custom-network',
    numOrgs: 3,
    channelName: 'custom-channel',
    config: {
      'custom-param': 'custom-value',
    },
  });
}

/**
 * Example: Working with Binary Data
 */
async function binaryDataExample() {
  const fabricx = new FabricX();
  await fabricx.initNetwork();

  const result = await fabricx.query('mycc', 'getBinaryData', ['key1']);

  if (result.payload) {
    // Handle as binary
    const bytes = result.payload;
    console.log('Byte length:', bytes.length);

    // Convert to string if it's text
    const text = new TextDecoder().decode(bytes);
    console.log('Text:', text);

    // Parse as JSON if it's JSON
    try {
      const json = JSON.parse(text);
      console.log('JSON:', json);
    } catch {
      console.log('Not JSON data');
    }
  }
}
