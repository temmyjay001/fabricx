import chalk from 'chalk';
import { Command } from 'commander';
import ora from 'ora';
import { loadConfig, createFabricX, formatDuration, handleError } from '../helpers';

export function createDeployCommand(): Command {
  const deploy = new Command('deploy');

  deploy
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

  return deploy;
}
