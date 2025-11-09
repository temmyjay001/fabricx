#!/usr/bin/env node

import { Command } from 'commander';
import { FabricX, FabricXError, LogLevel } from '@fabricx/sdk';
import chalk from 'chalk';
import ora from 'ora';
import { readFileSync, existsSync } from 'fs';
import { homedir } from 'os';
import { join } from 'path';
import { RetryStrategy } from '@fabricx/sdk/src/utils/retry';

/**
 * CLI Configuration
 */
interface CliConfig {
  serverAddr?: string;
  timeout?: number;
  useTls?: boolean;
  logLevel?: LogLevel;
  lastNetworkId?: string;
}

/**
 * Load CLI configuration
 */
function loadConfig(): CliConfig {
  const configPath = join(homedir(), '.fabricxrc');

  if (existsSync(configPath)) {
    try {
      const content = readFileSync(configPath, 'utf-8');
      return JSON.parse(content);
    } catch (error) {
      console.warn(chalk.yellow('Warning: Failed to load config file'));
    }
  }

  return {};
}

/**
 * Save CLI configuration
 */
function saveConfig(config: CliConfig): void {
  const configPath = join(homedir(), '.fabricxrc');

  try {
    const fs = require('fs');
    fs.writeFileSync(configPath, JSON.stringify(config, null, 2));
  } catch (error) {
    console.warn(chalk.yellow('Warning: Failed to save config file'));
  }
}

/**
 * Create FabricX instance with CLI config
 */
function createFabricX(config: CliConfig): FabricX {
  return new FabricX({
    serverAddr: config.serverAddr || 'localhost:50051',
    timeout: config.timeout || 120000,
    useTls: config.useTls || false,
    useConnectionPool: true,
    minConnections: 1,
    maxConnections: 5,
    logger: {
      level: config.logLevel || LogLevel.INFO,
      enableConsole: false, // We'll handle console output ourselves
    },
    retry: {
      maxAttempts: 3,
      initialDelay: 1000,
      strategy: 'exponential' as RetryStrategy,
      backoffMultiplier: 2,
    },
  });
}

/**
 * Format duration in ms to human-readable string
 */
function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

/**
 * Format bytes to human-readable string
 */
