# Contributing hub

- **Local development:** [Installation guide](../getting-started/installation.md) — **`make up`** (Docker gateway) then **`npm run dev`** (Vite). No local Python backend.
- **Commit messages:** [commit-messages.md](commit-messages.md) — Conventional Commits with required **scope** (enforced by Commitlint).
- **Husky hooks:** See [CONTRIBUTING.md](../../CONTRIBUTING.md) at the repo root for `pre-commit` / `pre-push` behavior.
- **Changesets / release:** Root `package.json` scripts `release`, `changelog:release`; [Changesets](https://github.com/changesets/changesets) when cutting versions.
