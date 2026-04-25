# Stack ownership

**End state (COMPLETE):** Every control plane, simulation cycle, and HTTP surface is owned by **Go** (plus Vue, built for release or run via Vite in development). Python is not in the product path.

**Default developer workflow (single documented path):**

```bash
make up          # Go gateway in Docker on :3000 (API; dev image has no embedded Vue)
npm run dev      # Vite on :5173; proxies /api → :3000
```

Open the app at **http://127.0.0.1:5173**. There is no Python process, no Flask, no `backend/.venv`.

**Optional — all-in-one container (static UI baked in, no `npm run dev`):**

```bash
make up-release
```

Then use **http://localhost:3000** for both UI and API (for demos or CI-like runs, not the default dev loop).

**Stack:**

| Layer | Technology |
| --- | --- |
| Control plane | Go (gateway) |
| Simulation engine | Go native |
| API surface | Go (gateway) — all `/api/*` routes |
| UI (development) | Vue + Vite on :5173 |
| UI (release image) | Vue static `dist/` in `release` Docker stage |
| Docker (default) | `gateway` service, `target: dev` |
| Orchestration | Makefile + shell + Docker |

**Data directories:** runtime data for the Go gateway under `data/` (projects, reports, tasks, simulations). In Docker, a named volume `mirofish-data` mounts to `/app/data`.

**Rules for PRs:**

1. All new features ship in Go.
2. All HTTP API surface lives on the gateway (dev: typically `:3000`).
3. The Go-native simulation engine is the only supported worker runtime.
4. Default dev documentation must not imply a second “official” way to start the full stack (e.g. embedded Vue in the default `make up` image).
