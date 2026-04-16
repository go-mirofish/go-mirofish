# Commit messages (Conventional Commits)

Commits must satisfy the **`commit-msg`** Husky hook, which runs **Commitlint** (`commitlint.config.cjs` at the repo root).

## Format

```
type(scope): short description

[optional body]

[optional footer]
```

- Use the **imperative mood** in the subject (`add`, not `added`).
- Keep the **subject ≤ 72** characters.
- **Type** and **scope** are **required** (scope must be one of the values below).

## Valid types

| Type | When to use |
| --- | --- |
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `refactor` | Code change that is neither a fix nor a feature |
| `test` | Adding or updating tests |
| `chore` | Maintenance, tooling, housekeeping |
| `ci` | CI/CD workflow changes |
| `perf` | Performance improvement |
| `build` | Build system or packaging changes |
| `revert` | Reverts a previous commit |

## Valid scopes for go-mirofish

| Scope | What it covers |
| --- | --- |
| `gateway` | Go reverse proxy / gateway layer |
| `python` | Python backend (`backend/`) |
| `frontend` | Vue frontend (`frontend/`) |
| `docs` | Documentation site / guides under `docs/` |
| `readme` | Root `README.md`, `README-ZH.md`, or landing copy |
| `ci` | GitHub Actions and automation |
| `config` | Tooling config (e.g. Commitlint, Renovate, Husky), `.env.example` |
| `deps` | Dependency bumps (npm, uv, Go modules) |
| `release` | Release / Changesets / versioning flow |

## Examples that pass

```
feat(gateway): add health check endpoint at /health

fix(python): resolve simulation IPC timeout on slow machines

docs(docs): update Ollama setup in installation guide

readme(readme): update quick start for local LLM setup

chore(deps): bump Go version to 1.22

ci(release): add ARM64 binary build to release workflow

refactor(gateway): extract proxy handler into separate package

test(gateway): add integration test for /api/graph proxy route
```

## Examples that fail

```
# No type
updated the readme

# Wrong shape (capital letter, no colon)
Fix Bug

# Missing description
feat(gateway):

# Type not in the allowed list
update(gateway): something

# Missing or invalid scope
feat: add feature

feat(infra): add feature
```

## Hooks (what runs when)

| Hook | What it does |
| --- | --- |
| **`commit-msg`** | **Commitlint** — validates type, scope, subject length, and conventional format. |
| **`pre-commit`** | If `gateway/` exists: `go vet ./...`. For staged `*.py` files: `python -m py_compile` via `uv` from `backend/`. |
| **`pre-push`** | If `gateway/` exists: `go test ./...`. In `backend/`: `uv run pytest` (exit code5 = no tests collected is treated as skip). |

So: **message format** is only enforced on **`commit-msg`**. **`pre-commit`** does **not** run `go test` or `pytest`; those run on **`pre-push`**.

## Terminal one-liner

```bash
git commit -m "feat(gateway): add reverse proxy for /api/simulation"
```

## Editor tip

In VS Code, the **Conventional Commits** extension gives a guided commit builder so you do not have to memorize types and scopes.

## See also

- [Contributing index](README.md) — Husky, Changesets, Renovate overview.
