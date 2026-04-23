# Git Commit Automation Guide

This repository includes a CLI for git commit management with parallel processing and security checks. Built with composable commands (cli-building skill) and optional Linear integration (linear-cli skill).

## Quick Start

### CLI (Recommended)

```bash
# Default: parallel commit with security checks
pnpm run commit
# or: node .github/version/scripts/commit/cli/index.js commit

# Preview what will be committed
pnpm run commit:dry

# Security check only
pnpm run security:check

# Linear issue trailer (requires linear CLI)
node .github/version/scripts/commit/cli/index.js linear
```

### CLI Commands

| Command | Description |
|---------|-------------|
| `commit` | Parallel commit with security checks (default) |
| `security` | Scan staged/working dir for sensitive files |
| `linear` | Print Linear issue trailer for commit messages |

### Legacy Scripts

```bash
# Bash version (no Node.js)
pnpm run commit:sh
pnpm run commit:sh:dry

# Single commit for all changes
pnpm run commit:all
```

## What the Scripts Do

1. **Scan for Changes**: Automatically detects all modified, added, or deleted files
2. **Security Checks**: **NEW** - Automatically blocks commits of sensitive files (.env, secrets, credentials, etc.)
3. **Generate Commit Messages**: Diff-parser extracts specific changes from `git diff --word-diff` (no LLM); Regex Router for .ts/.js, package.json, .css/.scss
4. **Parallel Processing**: Commits multiple files simultaneously for speed
5. **Smart Filtering**: Excludes build artifacts, logs, and temporary files
6. **Space-Safe Handling**: Properly handles files and directories with spaces using double quotes
7. **Error Handling**: Continues processing even if individual commits fail

## Security Features

The scripts now include **automatic security checks** to prevent committing sensitive files:

### Blocked Files (Will Fail Commit)
- `.env` files (except `.env.example`, `.env.template`, `.env.sample`)
- Secret files (`.secrets`, `credentials.json`, `service-account.json`)
- Certificate and key files (`.pem`, `.key`, `.cert`, `.p12`, `.pfx`, `.jks`)
- MCP config with secrets (`.cursor/mcp.json` - if it contains secrets)
- Backup files (`.env.backup`, `*.env.*.backup`)

### Allowed Files (Safe to Commit)
- `.env.example`
- `.env.template`
- `.env.sample`
- `.cursor/mcp.json.example`
- `.cursor/mcp.example.json`

### Security Options

```bash
# Default: Security checks enabled, fails on sensitive files
npm run commit

# Warn only (not recommended for production)
node scripts/parallel-commit.js --warn-only

# Disable security checks (NOT RECOMMENDED)
node scripts/parallel-commit.js --no-security-check
```

## Commit Message Generation (Diff-Parser, No LLM)

The script uses `git diff -U0 --word-diff` to extract specific changed "words" and generates highly specific messages without an LLM:

| Scenario | Example |
|----------|---------|
| Word-diff match | `useNetworks.ts – Updated signal to undefined, signal` |
| .ts/.js (Regex Router) | `page.tsx – Modified getExplorerUrl` |
| package.json | `package.json – Upgraded react to 18.2.0` |
| .css/.scss | `styles.css – Changed color to #fff` |
| New file | `feat: add README.md in docs` |
| Deleted | `remove: delete old-file.txt` |
| Renamed | `refactor: rename component.js in src/components` |
| Fallback | `update: modify script.js in hooks` |

## Configuration

### Adjust Concurrency
```bash
# Limit to 3 concurrent commits
bash scripts/parallel-commit.sh --max 3
```

### Exclude Files
Edit the `excludePatterns` in `scripts/parallel-commit.js`:
```javascript
excludePatterns: [
  'node_modules/**',
  '.git/**',
  '*.log',
  '*.tmp',
  '.DS_Store',
  'Thumbs.db'
]
```

## Workflow Examples

### Daily Development
```bash
# 1. Make your changes
# 2. Preview what will be committed
npm run commit:dry

# 3. Commit all changes
npm run commit

# 4. Push to remote
git push
```

### Large Refactoring
```bash
# For many files, use limited concurrency
bash scripts/parallel-commit.sh --max 2

# Or commit everything at once
npm run commit:all
```

### Safe Testing
```bash
# Always test first with dry run
npm run commit:dry

# Then commit for real
npm run commit
```

## Troubleshooting

### Common Issues

1. **Script not found**: Make sure you're in the repository root
2. **Permission denied**: Run `chmod +x scripts/parallel-commit.sh`
3. **Node.js not found**: Use the bash version instead
4. **No changes**: Script will tell you if there's nothing to commit

### Performance Tips

- Use `--max 3` for slower systems
- Use `npm run commit:all` for bulk changes
- Always run `--dry-run` first to preview

## Available Commands

| Command | Description |
|---------|-------------|
| `pnpm run commit` | CLI: parallel commit with security checks |
| `pnpm run commit:dry` | CLI: preview what would be committed |
| `pnpm run commit:cli` | CLI: full interface (commit \| security \| linear) |
| `pnpm run commit:sh` | Bash script version |
| `pnpm run commit:sh:dry` | Bash script dry run |
| `pnpm run commit:all` | Single commit for all changes |
| `pnpm run commit:auto` | CLI commit with max 3 concurrent |
| `pnpm run security:check` | CLI: security scan only |
| `pnpm run status` | Show git status |
| `pnpm run changes` | Show changed files |
| `pnpm run staged` | Show staged files |

## Security Best Practices

### Before Committing

Always run a security check:
```bash
npm run security:check
```

This will:
- ✅ Check if `.env` is in `.gitignore`
- ✅ Scan staged files for sensitive data
- ✅ Scan working directory for sensitive files
- ✅ Warn about potential security issues

### If Sensitive Files Are Detected

1. **Remove from staging**:
   ```bash
   git reset HEAD .env
   ```

2. **Ensure in .gitignore**:
   ```bash
   echo ".env" >> .gitignore
   ```

3. **If already committed**:
   ```bash
   # Remove from git history (use with caution)
   git filter-branch --force --index-filter \
     "git rm --cached --ignore-unmatch .env" \
     --prune-empty --tag-name-filter cat -- --all
   ```

4. **Revoke exposed credentials**:
   - GitHub: https://github.com/settings/tokens
   - Other services: Revoke immediately

### Example Files (Safe to Commit)

These files are **allowed** and safe to commit:
- ✅ `.env.example`
- ✅ `.env.template`
- ✅ `.env.sample`
- ✅ `.cursor/mcp.json.example`

## Benefits

- **Security**: Automatic detection and blocking of sensitive files
- **Speed**: Parallel processing commits multiple files simultaneously
- **Intelligence**: Automatic commit message generation
- **Safety**: Dry-run mode prevents accidental commits
- **Flexibility**: Multiple options for different use cases
- **Cross-platform**: Works on Windows, macOS, and Linux

## Troubleshooting Security Issues

### Issue: "Commit blocked for security reasons"

**Solution**: 
1. Check which files are sensitive: `npm run security:check`
2. Remove sensitive files from staging: `git reset HEAD <file>`
3. Ensure files are in `.gitignore`
4. Use `.env.example` as a template instead

### Issue: False positives (file incorrectly flagged)

**Solution**:
- If it's a legitimate example file, ensure it matches allowed patterns:
  - Must end with `.example`, `.template`, or `.sample`
  - Or be in `.cursor/` with `.example` in the name
- For other cases, you can temporarily use `--warn-only` (not recommended)

For detailed documentation, see `scripts/README.md`.
