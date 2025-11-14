import { LogLevel } from '@fabricx/sdk';
import chalk from 'chalk';
import { Command } from 'commander';
import { loadConfig, saveConfig } from '../helpers';

export function createConfigCommand(): Command {
  const config = new Command('config');

  config
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
        console.log();
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
        console.log();
      } else {
        console.log(chalk.yellow('No configuration changes made'));
        console.log(chalk.gray('Use --help to see available options\n'));
      }
    });

  return config;
}
