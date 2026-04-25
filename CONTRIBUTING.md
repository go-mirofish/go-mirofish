# Contributing to go-mirofish

Thanks for helping improve **go-mirofish**: the Go gateway, Vue UI, and documentation for this stack.

## Where to start

| Topic | Link |
| --- | --- |
| **Contributing hub** (Husky, Changesets, Renovate) | [docs/contributing/README.md](docs/contributing/README.md) |
| **Commit messages** (Conventional Commits + Commitlint) | [docs/contributing/commit-messages.md](docs/contributing/commit-messages.md) |
| **Local setup** | [docs/getting-started/installation.md](docs/getting-started/installation.md) — **`make up`** (Docker gateway) + **`npm run dev`** (Vite) |
| **PR template** | [.github/PULL_REQUEST_TEMPLATE.md](.github/PULL_REQUEST_TEMPLATE.md) |
| **Issue templates** | [.github/ISSUE_TEMPLATE/](.github/ISSUE_TEMPLATE/) |

## One-time setup after clone

1. **Root JS + Husky:** From the repo root, run **`npm install`**. This installs root devDependencies (Husky, Commitlint, Changesets).
2. **Frontend deps:** `npm run setup` (or `npm install --prefix frontend`) for Vite/Vue work.
3. **Docker:** Install [Docker](https://docs.docker.com/get-docker/) with Compose for `make up`.

There is **no** Python backend or `backend/.venv` in the product path.

**Gateway on the host:** `make gateway` / `npm run gateway` exist only for **debugging** the Go binary without Docker. They are **not** an alternate “official” dev path. The documented default is **`make up`** (Docker) **+** **`npm run dev`** (Vite).

## Git hooks (Husky)

| Hook | What runs |
| --- | --- |
| `commit-msg` | [Commitlint](commitlint.config.cjs): `type(scope): subject` (see [commit-messages.md](docs/contributing/commit-messages.md)) |
| `pre-commit` | `go vet ./...` under `gateway/` |
| `pre-push` | `go test ./...` under `gateway/` |

`pre-push` runs the full Go gateway test suite. If a hook is too strict for a one-off, coordinate with maintainers (e.g. `git push --no-verify` is a last resort).

## Quick rules

1. **Commits** must pass Commitlint: required **type** and **scope**, subject ≤ **72** characters. Scopes are listed in [commit-messages.md](docs/contributing/commit-messages.md).
2. Prefer **small, reviewable** changes; **AGPL-3.0** applies. Preserve license headers and share-alike expectations where relevant.
3. Prefer **additive** changes that stay easy to merge.

## Questions

Open an issue via [.github/ISSUE_TEMPLATE/](.github/ISSUE_TEMPLATE/) or see the **Community** / support section in [README.md](README.md).
