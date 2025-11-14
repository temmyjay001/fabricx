import chalk from 'chalk';
import { Command } from 'commander';
import ora from 'ora';
import { createFabricX, handleError, loadConfig, saveConfig } from '../helpers';

export function createStopCommand(): Command {
  const stop = new Command('stop');

  stop
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

        console.log();
        await fabricx.close();
      } catch (error) {
        spinner.fail(chalk.red('Failed to stop network'));
        await fabricx.close();
        handleError(error);
      }
    });

  return stop;
}
