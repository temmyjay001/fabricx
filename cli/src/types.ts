import { LogLevel } from '@fabricx/sdk';

/**
 * CLI Configuration
 */
export interface CliConfig {
  serverAddr?: string;
  timeout?: number;
  useTls?: boolean;
  logLevel?: LogLevel;
  lastNetworkId?: string;
}
