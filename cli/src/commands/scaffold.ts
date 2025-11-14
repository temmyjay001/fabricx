// cli/src/commands/scaffold.ts
import { Command } from 'commander';
import chalk from 'chalk';
import ora from 'ora';
import { existsSync, mkdirSync, copyFileSync, readdirSync, statSync } from 'fs';
import { join } from 'path';

interface TemplateInfo {
  name: string;
  description: string;
  languages: string[];
  path: string;
}

const templates: Record<string, TemplateInfo> = {
  'asset-transfer': {
    name: 'Asset Transfer',
    description: 'Basic CRUD operations for assets with ownership tracking',
    languages: ['go', 'typescript'],
    path: 'asset-transfer',
  },
  erc20: {
    name: 'ERC-20 Token',
    description: 'Fungible token standard with mint, transfer, and allowance',
    languages: ['go', 'typescript'],
    path: 'erc20',
  },
  escrow: {
    name: 'Escrow',
    description: 'Multi-party escrow with time locks and arbitration',
    languages: ['go', 'typescript'],
    path: 'escrow',
  },
  'supply-chain': {
    name: 'Supply Chain',
    description: 'Track products through supply chain with provenance',
    languages: ['go', 'typescript'],
    path: 'supply-chain',
  },
};

export function createScaffoldCommand(): Command {
  const scaffold = new Command('scaffold');

  scaffold
    .description('Scaffold a new chaincode from a template')
    .argument('[template]', 'Template name (asset-transfer, erc20, escrow, supply-chain)')
    .argument('[path]', 'Target directory for the new chaincode')
    .option('-l, --lang <language>', 'Language (go, typescript)', 'go')
    .option('--list', 'List all available templates')
    .action(async (templateName, targetPath, options) => {
      // List templates
      if (options.list || !templateName) {
        console.log(chalk.cyan('\n📋 Available Templates:\n'));

        Object.entries(templates).forEach(([key, template]) => {
          console.log(chalk.white(`  ${key}`));
          console.log(chalk.gray(`    ${template.description}`));
          console.log(chalk.gray(`    Languages: ${template.languages.join(', ')}\n`));
        });

        if (!templateName) {
          console.log(chalk.yellow('💡 Usage:'));
          console.log(chalk.white('  npx fabricx scaffold <template> <path> --lang <language>\n'));
          console.log(chalk.yellow('💡 Example:'));
          console.log(chalk.white('  npx fabricx scaffold erc20 ./my-token --lang go\n'));
        }

        return;
      }

      // Validate template
      const template = templates[templateName];
      if (!template) {
        console.error(chalk.red(`\n✗ Template "${templateName}" not found`));
        console.log(
          chalk.yellow('\n💡 Run'),
          chalk.white('npx fabricx scaffold --list'),
          chalk.yellow('to see available templates\n')
        );
        process.exit(1);
      }

      // Validate language
      if (!template.languages.includes(options.lang)) {
        console.error(
          chalk.red(`\n✗ Language "${options.lang}" not available for ${templateName}`)
        );
        console.log(chalk.yellow(`   Available languages: ${template.languages.join(', ')}\n`));
        process.exit(1);
      }

      // Validate target path
      if (!targetPath) {
        console.error(chalk.red('\n✗ Target path is required\n'));
        console.log(chalk.yellow('💡 Usage:'));
        console.log(
          chalk.white(`  npx fabricx scaffold ${templateName} <path> --lang ${options.lang}\n`)
        );
        process.exit(1);
      }

      // Check if target exists
      if (existsSync(targetPath)) {
        console.error(chalk.red(`\n✗ Directory already exists: ${targetPath}\n`));
        process.exit(1);
      }

      const spinner = ora(`Scaffolding ${template.name} (${options.lang})...`).start();

      try {
        // Determine template source path
        // In production, this would be in node_modules/@fabricx/cli/templates
        // For development, it's relative to the CLI package
        const templateBasePath = join(__dirname, '../../../templates');
        const templateSourcePath = join(
          templateBasePath,
          template.path,
          options.lang === 'typescript' ? 'typescript' : 'go'
        );

        if (!existsSync(templateSourcePath)) {
          spinner.fail(chalk.red('Template source not found'));
          console.error(chalk.gray(`  Expected at: ${templateSourcePath}\n`));
          process.exit(1);
        }

        // Create target directory
        mkdirSync(targetPath, { recursive: true });

        // Copy template files
        copyDirectory(templateSourcePath, targetPath);

        spinner.succeed(chalk.green('Chaincode scaffolded successfully!'));

        console.log(chalk.cyan('\n📦 Chaincode Details:'));
        console.log(chalk.gray('  Template:'), chalk.white(template.name));
        console.log(chalk.gray('  Language:'), chalk.white(options.lang));
        console.log(chalk.gray('  Location:'), chalk.white(targetPath));

        console.log(chalk.cyan('\n🚀 Next Steps:\n'));

        if (options.lang === 'go') {
          console.log(chalk.white('  1. Review the chaincode:'));
          console.log(chalk.gray(`     cd ${targetPath}`));
          console.log(chalk.gray('     cat main.go\n'));

          console.log(chalk.white('  2. Install dependencies:'));
          console.log(chalk.gray('     go mod tidy\n'));

          console.log(chalk.white('  3. Run tests:'));
          console.log(chalk.gray('     go test -v\n'));

          console.log(chalk.white('  4. Deploy to network:'));
          console.log(chalk.gray(`     npx fabricx deploy my-chaincode ${targetPath}\n`));
        } else {
          console.log(chalk.white('  1. Review the chaincode:'));
          console.log(chalk.gray(`     cd ${targetPath}`));
          console.log(chalk.gray('     cat src/index.ts\n'));

          console.log(chalk.white('  2. Install dependencies:'));
          console.log(chalk.gray('     npm install\n'));

          console.log(chalk.white('  3. Run tests:'));
          console.log(chalk.gray('     npm test\n'));

          console.log(chalk.white('  4. Deploy to network:'));
          console.log(chalk.gray(`     npx fabricx deploy my-chaincode ${targetPath}\n`));
        }

        console.log(
          chalk.yellow('💡 Tip:'),
          chalk.white('Check the README.md for template-specific instructions\n')
        );
      } catch (error) {
        spinner.fail(chalk.red('Failed to scaffold chaincode'));
        console.error(chalk.gray(`  ${(error as Error).message}\n`));
        process.exit(1);
      }
    });

  return scaffold;
}

function copyDirectory(src: string, dest: string): void {
  const entries = readdirSync(src);

  for (const entry of entries) {
    const srcPath = join(src, entry);
    const destPath = join(dest, entry);

    if (statSync(srcPath).isDirectory()) {
      mkdirSync(destPath, { recursive: true });
      copyDirectory(srcPath, destPath);
    } else {
      copyFileSync(srcPath, destPath);
    }
  }
}
