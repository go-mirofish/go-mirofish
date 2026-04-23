#!/usr/bin/env node

/**
 * Security Check Script
 * Scans for sensitive files that should not be committed
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

// Colors for console output
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  cyan: '\x1b[36m'
};

// Security: Sensitive file patterns
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

// Security: Allowed example files
const ALLOWED_EXAMPLES = [
  /\.env\.example$/,
  /\.env\.template$/,
  /\.env\.sample$/,
  /\.env\.example\.local$/,
  /\.cursor\/mcp\.json\.example$/,
  /\.cursor\/mcp\.example\.json$/
];

function log(message, color = 'reset') {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

function isSensitiveFile(file) {
  // Check if it's an allowed example file first
  if (ALLOWED_EXAMPLES.some(pattern => pattern.test(file))) {
    return false;
  }
  
  // Check against sensitive patterns
  return SENSITIVE_PATTERNS.some(pattern => pattern.test(file));
}

function checkStagedFiles() {
  try {
    const output = execSync('git diff --cached --name-only', { encoding: 'utf8' });
    return output
      .split('\n')
      .filter(line => line.trim())
      .map(file => file.trim());
  } catch (error) {
    return [];
  }
}

function checkWorkingDirectory() {
  try {
    const output = execSync('git status --porcelain', { encoding: 'utf8' });
    return output
      .split('\n')
      .filter(line => line.trim())
      .map(line => {
        const status = line.substring(0, 2);
        const file = line.substring(3).trim();
        return { status, file };
      });
  } catch (error) {
    log('Error getting git status:', 'red');
    log(error.message, 'red');
    return [];
  }
}

function checkGitignore() {
  const gitignorePath = path.join(process.cwd(), '.gitignore');
  
  if (!fs.existsSync(gitignorePath)) {
    return { exists: false, hasEnv: false };
  }
  
  const content = fs.readFileSync(gitignorePath, 'utf8');
  const hasEnv = /^\.env$/m.test(content) || /^\.env\s*$/m.test(content);
  
  return { exists: true, hasEnv };
}

function main() {
  log('🔒 Security Check for Git Repository', 'bright');
  log('====================================', 'bright');
  
  // Check if in git repository
  try {
    execSync('git rev-parse --git-dir', { stdio: 'pipe' });
  } catch (error) {
    log('❌ Not in a git repository!', 'red');
    process.exit(1);
  }
  
  // Check .gitignore
  log('\n📋 Checking .gitignore...', 'blue');
  const gitignore = checkGitignore();
  if (!gitignore.exists) {
    log('⚠️  .gitignore file not found!', 'yellow');
  } else if (!gitignore.hasEnv) {
    log('⚠️  .gitignore may not properly exclude .env files', 'yellow');
    log('   Ensure .gitignore contains: .env', 'yellow');
  } else {
    log('✅ .gitignore properly configured for .env files', 'green');
  }
  
  // Check staged files
  log('\n🔍 Checking staged files...', 'blue');
  const stagedFiles = checkStagedFiles();
  const sensitiveStaged = stagedFiles.filter(file => isSensitiveFile(file));
  
  if (sensitiveStaged.length > 0) {
    log(`❌ Found ${sensitiveStaged.length} sensitive file(s) in staging area:`, 'red');
    sensitiveStaged.forEach(file => {
      log(`   - ${file}`, 'red');
    });
    log('\n⚠️  ACTION REQUIRED: Remove these files from staging:', 'yellow');
    log('   git reset HEAD <file>', 'yellow');
    process.exit(1);
  } else {
    log('✅ No sensitive files in staging area', 'green');
  }
  
  // Check working directory
  log('\n🔍 Checking working directory...', 'blue');
  const workingFiles = checkWorkingDirectory();
  const sensitiveWorking = workingFiles
    .filter(({ file }) => isSensitiveFile(file))
    .map(({ file, status }) => ({ file, status }));
  
  if (sensitiveWorking.length > 0) {
    log(`⚠️  Found ${sensitiveWorking.length} sensitive file(s) in working directory:`, 'yellow');
    sensitiveWorking.forEach(({ file, status }) => {
      log(`   - ${file} (${status})`, 'yellow');
    });
    log('\n💡 These files will be blocked if you try to commit them', 'cyan');
    log('   Ensure they are in .gitignore', 'cyan');
  } else {
    log('✅ No sensitive files detected in working directory', 'green');
  }
  
  // Summary
  log('\n📊 Security Check Summary:', 'bright');
  log(`   Staged files checked: ${stagedFiles.length}`, 'blue');
  log(`   Working directory files checked: ${workingFiles.length}`, 'blue');
  log(`   Sensitive files found: ${sensitiveStaged.length + sensitiveWorking.length}`, 
      (sensitiveStaged.length + sensitiveWorking.length) > 0 ? 'red' : 'green');
  
  if (sensitiveStaged.length === 0 && sensitiveWorking.length === 0) {
    log('\n✅ Security check passed!', 'green');
    process.exit(0);
  } else {
    log('\n❌ Security check failed!', 'red');
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = { main, isSensitiveFile, checkStagedFiles, checkWorkingDirectory, checkGitignore };

