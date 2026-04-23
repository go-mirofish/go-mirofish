# Contributing

Hub for contributor docs. The repo root **[CONTRIBUTING.md](../../CONTRIBUTING.md)** is the short entry point GitHub highlights for new contributors.

- **issue templates** (chooser): [`.github/ISSUE_TEMPLATE/`](../../.github/ISSUE_TEMPLATE/).
- **[Commit messages (Conventional Commits)](commit-messages.md):** types, scopes, examples, and what each Husky hook runs.

## Dev tooling (root)

| Tool | Role |
| --- | --- |
| **Husky** (`.husky/`) | `commit-msg` → Commitlint; `pre-commit` → optional `go vet`, Python `py_compile`; `pre-push` → `go test` (if `gateway/`), `pytest`. Python: **`uv` first, else `backend/.venv`**, per [Installation](../getting-started/installation.md#python-uv-and-venv). Missing env → **`npm run setup:backend`**. Hooks prepend `$HOME/.local/bin` and `$HOME/.cargo/bin` for a minimal hook `PATH`. |
| **Commitlint** (`commitlint.config.cjs`) | Conventional commits: required **type** + **scope**, subject max **72** chars. Types include `build` and `revert`. Scopes: `gateway`, `python`, `frontend`, `docs`, `readme`, `ci`, `config`, `deps`, `release`. Details: [commit-messages.md](commit-messages.md). |
| **Changesets** (`.changeset/config.json`) | Version PRs / changelog; run `npx changeset` after user-facing changes. Release workflow: `.github/workflows/release.yml`. |
| **Renovate** (`renovate.json`) | Weekly deps; **pinned AI packages** (`camel-oasis`, `camel-ai`, `zep-cloud`) are excluded from auto-updates. |

After cloning, run **`npm install`** at the repo root so `prepare` installs Husky and devDependencies (updates `package-lock.json` if needed).
