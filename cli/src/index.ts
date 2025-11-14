#!/usr/bin/env node

import { Command } from 'commander';
import { createScaffoldCommand } from './commands/scaffold';
import { createConfigCommand } from './commands/config';
import { createStopCommand } from './commands/stop';
import { createLogsCommand } from './commands/logs';
import { createQueryCommand } from './commands/query';
import { createInvokeCommand } from './commands/invoke';
import { createDeployCommand } from './commands/deploy';
import { createStatusCommand } from './commands/status';
import { createInitCommand } from './commands/init';

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
program.addCommand(createInitCommand());

/**
 * Command: status
 * Get network status
 */
program.addCommand(createStatusCommand());

/**
 * Command: deploy
 * Deploy chaincode
 */
program.addCommand(createDeployCommand());

/**
 * Command: invoke
 * Invoke a transaction
 */
program.addCommand(createInvokeCommand());

/**
 * Command: query
 * Query the ledger
 */
program.addCommand(createQueryCommand());

/**
 * Command: logs
 * Stream container logs
 */
program.addCommand(createLogsCommand());

/**
 * Command: stop
 * Stop and cleanup network
 */
program.addCommand(createStopCommand());

/**
 * Command: config
 * Manage CLI configuration
 */
program.addCommand(createConfigCommand());

/**
 * Command: scaffold
 * Scaffold a new chaincode
 */
program.addCommand(createScaffoldCommand());

// Parse arguments
program.parse(process.argv);

// Show help if no command provided
if (!process.argv.slice(2).length) {
  program.outputHelp();
}
