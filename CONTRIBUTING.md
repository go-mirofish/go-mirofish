# Contributing to go-mirofish

Thanks for helping improve **go-mirofish**. This repository focuses on the **local-first go-mirofish product surface** and its runtime, UI, tooling, and documentation.

## Where to start

| Topic | Link |
| --- | --- |
| **Index** (templates, hooks, tooling) | [docs/contributing/README.md](docs/contributing/README.md) |
| **6-layer PR planning** (narrative) | [docs/contributing/github-pr-6-layer.md](docs/contributing/github-pr-6-layer.md) |
| **Commits** (Conventional Commits + Commitlint) | [docs/contributing/commit-messages.md](docs/contributing/commit-messages.md) |
| **Local setup** | [docs/getting-started/installation.md](docs/getting-started/installation.md) (includes Python, **uv**, and `backend/.venv`; [see section](docs/getting-started/installation.md#python-uv-and-venv)) |
| **PR body template** | [.github/PULL_REQUEST_TEMPLATE.md](.github/PULL_REQUEST_TEMPLATE.md) |
| **New issues** (chooser) | [.github/ISSUE_TEMPLATE/](.github/ISSUE_TEMPLATE/) |

## Quick rules

1. **Commits** must pass Commitlint: `type(scope): subject` (≤72 chars). See [commit-messages.md](docs/contributing/commit-messages.md).
2. After clone, run **`npm install`** at the repo root so **Husky** installs (`commit-msg`, `pre-commit`, `pre-push`). For Python, run **`npm run setup:backend`** (or `cd backend && uv sync` with [uv](https://docs.astral.sh/uv/)). [Installation](docs/getting-started/installation.md#python-uv-and-venv).
3. **AGPL-3.0** applies; keep notices and compatibility with derivative works in mind.
4. Prefer **additive** changes that stay easy to merge with upstream where that’s a project goal.

## Questions

Open an issue using [.github/ISSUE_TEMPLATE/](.github/ISSUE_TEMPLATE/) or ask in [Discord](http://discord.gg/ePf5aPaHnA) (see [README.md](README.md)).
