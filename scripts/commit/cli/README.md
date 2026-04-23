# hyperagent-commit CLI

Composable CLI for git commits with security checks. Uses cli-building (async-first, composable commands) and optional linear-cli integration.

## Commands

- **commit** (default) — Parallel commit with security checks
- **security** — Scan for sensitive files only
- **linear** — Print Linear issue trailer (requires `linear` CLI on PATH)

## Usage

```bash
# From repo root
node .github/version/scripts/commit/cli/index.js [command] [options]

# Or via pnpm
pnpm run commit          # commit (default)
pnpm run commit:dry     # commit --dry-run
pnpm run security:check # security
pnpm run commit:cli     # show usage
```

## Options (commit)

- `--dry-run` — Preview without committing
- `--no-security-check` — Disable security checks (not recommended)
- `--warn-only` — Warn on sensitive files, do not fail
- `--max <n>` — Max concurrent commits (default: 5)

## Linear Integration

When `linear` CLI is installed, use `linear issue id` to resolve the current issue from the branch name (e.g. `feature/ENG-123`). Append the output to commit messages for traceability.

```bash
node .github/version/scripts/commit/cli/index.js linear
# Output: Linear-issue: ENG-123
```

## NO_COLOR

Respects `NO_COLOR` for CI and accessibility.
