# Contributing

- **[6-layer planning (narrative)](github-issues-6-layer.md)** — full guide; **PR template** (six layers): [`.github/PULL_REQUEST_TEMPLATE.md`](../../.github/PULL_REQUEST_TEMPLATE.md); **issue template** (common uses): [`.github/ISSUE_TEMPLATE.md`](../../.github/ISSUE_TEMPLATE.md).
- **[Commit messages (Conventional Commits)](commit-messages.md)** — types, scopes, examples, and what each Husky hook runs.

## Dev tooling (root)

| Tool | Role |
| --- | --- |
| **Husky** (`.husky/`) | `commit-msg` → Commitlint; `pre-commit` → optional `go vet`, Python `py_compile` on staged files; `pre-push` → `go test` (if `gateway/`), `pytest` in `backend/`. |
| **Commitlint** (`commitlint.config.cjs`) | Conventional commits: required **type** + **scope**, subject max **72** chars. Types include `build` and `revert`. Scopes: `gateway`, `python`, `frontend`, `docs`, `readme`, `ci`, `config`, `deps`, `release`. Details: [commit-messages.md](commit-messages.md). |
| **Changesets** (`.changeset/config.json`) | Version PRs / changelog; run `npx changeset` after user-facing changes. Release workflow: `.github/workflows/release.yml`. |
| **Renovate** (`renovate.json`) | Weekly deps; **pinned AI packages** (`camel-oasis`, `camel-ai`, `zep-cloud`) are excluded from auto-updates. |

After cloning, run **`npm install`** at the repo root so `prepare` installs Husky and devDependencies (updates `package-lock.json` if needed).
