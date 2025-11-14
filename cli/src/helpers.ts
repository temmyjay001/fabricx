import { homedir } from 'os';
import { CliConfig } from './types';
import { join } from 'path';
import { existsSync, readFileSync } from 'fs';
import chalk from 'chalk';
import { FabricX, FabricXError, LogLevel, RetryStrategy } from '@fabricx/sdk';

/**
 * Load CLI configuration
 */
export function loadConfig(): CliConfig {
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
export function saveConfig(config: CliConfig): void {
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
export function createFabricX(config: CliConfig): FabricX {
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
 * Handle errors consistently
 */
export function handleError(error: unknown): never {
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

/**
 * Format duration in ms to human-readable string
 */
export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}
