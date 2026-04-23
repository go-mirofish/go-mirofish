#!/usr/bin/env node

/**
 * hyperagent-commit CLI
 * Composable commands for git commit with security checks and optional Linear integration.
 * Uses cli-building (async-first, composable) and linear-cli skills.
 */

const { parse } = require('./lib/parser');
const { style } = require('./lib/output');

const commands = {
  commit: require('./commands/commit'),
  security: require('./commands/security'),
  linear: require('./commands/linear'),
};

async function main() {
  const { args, opts } = parse();

  const sub = args[0] && !args[0].startsWith('-') ? args[0] : 'commit';
  const cmd = commands[sub];

  if (!cmd) {
    console.error(`Unknown command: ${args[0] || '(none)'}`);
    printUsage();
    process.exit(1);
  }

  if (opts.help) {
    cmd.help();
    return;
  }

  try {
    await cmd.run(opts);
  } catch (err) {
    console.error(err.message || err);
    process.exit(1);
  }
}

function printUsage() {
  console.log(`
${style('hyperagent-commit', 'bright')} - Git commit CLI with security checks

Usage: hyperagent commit [options]   (default)
       hyperagent security
       hyperagent linear [--issue <id>]

Commands:
  commit    Parallel commit with security checks (default)
  security  Scan for sensitive files only
  linear    Print Linear issue trailer for commit messages

Options:
  -h, --help    Show help for command
`);
}

if (require.main === module) {
  main().catch((err) => {
    console.error(err);
    process.exit(1);
  });
}

module.exports = { main, commands };
