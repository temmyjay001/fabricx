import chalk from 'chalk';
import { Command } from 'commander';
import ora from 'ora';
import { loadConfig, createFabricX, handleError } from '../helpers';

export function createStatusCommand(): Command {
  const status = new Command('status');

  status
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
        console.log(
          chalk.gray('  Running:'),
          status.running ? chalk.green('Yes') : chalk.red('No')
        );
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

        console.log();
        await fabricx.close();
      } catch (error) {
        spinner.fail(chalk.red('Failed to get network status'));
        await fabricx.close();
        handleError(error);
      }
    });

  return status;
}
