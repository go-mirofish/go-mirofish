# Contributing

- **[6-layer PR planning (narrative)](github-pr-6-layer.md)** — full guide (pairs with **PR template**); **PR template**: [`.github/PULL_REQUEST_TEMPLATE.md`](../../.github/PULL_REQUEST_TEMPLATE.md); **issue templates** (chooser): [`.github/ISSUE_TEMPLATE/`](../../.github/ISSUE_TEMPLATE/).
- **[Commit messages (Conventional Commits)](commit-messages.md)** — types, scopes, examples, and what each Husky hook runs.

## Dev tooling (root)

| Tool | Role |
| --- | --- |
| **Husky** (`.husky/`) | `commit-msg` → Commitlint; `pre-commit` → optional `go vet`, Python `py_compile` (uses `uv` or `backend/.venv`); `pre-push` → `go test` (if `gateway/`), `pytest` via `uv` or `.venv`. Hooks prepend `$HOME/.local/bin` and `$HOME/.cargo/bin` because Git often runs hooks with a minimal `PATH` (e.g. `uv: command not found`). |
| **Commitlint** (`commitlint.config.cjs`) | Conventional commits: required **type** + **scope**, subject max **72** chars. Types include `build` and `revert`. Scopes: `gateway`, `python`, `frontend`, `docs`, `readme`, `ci`, `config`, `deps`, `release`. Details: [commit-messages.md](commit-messages.md). |
| **Changesets** (`.changeset/config.json`) | Version PRs / changelog; run `npx changeset` after user-facing changes. Release workflow: `.github/workflows/release.yml`. |
| **Renovate** (`renovate.json`) | Weekly deps; **pinned AI packages** (`camel-oasis`, `camel-ai`, `zep-cloud`) are excluded from auto-updates. |

After cloning, run **`npm install`** at the repo root so `prepare` installs Husky and devDependencies (updates `package-lock.json` if needed).