function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes}B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)}MB`;
}

/**
 * Handle errors consistently
 */
function handleError(error: unknown): never {
  if (error instanceof FabricXError) {
    console.error(chalk.red('\n✗ Error:'), error.message);
    console.error(chalk.gray(`  Code: ${error.code}`));

    if (error.details) {
      console.error(chalk.gray('  Details:'), JSON.stringify(error.details, null, 2));
    }
  } else if (error instanceof Error) {
    console.error(chalk.red('\n✗ Error:'), error.message);
  } else {
    console.error(chalk.red('\n✗ Unknown error occurred'));
  }

  process.exit(1);
}

// Create CLI program
const program = new Command();

program
  .name('fabricx')
  .description('FabricX CLI - A developer-friendly toolkit for Hyperledger Fabric')
  .version('1.0.0')
  .option('-s, --server <address>', 'FabricX runtime server address')
  .option('-t, --timeout <ms>', 'Request timeout in milliseconds')
  .option('--tls', 'Enable TLS')
  .option('--log-level <level>', 'Log level (debug, info, warn, error, silent)');

/**
 * Command: init
 * Initialize a new Fabric network
 */
program
  .command('init')
  .description('Initialize a new Fabric network')
  .option('-n, --name <name>', 'Network name', 'fabricx-network')
  .option('-o, --orgs <number>', 'Number of organizations', '2')
  .option('-c, --channel <name>', 'Channel name', 'mychannel')
  .action(async (options) => {
    const config = loadConfig();
    const fabricx = createFabricX(config);
    const spinner = ora('Initializing Fabric network...').start();
    const startTime = Date.now();

    try {
      const result = await fabricx.initNetwork({
        name: options.name,
        numOrgs: parseInt(options.orgs),
        channelName: options.channel,
      });

      const duration = Date.now() - startTime;
      spinner.succeed(chalk.green(`Network initialized in ${formatDuration(duration)}`));

      console.log(chalk.cyan('\n📋 Network Details:'));
      console.log(chalk.gray('  Network ID:'), chalk.white(result.networkId));
      console.log(chalk.gray('  Name:'), chalk.white(options.name));
      console.log(chalk.gray('  Organizations:'), chalk.white(options.orgs));
      console.log(chalk.gray('  Channel:'), chalk.white(options.channel));
      console.log(chalk.gray('  Endpoints:'));
      result.endpoints.forEach((endpoint) => {
        console.log(chalk.gray('    •'), chalk.white(endpoint));
      });

      // Save network ID
      config.lastNetworkId = result.networkId;
      saveConfig(config);

      console.log(chalk.green('\n✓ Network ready for chaincode deployment'));
      console.log(
        chalk.gray('\n💡 Tip: Save this network ID or use it automatically in future commands\n')
      );

      await fabricx.close();
    } catch (error) {
      spinner.fail(chalk.red('Network initialization failed'));
      await fabricx.close();
      handleError(error);
    }
  });

/**
 * Command: status
 * Get network status
 */
program
  .command('status')
  .description('Get the status of a network')
  .argument('[network-id]', 'Network ID (uses last network if not provided)')
  .action(async (networkIdArg) => {
    const config = loadConfig();
    const fabricx = createFabricX(config);
    const spinner = ora('Fetching network status...').start();

    try {
      const networkId = networkIdArg || config.lastNetworkId;

      if (!networkId) {
        spinner.fail(chalk.red('No network ID provided'));
        console.error(
          chalk.yellow('\n💡 Tip: Initialize a network first with'),
          chalk.white('fabricx init')
        );
        process.exit(1);
      }

      fabricx.setNetworkId(networkId);
      const status = await fabricx.getNetworkStatus();

      spinner.succeed(chalk.green('Network status retrieved'));

      console.log(chalk.cyan('\n📊 Network Status:'));
      console.log(chalk.gray('  Network ID:'), chalk.white(networkId));
      console.log(chalk.gray('  Running:'), status.running ? chalk.green('Yes') : chalk.red('No'));
      console.log(chalk.gray('  Status:'), chalk.white(status.status));

      if (status.peers.length > 0) {
        console.log(chalk.cyan('\n👥 Peers:'));
        status.peers.forEach((peer) => {
          console.log(chalk.gray(`  • ${peer.name}`));
          console.log(chalk.gray(`    Organization: ${peer.org}`));
          console.log(chalk.gray(`    Status: ${peer.status}`));
          console.log(chalk.gray(`    Endpoint: ${peer.endpoint}`));
        });
      }

      if (status.orderers.length > 0) {
        console.log(chalk.cyan('\n⚙️  Orderers:'));
        status.orderers.forEach((orderer) => {
          console.log(chalk.gray(`  • ${orderer.name}`));
          console.log(chalk.gray(`    Status: ${orderer.status}`));
          console.log(chalk.gray(`    Endpoint: ${orderer.endpoint}`));
        });
      }

      // Show connection pool stats if available
      const poolStats = fabricx.getPoolStats();
      if (poolStats) {
        console.log(chalk.cyan('\n🔗 Connection Pool:'));
        console.log(chalk.gray('  Total Connections:'), chalk.white(poolStats.totalConnections));
        console.log(chalk.gray('  Active:'), chalk.white(poolStats.activeConnections));
        console.log(chalk.gray('  Idle:'), chalk.white(poolStats.idleConnections));
        console.log(chalk.gray('  Total Requests:'), chalk.white(poolStats.totalRequests));
      }

      console.log(); // Empty line
      await fabricx.close();
    } catch (error) {
      spinner.fail(chalk.red('Failed to get network status'));
      await fabricx.close();
      handleError(error);
    }
  });

/**
 * Command: deploy
 * Deploy chaincode
 */
program
  .command('deploy')
  .description('Deploy a chaincode to the network')
  .argument('<chaincode>', 'Chaincode name')
  .argument('[path]', 'Path to chaincode directory')
  .option('-v, --version <version>', 'Chaincode version', '1.0')
  .option('-l, --language <lang>', 'Chaincode language (golang, node, java)', 'golang')
  .option('-n, --network <id>', 'Network ID (uses last network if not provided)')
  .option('-e, --endorsement <orgs>', 'Endorsement policy organizations (comma-separated)')
  .action(async (chaincode, pathArg, options) => {
    const config = loadConfig();
    const fabricx = createFabricX(config);
    const networkId = options.network || config.lastNetworkId;

    if (!networkId) {
      console.error(chalk.red('✗ No network ID provided'));
      console.error(
        chalk.yellow('\n💡 Tip: Initialize a network first with'),
        chalk.white('fabricx init')
      );
      process.exit(1);
    }

    fabricx.setNetworkId(networkId);

    const spinner = ora(`Deploying chaincode: ${chaincode}`).start();
    const startTime = Date.now();

    try {
      const endorsementOrgs = options.endorsement
        ? options.endorsement.split(',').map((org: string) => org.trim())
        : [];

      const result = await fabricx.deployChaincode(chaincode, {
        path: pathArg || `./${chaincode}`,
        version: options.version,
        language: options.language,
        endorsementPolicyOrgs: endorsementOrgs,
      });

      const duration = Date.now() - startTime;
      spinner.succeed(chalk.green(`Chaincode deployed in ${formatDuration(duration)}`));

      console.log(chalk.cyan('\n📦 Deployment Details:'));
      console.log(chalk.gray('  Chaincode ID:'), chalk.white(result.chaincodeId));
      console.log(chalk.gray('  Name:'), chalk.white(chaincode));
      console.log(chalk.gray('  Version:'), chalk.white(options.version));
      console.log(chalk.gray('  Language:'), chalk.white(options.language));
      console.log(chalk.gray('  Path:'), chalk.white(pathArg || `./${chaincode}`));

      console.log(chalk.green('\n✓ Chaincode ready for transactions\n'));

      await fabricx.close();
    } catch (error) {
      spinner.fail(chalk.red('Chaincode deployment failed'));
      await fabricx.close();
      handleError(error);
    }
  });

/**
 * Command: invoke
 * Invoke a transaction
 */
program
  .command('invoke')
  .description('Invoke a chaincode transaction')
  .argument('<chaincode>', 'Chaincode name')
  .argument('<function>', 'Function name')
  .argument('[args...]', 'Function arguments')
  .option('-n, --network <id>', 'Network ID (uses last network if not provided)')
  .option('--transient', 'Use transient data')
  .action(async (chaincode, func, args, options) => {
    const config = loadConfig();
    const fabricx = createFabricX(config);
    const networkId = options.network || config.lastNetworkId;

    if (!networkId) {
      console.error(chalk.red('✗ No network ID provided'));
      console.error(
        chalk.yellow('\n💡 Tip: Initialize a network first with'),
        chalk.white('fabricx init')
      );
      process.exit(1);
    }

    fabricx.setNetworkId(networkId);

    const spinner = ora(`Invoking: ${chaincode}.${func}()`).start();
    const startTime = Date.now();

    try {
      const result = await fabricx.invoke(chaincode, func, args, {
        transient: options.transient,
      });

      const duration = Date.now() - startTime;
      spinner.succeed(chalk.green(`Transaction completed in ${formatDuration(duration)}`));

      console.log(chalk.cyan('\n📝 Transaction Details:'));
      console.log(chalk.gray('  Transaction ID:'), chalk.white(result.transactionId));
      console.log(chalk.gray('  Chaincode:'), chalk.white(chaincode));
      console.log(chalk.gray('  Function:'), chalk.white(func));
      console.log(chalk.gray('  Arguments:'), chalk.white(JSON.stringify(args)));

      if (result.payload && result.payload.length > 0) {
        console.log(chalk.cyan('\n📄 Response:'));
        try {
          const payloadStr = new TextDecoder().decode(result.payload);
          const payloadJson = JSON.parse(payloadStr);
          console.log(chalk.white(JSON.stringify(payloadJson, null, 2)));
        } catch {
          const payloadStr = new TextDecoder().decode(result.payload);
          console.log(chalk.white(payloadStr));
        }
      }

      console.log(); // Empty line
      await fabricx.close();
    } catch (error) {
      spinner.fail(chalk.red('Transaction failed'));
      await fabricx.close();
      handleError(error);
    }
  });

/**
 * Command: query
 * Query the ledger
 */
program
  .command('query')
  .description('Query the ledger')
  .argument('<chaincode>', 'Chaincode name')
  .argument('<function>', 'Function name')
  .argument('[args...]', 'Function arguments')
  .option('-n, --network <id>', 'Network ID (uses last network if not provided)')
  .action(async (chaincode, func, args, options) => {
    const config = loadConfig();
    const fabricx = createFabricX(config);
    const networkId = options.network || config.lastNetworkId;

    if (!networkId) {
      console.error(chalk.red('✗ No network ID provided'));
      console.error(
        chalk.yellow('\n💡 Tip: Initialize a network first with'),
        chalk.white('fabricx init')
      );
      process.exit(1);
    }

    fabricx.setNetworkId(networkId);

    const spinner = ora(`Querying: ${chaincode}.${func}()`).start();
    const startTime = Date.now();

    try {
      const result = await fabricx.query(chaincode, func, args);

      const duration = Date.now() - startTime;
      spinner.succeed(chalk.green(`Query completed in ${formatDuration(duration)}`));

      console.log(chalk.cyan('\n🔍 Query Details:'));
      console.log(chalk.gray('  Chaincode:'), chalk.white(chaincode));
      console.log(chalk.gray('  Function:'), chalk.white(func));
      console.log(chalk.gray('  Arguments:'), chalk.white(JSON.stringify(args)));

      if (result.payload && result.payload.length > 0) {
        console.log(chalk.cyan('\n📄 Result:'));
        try {
          const payloadStr = new TextDecoder().decode(result.payload);
          const payloadJson = JSON.parse(payloadStr);
          console.log(chalk.white(JSON.stringify(payloadJson, null, 2)));
        } catch {
          const payloadStr = new TextDecoder().decode(result.payload);
          console.log(chalk.white(payloadStr));
        }
      }

      console.log(); // Empty line
      await fabricx.close();
    } catch (error) {
      spinner.fail(chalk.red('Query failed'));
      await fabricx.close();
      handleError(error);
    }
  });

/**
 * Command: logs
 * Stream container logs
 */
program
  .command('logs')
  .description('Stream logs from network containers')
  .argument('[container]', 'Container name (all containers if not provided)')
  .option('-n, --network <id>', 'Network ID (uses last network if not provided)')
  .action(async (container, options) => {
    const config = loadConfig();
    const fabricx = createFabricX(config);
    const networkId = options.network || config.lastNetworkId;

    if (!networkId) {
      console.error(chalk.red('✗ No network ID provided'));
      console.error(
        chalk.yellow('\n💡 Tip: Initialize a network first with'),
        chalk.white('fabricx init')
      );
      process.exit(1);
    }

    fabricx.setNetworkId(networkId);

    console.log(chalk.cyan(`\n📜 Streaming logs from network: ${networkId}`));
    if (container) {
      console.log(chalk.gray(`   Container: ${container}`));
    }
    console.log(chalk.yellow('   Press Ctrl+C to stop\n'));

    try {
      const cancelStream = await fabricx.streamLogs((log) => {
        const timestamp = new Date(log.timestamp).toLocaleTimeString();
        console.log(
          chalk.gray(`[${timestamp}]`),
          chalk.cyan(`[${log.container}]`),
          chalk.white(log.message)
        );
      }, container);

      // Handle Ctrl+C
      process.on('SIGINT', async () => {
        console.log(chalk.yellow('\n\n⏹  Stopping log stream...'));
        cancelStream();
        await fabricx.close();
        process.exit(0);
      });
    } catch (error) {
      await fabricx.close();
      handleError(error);
    }
  });

/**
 * Command: stop
 * Stop and cleanup network
 */
program
  .command('stop')
  .description('Stop and cleanup a network')
  .argument('[network-id]', 'Network ID (uses last network if not provided)')
  .option('--cleanup', 'Remove containers and volumes')
  .action(async (networkIdArg, options) => {
    const config = loadConfig();
    const fabricx = createFabricX(config);
    const networkId = networkIdArg || config.lastNetworkId;

    if (!networkId) {
      console.error(chalk.red('✗ No network ID provided'));
      console.error(
        chalk.yellow('\n💡 Tip: Initialize a network first with'),
        chalk.white('fabricx init')
      );
      process.exit(1);
    }

    fabricx.setNetworkId(networkId);

    const spinner = ora(`Stopping network: ${networkId}`).start();

    try {
      await fabricx.stopNetwork({ cleanup: options.cleanup });

      spinner.succeed(chalk.green('Network stopped successfully'));

      if (options.cleanup) {
        console.log(chalk.gray('  All containers and volumes removed'));
      }

      // Clear last network ID if it was stopped
      if (config.lastNetworkId === networkId) {
        delete config.lastNetworkId;
        saveConfig(config);
      }

      console.log(); // Empty line
      await fabricx.close();
    } catch (error) {
      spinner.fail(chalk.red('Failed to stop network'));
      await fabricx.close();
      handleError(error);
    }
  });

/**
 * Command: config
 * Manage CLI configuration
 */
program
  .command('config')
  .description('Manage CLI configuration')
  .option('--show', 'Show current configuration')
  .option('--reset', 'Reset configuration to defaults')
  .option('--set-server <address>', 'Set server address')
  .option('--set-timeout <ms>', 'Set timeout in milliseconds')
  .option('--set-log-level <level>', 'Set log level')
  .action((options) => {
    const config = loadConfig();

    if (options.show) {
      console.log(chalk.cyan('\n⚙️  Current Configuration:'));
      console.log(chalk.gray('  Server:'), chalk.white(config.serverAddr || 'localhost:50051'));
      console.log(chalk.gray('  Timeout:'), chalk.white(`${config.timeout || 120000}ms`));
      console.log(chalk.gray('  TLS:'), chalk.white(config.useTls ? 'Enabled' : 'Disabled'));
      console.log(chalk.gray('  Log Level:'), chalk.white(config.logLevel || 'info'));
      if (config.lastNetworkId) {
        console.log(chalk.gray('  Last Network:'), chalk.white(config.lastNetworkId));
      }
      console.log(); // Empty line
      return;
    }

    if (options.reset) {
      saveConfig({});
      console.log(chalk.green('✓ Configuration reset to defaults\n'));
      return;
    }

    let updated = false;

    if (options.setServer) {
      config.serverAddr = options.setServer;
      updated = true;
      console.log(chalk.green(`✓ Server address set to: ${options.setServer}`));
    }

    if (options.setTimeout) {
      config.timeout = parseInt(options.setTimeout);
      updated = true;
      console.log(chalk.green(`✓ Timeout set to: ${options.setTimeout}ms`));
    }

    if (options.setLogLevel) {
      config.logLevel = options.setLogLevel as LogLevel;
      updated = true;
      console.log(chalk.green(`✓ Log level set to: ${options.setLogLevel}`));
    }

    if (updated) {
      saveConfig(config);
      console.log(); // Empty line
    } else {
      console.log(chalk.yellow('No configuration changes made'));
      console.log(chalk.gray('Use --help to see available options\n'));
    }
  });

// Parse arguments
program.parse(process.argv);

// Show help if no command provided
if (!process.argv.slice(2).length) {
  program.outputHelp();
}
