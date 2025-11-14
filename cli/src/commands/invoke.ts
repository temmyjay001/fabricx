import chalk from 'chalk';
import { Command } from 'commander';
import ora from 'ora';
import { loadConfig, createFabricX, formatDuration, handleError } from '../helpers';

export function createInvokeCommand(): Command {
  const invoke = new Command('invoke');

  invoke
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

        console.log();
        await fabricx.close();
      } catch (error) {
        spinner.fail(chalk.red('Transaction failed'));
        await fabricx.close();
        handleError(error);
      }
    });

  return invoke;
}
