#!/usr/bin/env node

const { execSync, spawnSync } = require('child_process');
const fs = require('fs');
const path = require('path');

// Colors for console output
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  dim: '\x1b[2m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m'
};

// Security: Sensitive file patterns that should NEVER be committed
const SENSITIVE_PATTERNS = [
  /^\.env$/,
  /^\.env\.local$/,
  /^\.env\.[^.]*\.local$/,
  /^\.env\.production$/,
  /^\.env\.development$/,
  /^\.env\.test$/,
  /^\.env\.staging$/,
  /\.env$/,
  /\.secrets?$/,
  /secrets\/.*/,
  /\.secret$/,
  /\.envrc$/,
  /\.env\.backup$/,
  /\.env\.[^.]*\.backup$/,
  /\.pem$/,
  /\.key$/,
  /\.cert$/,
  /\.p12$/,
  /\.pfx$/,
  /\.jks$/,
  /\.keystore$/,
  /credentials\.json$/,
  /service-account\.json$/,
  /auth\.json$/,
  /token\.json$/,
  /\.gpg$/,
  /\.pgp$/,
  /\.cursor\/mcp\.json$/  // MCP config may contain secrets
];

// Security: Allowed example/template files (these are safe to commit)
const ALLOWED_EXAMPLES = [
  /\.env\.example$/,
  /\.env\.template$/,
  /\.env\.sample$/,
  /\.env\.example\.local$/,
  /\.cursor\/mcp\.json\.example$/,
  /\.cursor\/mcp\.example\.json$/
];

// Configuration
const config = {
  maxConcurrentCommits: 1, // Sequential: each file must be added+committed before next (parallel would race)
  // Message generation: diff-parser (getSpecificMessage) extracts exact changes; no static prefix
  dryRun: false, // Set to true to see what would be committed without actually committing
  excludePatterns: [
    'node_modules/**',
    '.git/**',
    '*.log',
    '*.tmp',
    '.DS_Store',
    'Thumbs.db'
  ],
  // Windows reserved device names (nul, con, prn, aux, com1-9, lpt1-9) - cannot be committed
  windowsReservedNames: /(^|\/)(nul|con|prn|aux|com[1-9]|lpt[1-9])(\.|$)/i,
  securityCheck: true, // Enable security checks for sensitive files
  failOnSensitive: true // Fail if sensitive files are detected (set to false to only warn)
};

