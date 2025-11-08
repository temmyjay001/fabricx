#!/usr/bin/env node

import { Command } from 'commander';
import { FabricX } from '@fabricx/sdk';

const program = new Command();
const fabricx = new FabricX();

program
  .name('fabricx')
  .description('A CLI for interacting with Hyperledger Fabric networks')
  .version('1.0.0');

program
  .command('init')
  .description('Bootstrap a local Fabric network')
  .action(async () => {
    await fabricx.initNetwork();
  });

program
  .command('deploy <chaincode>')
  .description('Deploy a chaincode from a template')
  .action(async (chaincode) => {
    await fabricx.deployChaincode(chaincode);
  });

program
  .command('invoke <chaincode> <function>')
  .description('Invoke a chaincode function')
  .option('--args <args>', 'Arguments for the function as a JSON array', '[]')
  .action(async (chaincode, func, options) => {
    const args = JSON.parse(options.args);
    await fabricx.invoke(chaincode, func, args);
  });

program
  .command('query <chaincode> <function>')
  .description('Query a chaincode function')
  .option('--args <args>', 'Arguments for the function as a JSON array', '[]')
  .action(async (chaincode, func, options) => {
    const args = JSON.parse(options.args);
    await fabricx.query(chaincode, func, args);
  });

program
  .command('stop')
  .description('Tear down the local Fabric network')
  .action(async () => {
    await fabricx.stopNetwork();
  });

program.parse(process.argv);
