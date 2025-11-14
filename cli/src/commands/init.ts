import chalk from 'chalk';
import { Command } from 'commander';
import ora from 'ora';
import { loadConfig, createFabricX, formatDuration, saveConfig, handleError } from '../helpers';

export function createInitCommand(): Command {
  const init = new Command('init');

  init
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

  return init;
}
