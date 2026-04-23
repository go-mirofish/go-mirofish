/**
 * Linear command: optional integration with linear CLI.
 * Resolves current issue from branch, can append to commit messages.
 * Requires: linear CLI on PATH (linear --version).
 */

const { execSync, spawn } = require('child_process');
const { ok, warn, fail, arrow } = require('../lib/output');

function hasLinear() {
  try {
    execSync('linear --version', { stdio: 'pipe' });
    return true;
  } catch {
    return false;
  }
}

async function getIssueId() {
  if (!hasLinear()) return null;
  try {
    const out = execSync('linear issue id', { encoding: 'utf8' });
    return (out || '').trim() || null;
  } catch {
    return null;
  }
}

function describeIssue(issueId) {
  if (!hasLinear() || !issueId) return null;
  try {
    const out = execSync(`linear issue describe ${issueId}`, { encoding: 'utf8' });
    return (out || '').trim() || null;
  } catch {
    return null;
  }
}

async function run(opts = {}) {
  if (!hasLinear()) {
    console.log(warn('Linear CLI not found. Install: https://github.com/schpet/linear-cli'));
    process.exitCode = 1;
    return;
  }

  const issueId = opts.issue || (await getIssueId());
  if (!issueId) {
    console.log(warn('No Linear issue in branch name or --issue. Use feature/ENG-123 style branch.'));
    return;
  }

  const trailer = describeIssue(issueId);
  if (trailer) {
    console.log(trailer);
  } else {
    console.log(`Linear-issue: ${issueId}`);
  }
}

function help() {
  console.log(`
  hyperagent linear [--issue <id>]

  Print Linear issue trailer for commit messages.
  Uses current branch to resolve issue ID if --issue not given.
  Requires linear CLI on PATH.

  Example:
    hyperagent linear
    hyperagent linear --issue ENG-123
`);
}

module.exports = { run, help, getIssueId, describeIssue, hasLinear };