// Utility functions
function log(message, color = 'reset') {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

// Security: Check if file is sensitive
function isSensitiveFile(file) {
  // Check if it's an allowed example file first
  if (ALLOWED_EXAMPLES.some(pattern => pattern.test(file))) {
    return false; // Example files are safe
  }
  
  // Check against sensitive patterns
  return SENSITIVE_PATTERNS.some(pattern => pattern.test(file));
}

// Security: Validate files before committing
function validateFiles(files) {
  const sensitiveFiles = files.filter(({ file }) => isSensitiveFile(file));
  
  if (sensitiveFiles.length > 0) {
    log('\n⚠️  SECURITY WARNING: Sensitive files detected!', 'red');
    log('The following files contain sensitive information and should NOT be committed:', 'red');
    
    sensitiveFiles.forEach(({ file, status }) => {
      log(`  ❌ "${file}" (${status})`, 'red');
    });
    
    log('\n💡 Allowed example files (safe to commit):', 'yellow');
    log('  ✅ .env.example', 'green');
    log('  ✅ .env.template', 'green');
    log('  ✅ .env.sample', 'green');
    log('  ✅ .cursor/mcp.json.example', 'green');
    
    log('\n🔒 Security Recommendations:', 'yellow');
    log('  1. Ensure .env is in .gitignore', 'yellow');
    log('  2. Use .env.example as a template', 'yellow');
    log('  3. Never commit actual credentials or tokens', 'yellow');
    log('  4. Revoke any exposed credentials immediately', 'yellow');
    
    if (config.failOnSensitive) {
      log('\n❌ Commit blocked for security reasons!', 'red');
      log('Remove sensitive files from staging or update .gitignore', 'red');
      return false;
    } else {
      log('\n⚠️  Continuing with commit (failOnSensitive is disabled)', 'yellow');
      log('⚠️  This is NOT recommended for production!', 'yellow');
    }
  }
  
  return true;
}

function getChangedFiles() {
  try {
    const output = execSync('git status --porcelain', { encoding: 'utf8' });
    const allFiles = output
      .split('\n')
      .filter(line => line.trim())
      .map(line => {
        const status = line.substring(0, 2);
        const file = line.substring(3);
        return { status, file };
      })
      .filter(({ file }) => {
        // Filter out excluded patterns
        if (config.excludePatterns.some(pattern => {
          const regex = new RegExp(pattern.replace(/\*\*/g, '.*').replace(/\*/g, '[^/]*'));
          return regex.test(file);
        })) return false;
        // Filter out Windows reserved device names (nul, con, prn, aux, com1-9, lpt1-9)
        if (config.windowsReservedNames && config.windowsReservedNames.test(file)) return false;
        return true;
      })
      .map(({ status, file }) => {
        // Handle files with spaces by ensuring proper quoting
        // Remove any existing quotes and re-add them properly
        const cleanFile = file.replace(/^["']|["']$/g, '');
        return { status, file: cleanFile };
      });
    
    // Security check
    if (config.securityCheck) {
      if (!validateFiles(allFiles)) {
        process.exit(1);
      }
    }
    
    return allFiles;
  } catch (error) {
    log('Error getting git status:', 'red');
    log(error.message, 'red');
    process.exit(1);
  }
}

/**
 * Stage-then-Commit: stages the file, parses word-diff of staged vs HEAD, returns specific message.
 * Fixes ?? untracked files (git add runs first) and uses --only for per-file commits.
 * When dryRun=true, does NOT add; uses working-tree diff instead.
 */
function inferScope(filePath) {
  const p = (filePath || '').replace(/\\/g, '/');
  if (p.startsWith('frontend/')) return 'frontend';
  if (p.startsWith('backend/')) return 'python';
  if (p.startsWith('gateway/')) return 'gateway';
  if (p.startsWith('docs/')) return 'docs';
  if (p.startsWith('.github/')) return 'ci';
  if (p.startsWith('scripts/')) return 'ci';
  if (/^README\.md$/i.test(p) || p.toLowerCase().includes('readme')) return 'readme';
  if (/package(-lock)?\.json$/i.test(p) || /pnpm-lock\.yaml$/i.test(p) || /yarn\.lock$/i.test(p)) return 'deps';
  if (p.startsWith('.cursor/') || p.startsWith('.husky/') || p.endsWith('.env.example')) return 'config';
  return 'config';
}

function conventional(type, scope, subject) {
  const s = (subject || '').replace(/[\r\n]+/g, ' ').replace(/\s+/g, ' ').trim();
  // commitlint: subject max 72
  const trimmed = s.length > 72 ? s.slice(0, 69).trimEnd() + '…' : s;
  return `${type}(${scope}): ${trimmed}`;
}

function getSpecificMessage(filePath, status, dryRun) {
  const cwd = process.cwd();
  const opts = { encoding: 'utf8', cwd, maxBuffer: 1024 * 1024 };
  const scope = inferScope(filePath);

  try {
    // 1. Get status (for ??, A, D)
    let statusLine = status || '';
    if (!statusLine) {
      const statusResult = spawnSync('git', ['status', '--porcelain', '--', filePath], opts);
      statusLine = (statusResult.stdout || '').trim().split('\n')[0] || '';
    }

    // 2. Stage the file (skip in dry-run to avoid modifying index)
    if (!dryRun) {
      const addResult = spawnSync('git', ['add', '--', filePath], opts);
      if (addResult.error || addResult.status !== 0) {
        return conventional('chore', scope, `update ${path.basename(filePath)}`);
      }
    }

    // 3. Fallback for new (??) or staged new (A) or deleted (D)
    if (statusLine.startsWith('??') || statusLine.includes('A')) {
      return conventional('feat', scope, `add ${path.basename(filePath)}`);
    }
    if (statusLine.includes('D')) {
      return conventional('chore', scope, `remove ${path.basename(filePath)}`);
    }

    // 4. Get word-level diff (staged vs HEAD, or working tree when dry-run)
    const diffArgs = dryRun
      ? ['diff', '--word-diff=plain', '--', filePath]
      : ['diff', '--cached', '--word-diff=plain', '--', filePath];
    const diffResult = spawnSync('git', diffArgs, opts);
    const diff = (diffResult.stdout || '').toString();

    // 5. Regex to find the first specific code change [-removed-]{+added+}
    const match = diff.match(/\[-(.*?)-\]\{\+(.*?)\+\}/s);
    if (match) {
      const [, was, now] = match;
      const oldVal = (was || '').trim().substring(0, 50);
      const newVal = (now || '').trim().substring(0, 50);
      return conventional('chore', scope, `update ${path.basename(filePath)} (${oldVal} → ${newVal})`);
    }

    return conventional('refactor', scope, `adjust ${path.basename(filePath)}`);
  } catch (e) {
    return conventional('chore', scope, `update ${path.basename(filePath)}`);
  }
}

function getCommitMessage(file, status, dryRun) {
  const msg = getSpecificMessage(file, status, dryRun);
  return msg.replace(/[\r\n]+/g, ' ').replace(/\s+/g, ' ').trim().slice(0, 200);
}

function commitFile(file, status) {
  return new Promise((resolve, reject) => {
    if (config.securityCheck && isSensitiveFile(file)) {
      const err = `Security block: ${file} contains sensitive patterns.`;
      log(`❌ ${err}`, 'red');
      reject({ file, success: false, error: err });
      return;
    }

    const commitMessage = getCommitMessage(file, status, config.dryRun);

    if (config.dryRun) {
      log(`[DRY RUN] Would stage and commit: "${file}"\n  → Message: "${commitMessage}"`, 'yellow');
      resolve({ file, success: true, message: commitMessage });
      return;
    }

    const cwd = process.cwd();
    const spawnOpts = { encoding: 'utf8', cwd, maxBuffer: 1024 * 1024 };

    try {
      // getSpecificMessage already staged the file; commit only this file with --only
      const commitResult = spawnSync('git', ['commit', '-m', commitMessage, '--only', '--', file], spawnOpts);

      if (commitResult.error) {
        throw new Error(commitResult.error.message || 'git commit failed');
      }
      if (commitResult.status !== 0) {
        const err = (commitResult.stderr || commitResult.stdout || `git commit exit ${commitResult.status}`).trim();
        // "no changes added to commit" / "nothing to commit" = file has no diff vs HEAD; treat as skip (success)
        if (/no changes added to commit|nothing to commit/i.test(err)) {
          log(`⏭️  Skipped "${file}" (no changes vs HEAD)`, 'dim');
          resolve({ file, success: true, message: null, skipped: true });
          return;
        }
        throw new Error(err);
      }

      log(`✅ Committed: "${file}"`, 'green');
      resolve({ file, success: true, message: commitMessage });
    } catch (error) {
      const detail = error.message || String(error);
      log(`❌ Failed to commit "${file}": ${detail}`, 'red');
      reject({ file, success: false, error: detail });
    }
  });
}

function processCommitsInBatches(files) {
  const batches = [];
  for (let i = 0; i < files.length; i += config.maxConcurrentCommits) {
    batches.push(files.slice(i, i + config.maxConcurrentCommits));
  }
  
  const allResults = [];
  return batches.reduce((promise, batch, batchIndex) => {
    return promise.then(prevResults => {
      log(`\n📦 Processing batch ${batchIndex + 1}/${batches.length} (${batch.length} files)`, 'cyan');

      const promises = batch.map(({ file, status }) => commitFile(file, status));

      return Promise.allSettled(promises).then(results => {
        allResults.push(...results);
        const successful = results.filter(r => r.status === 'fulfilled' && r.value.success).length;
        const failed = results.filter(r => r.status === 'rejected' || (r.status === 'fulfilled' && !r.value.success)).length;

        log(`Batch ${batchIndex + 1} completed: ${successful} successful, ${failed} failed`,
            failed > 0 ? 'yellow' : 'green');

        return allResults;
      });
    });
  }, Promise.resolve([]));
}

function showSummary(results) {
  const allResults = results.flat();
  const successful = allResults.filter(r => r.status === 'fulfilled' && r.value.success);
  const failed = allResults.filter(r => r.status === 'rejected' || (r.status === 'fulfilled' && !r.value.success));
  
  log('\n📊 Summary:', 'bright');
  log(`Total files processed: ${allResults.length}`, 'blue');
  log(`Successfully committed: ${successful.length}`, 'green');
  log(`Failed: ${failed.length}`, failed.length > 0 ? 'red' : 'green');
  
  if (failed.length > 0) {
    log('\n❌ Failed files:', 'red');
    failed.forEach(result => {
      const file = result.status === 'fulfilled' ? result.value.file : result.reason.file;
      const error = result.status === 'fulfilled' ? result.value.error : result.reason.error;
      log(`  - "${file}": ${error}`, 'red');
    });
  }
}

// Main execution
async function main() {
  log('🚀 Parallel Git Commit Script', 'bright');
  log('================================', 'bright');
  
  // Parse command line arguments
  const args = process.argv.slice(2);
  if (args.includes('--dry-run')) {
    config.dryRun = true;
    log('🔍 DRY RUN MODE - No actual commits will be made', 'yellow');
  }
  
  if (args.includes('--help') || args.includes('-h')) {
    log('\nUsage: node parallel-commit.js [options]', 'cyan');
    log('Options:', 'cyan');
    log('  --dry-run              Show what would be committed without actually committing', 'cyan');
    log('  --no-security-check    Disable security checks (NOT RECOMMENDED)', 'cyan');
    log('  --warn-only            Warn about sensitive files but don\'t fail', 'cyan');
    log('  --help, -h             Show this help message', 'cyan');
    log('\nConfiguration:', 'cyan');
    log(`  Max concurrent commits: ${config.maxConcurrentCommits}`, 'cyan');
    log('  Message style: diff-parser (extracts specific changes from git diff)', 'cyan');
    log(`  Security check: ${config.securityCheck ? 'enabled' : 'disabled'}`, 'cyan');
    log(`  Fail on sensitive: ${config.failOnSensitive ? 'yes' : 'no (warn only)'}`, 'cyan');
    log(`  Exclude patterns: ${config.excludePatterns.join(', ')}`, 'cyan');
    log('\nSecurity:', 'yellow');
    log('  The script automatically blocks commits of sensitive files:', 'yellow');
    log('  - .env files (except .env.example)', 'yellow');
    log('  - Secret files (.secrets, credentials.json, etc.)', 'yellow');
    log('  - Certificate and key files', 'yellow');
    log('  - MCP config files with secrets', 'yellow');
    return;
  }
  
  if (args.includes('--no-security-check')) {
    config.securityCheck = false;
    log('⚠️  Security checks disabled - NOT RECOMMENDED!', 'yellow');
  }
  
  if (args.includes('--warn-only')) {
    config.failOnSensitive = false;
    log('⚠️  Warning mode: Will warn but not fail on sensitive files', 'yellow');
  }
  
  try {
    // Check if we're in a git repository
    execSync('git rev-parse --git-dir', { stdio: 'pipe' });
  } catch (error) {
    log('❌ Not in a git repository!', 'red');
    process.exit(1);
  }
  
  // Get changed files
  log('\n🔍 Scanning for changed files...', 'blue');
  const changedFiles = getChangedFiles();
  
  if (changedFiles.length === 0) {
    log('✅ No changes to commit', 'green');
    return;
  }
  
  log(`Found ${changedFiles.length} changed files:`, 'blue');
  changedFiles.forEach(({ file, status }) => {
    const statusIcon = status.includes('A') ? '🆕' : 
                      status.includes('M') ? '📝' : 
                      status.includes('D') ? '🗑️' : 
                      status.includes('R') ? '🔄' : '❓';
    log(`  ${statusIcon} "${file}" (${status})`, 'blue');
  });
  
  // Process commits
  log(`\n⚡ Processing commits (max ${config.maxConcurrentCommits} concurrent)...`, 'cyan');
  const results = await processCommitsInBatches(changedFiles);
  
  // Show summary
  showSummary(results);
  
  if (!config.dryRun && results.flat().some(r => r.status === 'fulfilled' && r.value.success)) {
    log('\n🎉 All commits completed!', 'green');
  }
}

// Handle uncaught errors
process.on('uncaughtException', (error) => {
  log(`\n💥 Uncaught Exception: ${error.message}`, 'red');
  process.exit(1);
});

process.on('unhandledRejection', (reason) => {
  log(`\n💥 Unhandled Rejection: ${reason}`, 'red');
  process.exit(1);
});

// Run the script
if (require.main === module) {
  main().catch(error => {
    log(`\n💥 Script failed: ${error.message}`, 'red');
    process.exit(1);
  });
}

module.exports = { main, getChangedFiles, commitFile, config };
