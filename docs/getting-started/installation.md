# Installation

**Canonical local development (only path documented as default):**

1. **`make up`** — builds and runs the **Go gateway in Docker** on [http://127.0.0.1:3000](http://127.0.0.1:3000) (API, health, readiness, metrics). The default image does **not** embed the Vue app; the container expects your local Vite dev server.
2. **`npm run dev`** (from the **repository root**) — starts **Vite** on [http://127.0.0.1:5173](http://127.0.0.1:5173) and proxies `/api`, `/health`, `/ready`, and `/metrics` to the gateway on :3000.

**Use the UI at :5173** during development. The gateway on :3000 is the API; it can also reverse-proxy browser traffic to Vite if you open :3000, but the documented workflow is Vite on :5173.

> [!NOTE]
> **Stack:** Go gateway (control plane, API, native simulation). Vue via local Vite in dev. There is no Python backend, no Flask, no `backend/.venv`.

**Optional all-in-one Docker image (static UI inside the container, no local Node):**  
`docker compose -f docker-compose.release.yml up -d --build` — then open [http://localhost:3000](http://localhost:3000) for both UI and API. Use for demos or hosts without a Node toolchain, not the default dev loop.

---

## 1. Clone the repository

```bash
git clone https://github.com/go-mirofish/go-mirofish.git
cd go-mirofish
```

## 2. Create `.env`

```bash
cp .env.example .env
```

| Variable | Required? | Notes |
| --- | --- | --- |
| `LLM_API_KEY` | Yes (default cloud path) | For the LLM route. |
| `ZEP_API_KEY` | Yes (default cloud path) | For graph memory. |
| `VITE_GATEWAY_PROXY_TARGET` | Optional | Defaults to `http://127.0.0.1:3000` in `frontend/vite.config.js` — must match the Docker gateway port. |

> [!NOTE]
> Optional lines you are not using should be left out of `.env` rather than set to empty.

## 3. Install frontend dependencies (one-time, for `npm run dev`)

```bash
npm run setup
# or: npm install --prefix frontend
```

**Requirements:** [Docker](https://docs.docker.com/get-docker/) with Compose, and **Node 18+** for local Vite.

## 4. Start the gateway in Docker

```bash
make up
```

| Command | What it does |
| --- | --- |
| `make up` | `docker compose up -d --build` — `target: dev` (gateway only). |
| `make down` / `make logs` | Stop the stack or follow logs. |
| `make up-release` | Same as `docker compose -f docker-compose.release.yml up -d --build` — all-in-one image with static UI (not the default dev path). |

| URL | Role |
| --- | --- |
| [http://127.0.0.1:3000/health](http://127.0.0.1:3000/health) | Gateway health (`"stack":"go"`, `"python_backend":"removed"`) |
| [http://127.0.0.1:3000/ready](http://127.0.0.1:3000/ready) | Readiness |

## 5. Start the local frontend (second terminal)

```bash
npm run dev
```

Open **[http://127.0.0.1:5173](http://127.0.0.1:5173)**. API calls from the app go through Vite’s proxy to the Docker gateway on :3000.

| Shortcut | Same as |
| --- | --- |
| `make dev` | `bash scripts/dev/frontend.sh` (Vite only) |
| `make up-release` | All-in-one Docker image (`docker-compose.release.yml`) — static UI in container |
| `make gateway` | **Non-default:** local Go gateway via `scripts/dev/run-gateway.cjs` for debugging only. The documented workflow is **`make up`** (Docker). |

## 6. (Non-default) Run the Go gateway on the host

**This is not the canonical development path.** Use only for stepping through the gateway in a debugger. The supported API surface for day-to-day work is **`make up`**.

After `cp .env`, `make gateway` runs the gateway on the host with `GATEWAY_PORT=3000` and `FRONTEND_DEV_URL=http://127.0.0.1:5173`. Never run Docker `make up` and `make gateway` at the same time on :3000.

---

## Troubleshooting

| Symptom | Fix |
| --- | --- |
| `Network Error` / API failures in the browser | Ensure `make up` succeeded and [http://127.0.0.1:3000/health](http://127.0.0.1:3000/health) returns JSON. Then run `npm run dev`. |
| Port **3000** in use | Stop the other process or change the host port in `docker-compose.yml` and set `VITE_GATEWAY_PROXY_TARGET` to match. |
| Port **5173** in use | Set a different Vite port: `npm run dev -- --port 5174` and set `FRONTEND_DEV_URL` in Docker to `http://host.docker.internal:5174` (see `docker-compose.yml`). |
| `vite` not found | `npm run setup` from the repo root. |
| Linux: gateway cannot reach Vite | `host.docker.internal` is set via `extra_hosts` in `docker-compose.yml`. |

## Verify

| Check | URL |
| --- | --- |
| UI (dev) | [http://127.0.0.1:5173](http://127.0.0.1:5173) |
| Gateway health | [http://127.0.0.1:3000/health](http://127.0.0.1:3000/health) |

## Benchmarks

Requires the gateway listening (after **`make up`**; `make gateway` only if you deliberately use a host-side gateway):

```bash
make benchmark-run    # BENCH_BASE_URL default http://127.0.0.1:3000
make benchmark-live   # local gateway + built frontend; writes under benchmark/results/
```

| Make target | What it does |
| --- | --- |
| `make benchmark-run` | Load/stress/soak against `BENCH_BASE_URL` |
| `make test-all` | `go vet` + all Go tests |

Artifacts: `benchmark/results/benchmarks/benchmark.json`, `benchmark/results/api-wiring/api-wiring-report.json`, and `docs/report/benchmark-report.md` (from `make benchmark-live` / `go run ./cmd/mirofish-hybrid live-benchmark`) when generated.

## Next steps

- [Ollama (local LLM)](../configuration/ollama.md)
- [OpenAI-compatible providers](../configuration/providers.md)
