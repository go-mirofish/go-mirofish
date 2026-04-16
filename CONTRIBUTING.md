# Contributing to go-mirofish

Thanks for helping improve **go-mirofish**. This repo is a **lightweight, local-first** fork of [MiroFish](https://github.com/666ghj/MiroFish); use **this repository** for fork-specific work and **upstream** for the original product direction.

## Where to start

| Topic | Link |
| --- | --- |
| **Index** (templates, hooks, tooling) | [docs/contributing/README.md](docs/contributing/README.md) |
| **6-layer PR planning** (narrative) | [docs/contributing/github-pr-6-layer.md](docs/contributing/github-pr-6-layer.md) |
| **Commits** (Conventional Commits + Commitlint) | [docs/contributing/commit-messages.md](docs/contributing/commit-messages.md) |
| **Local setup** | [docs/getting-started/installation.md](docs/getting-started/installation.md) |
| **PR body template** | [.github/PULL_REQUEST_TEMPLATE.md](.github/PULL_REQUEST_TEMPLATE.md) |
| **New issues** (chooser) | [.github/ISSUE_TEMPLATE/](.github/ISSUE_TEMPLATE/) |

## Quick rules

1. **Commits** must pass Commitlint: `type(scope): subject` (≤72 chars). See [commit-messages.md](docs/contributing/commit-messages.md).
2. After clone, run **`npm install`** at the repo root so **Husky** installs (`commit-msg`, `pre-commit`, `pre-push`). If `uv` is missing in Git’s hook `PATH`, ensure **`backend/.venv`** exists (`cd backend && uv sync`) or install [uv](https://docs.astral.sh/uv/).
3. **AGPL-3.0** applies; keep notices and compatibility with derivative works in mind.
4. Prefer **additive** changes that stay easy to merge with upstream where that’s a project goal.

## Questions

Open an issue using [.github/ISSUE_TEMPLATE/](.github/ISSUE_TEMPLATE/) or ask in [Discord](http://discord.gg/ePf5aPaHnA) (see [README.md](README.md)).
