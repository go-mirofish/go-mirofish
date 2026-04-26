# Release scripts

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
