import chalk from 'chalk';
import { Command } from 'commander';
import { loadConfig, createFabricX, handleError } from '../helpers';

export function createLogsCommand(): Command {
  const logs = new Command('logs');

  logs
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

  return logs;
}
