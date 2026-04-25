# Commit messages

Format: **`type(scope): subject`**

- **Subject** ≤ 72 characters.
- **Scope** is required and must be one of:

`gateway`, `python`, `frontend`, `docs`, `ci`, `config`, `deps`, `release`, `readme`

- **Types:** `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`, `perf`, `build`, `revert`

Examples:

- `feat(gateway): add readiness sub-checks`
- `fix(frontend): proxy error handling for cold gateway`
- `docs(readme): align quick start with make up + npm run dev`

Configuration: [commitlint.config.cjs](../../commitlint.config.cjs) at the repo root.
