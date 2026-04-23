/**
 * Commit command: parallel commit with security checks.
 * Composable; delegates to parallel-commit module.
 */

const path = require('path');

const parallelCommitPath = path.join(__dirname, '../../parallel-commit.js');

async function run(opts = {}) {
  const prevArgv = process.argv.slice();
  const args = ['node', 'hyperagent-commit'];
  if (opts.dryRun) args.push('--dry-run');
  if (opts.noSecurityCheck) args.push('--no-security-check');
  if (opts.warnOnly) args.push('--warn-only');
  process.argv = args;

  const mod = require(parallelCommitPath);
  if (opts.max) mod.config.maxConcurrentCommits = opts.max;
  await mod.main();
  process.argv = prevArgv;
}

function help() {
  console.log(`
  hyperagent commit [options]

  Parallel commit with security checks. Commits each changed file individually.

  Options:
    --dry-run              Preview what would be committed
    --no-security-check    Disable security checks (not recommended)
    --warn-only            Warn on sensitive files, do not fail
    --max <n>              Max concurrent commits (default: 5)
    -h, --help             Show this help

  Examples:
    hyperagent commit --dry-run
    hyperagent commit --max 3
`);
}

module.exports = { run, help };
