# Release scripts

## Step-by-step flow (repeat each time you ship doc + code)

Run in this order:

1. **`npm run release`** — release gate (gateway `go test ./...`). **Fix failures before step 2.**
2. **Changelog** — add a new top section (pick the next version and today’s date):
   - `node scripts/release/update-changelog.cjs --version vX.Y.Z --date YYYY-MM-DD`
   - or: `npm run changelog:release -- --version vX.Y.Z --date YYYY-MM-DD`  
   **Do not** invent a version higher than the git tag you will tag on GitHub unless you intend to catch up tags later.
3. **`npm run commit`** — commits **each changed file** as its own commit (parallel commit script). For one combined commit instead: `git add -A` then `git commit -m "type(scope): message"`.

| Command | What it does |
|--------|---------------|
| `npm run release` | Run gateway tests (`go test ./...` in `gateway/`; uses Python script if `python3` is available). |
| `npm run changelog:release` | Run `node scripts/release/update-changelog.cjs` (see below). |
| `npm run release:notes` | Run `node scripts/release/extract-release-notes.cjs` (see below). |
| `bash scripts/release/release.sh` | Same as `npm run release` (default mode). |
| `bash scripts/release/release.sh package` | Gateway test gate (same as default). |
| `bash scripts/release/release.sh changelog` | Changelog update script. |
| `bash scripts/release/release.sh notes` | Extract a release section from `CHANGELOG.md`. |
| `node scripts/release/update-changelog.cjs --version vX.Y.Z` | Append a release block to `CHANGELOG.md` from git log since the latest tag. Add `--date YYYY-MM-DD` and/or `--dry-run`. |
| `node scripts/release/extract-release-notes.cjs --tag vX.Y.Z` | Print the `## vX.Y.Z` section from `CHANGELOG.md`. Add `--output path` to write a file. |
| `python3 scripts/release/package_gateway_release.py` | Direct gateway test run (used by `release.sh`). |
