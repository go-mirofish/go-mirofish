/**
 * Security command: scan for sensitive files before commit.
 * Composable; delegates to security-check module.
 */

const path = require('path');
const securityCheckPath = path.join(__dirname, '../../security-check.js');

async function run() {
  const { main } = require(securityCheckPath);
  main();
}

function help() {
  console.log(`
  hyperagent security

  Scan staged files and working directory for sensitive files.
  Blocks: .env, secrets, credentials, keys, certs, .cursor/mcp.json
  Allows: .env.example, .env.template, .env.sample

  Use before committing or as a pre-commit hook.
`);
}

module.exports = { run, help };
