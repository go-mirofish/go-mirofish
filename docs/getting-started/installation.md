# Installation

For the public app ([gomirofish.vercel.app](https://gomirofish.vercel.app); **go.mirofish.ai** pending DNS), from zero to a running stack.

> [!NOTE]
> **Prerequisites**
> - **Python 3.11.x** on your machine when you run the backend from source (not 3.12+; backend deps such as `camel-oasis` do not install on newer Pythons yet; see `requires-python` in `backend/pyproject.toml`).
> - **Node.js 18+** if you use the `npm` dev workflow (`npm run dev`).
> - **LLM + Zep** keys in `.env` for the default cloud path, unless you go fully local. See [Ollama](../configuration/ollama.md) and [OpenAI-compatible providers](../configuration/providers.md).
> - **[uv](https://docs.astral.sh/uv/)** is optional; the repo can create **`backend/.venv`** with `pip` via `npm run setup:backend` instead.

> [!IMPORTANT]
> Use **`backend/.venv` only**. Do not use a separate `.venv` at the **repository root**. Wrong Python version? Delete `backend/.venv` and run `npm run setup:backend` again.

## System requirements (summary)

| Item | Notes |
| --- | --- |
| OS | Windows, macOS, or Linux (64-bit) |
| RAM | 4GB+; 8GB+ for heavier runs |
| Python | **3.11.x** for local backend (see above) |
| Node | 18+ for `npm run dev` / `npm run build` |
| Go | 1.21+ only to **build** the gateway (`gateway/bin/`) or use `./start.sh` / `start.bat` |
| Docker | Optional: full stack without local Node/Go (see below) |

## 1. Get the code

```bash
git clone https://github.com/go-mirofish/go-mirofish.git
cd go-mirofish
```

## 2. Configure `.env`

```bash
cp .env.example .env
```

Set at least **`LLM_API_KEY`** and **`ZEP_API_KEY`**. Optional **`LLM_BOOST_*`** lines: if you do not use them, **leave them out** of `.env` so the backend does not read empty boost config.

## 3. Run with Docker Compose

From the repo root:

```bash
docker compose up --build -d
```

- UI / gateway: [http://localhost:3000](http://localhost:3000)  
- Backend (in Compose): `http://backend:5001`  
Compose reads `.env` at the project root.

## 4. Run from source (development)

**Install once (Node, frontend, backend):**

```bash
npm run setup:all
```

**Run backend + frontend:**

```bash
npm run dev
```

- Backend: [http://localhost:5001](http://localhost:5001) (`npm run backend`)  
- Frontend (Vite): [http://localhost:3000](http://localhost:3000) (`npm run frontend`)  

`npm run backend` uses `scripts/dev/run-backend.cjs`: it prefers **`backend/.venv`**, else **`uv run`** if `uv` is on your `PATH`. You usually do **not** need to `activate` the venv for day-to-day work.

### Python, uv, and venv

| Task | Command |
| --- | --- |
| Create or refresh backend deps (from **repo root**) | `npm run setup:backend` runs `uv sync` in `backend/` if `uv` exists, otherwise **pip** into `backend/.venv` (see `scripts/dev/setup-backend.cjs`) |
| With **uv** only | `cd backend && uv sync` (if needed: `uv python install 3.11`) |
| Run the API | `npm run backend` or `cd backend && uv run python run.py` |
| **Activate** the venv (only for a manual shell) | See table below. Paths are always `backend/.venv` |

| Shell | Activate |
| --- | --- |
| macOS / Linux (bash, zsh) | `source backend/.venv/bin/activate` |
| Windows (Git Bash) | `source backend/.venv/Scripts/activate` |
| Windows (cmd) | `backend\.venv\Scripts\activate.bat` |
| Windows (PowerShell) | `backend\.venv\Scripts\Activate.ps1` |

After activation, from **`backend/`**: e.g. `python -m pytest`, `python run.py`.

> [!NOTE]
> **Git hooks (Husky)** use the same rule as the CLI: `uv` on `PATH` first, else `backend/.venv`. Hooks add `$HOME/.local/bin` and `$HOME/.cargo/bin` to `PATH`. If a hook says the env is missing, run `npm run setup:backend` from the repo root, then `npm install` (for Husky) if you have not already.

## 5. Local gateway + backend (`./start.sh` / `start.bat`)

No Node dev server: build the **Go** gateway, build the **frontend** for `frontend/dist/`, then start.

```bash
npm run build:gateway
npm run build
```

**Linux / macOS:** `./start.sh`  
**Windows:** `.\start.bat`

Expects: Python **3.11.x**, binary at `gateway/bin/go-mirofish-gateway` (or `gateway\bin\go-mirofish-gateway.exe`), and `frontend/dist/index.html`. Build the gateway manually if you prefer:

The Go module lives under **`gateway/`** (not the repo root). From the **repo root**:

```bash
mkdir -p gateway/bin
go -C gateway build -o bin/go-mirofish-gateway ./cmd/mirofish-gateway
```

**Windows (PowerShell),** from the repo root:

```powershell
New-Item -ItemType Directory -Force -Path gateway/bin | Out-Null
go -C gateway build -o bin/go-mirofish-gateway.exe ./cmd/mirofish-gateway
```

Or run **`npm run build:gateway`** (writes to `gateway/bin/`).

## Troubleshooting

| Symptom | What to do |
| --- | --- |
| `camel-oasis` / “no matching distribution” | You are on **Python 3.12+**. Install **3.11.x**, remove `backend/.venv`, run `npm run setup:backend`. |
| `uv` is not recognized | **Optional:** use `npm run setup:backend` (see [Python, uv, and venv](#python-uv-and-venv)). |
| `vite` is not recognized | Run `npm run setup` or `npm install` at root and `npm install --prefix frontend` so `frontend/node_modules` exists. |
| Prebuilt gateway missing | `npm run build:gateway` or `go build` as above; ensure `frontend/dist` exists (`npm run build`). |
| Wrong venv path | Only **`backend/.venv`**. See [Python, uv, and venv](#python-uv-and-venv). |

## Verify

1. Open [http://localhost:3000](http://localhost:3000) (dev or gateway path).  
2. Check health: [http://localhost:5001/health](http://localhost:5001/health) (dev backend) or [http://localhost:3000/health](http://localhost:3000/health) (through the gateway when using `start.sh`).

## Next steps

- [Ollama (local LLM)](../configuration/ollama.md)  
- [OpenAI-compatible providers](../configuration/providers.md)  
- Deeper **.env** reference on [gomirofish.vercel.app](https://gomirofish.vercel.app) (and go.mirofish.ai when live) as docs expand  

## Examples and benchmarks

The repo now ships a local-first example runner:

```bash
go run ./gateway/cmd/go-mirofish-examples --list
```

Common commands:

```bash
go run ./gateway/cmd/go-mirofish-examples --all --smoke-only --profile small
go run ./gateway/cmd/go-mirofish-examples --all --bench-only --profile medium
go run ./gateway/cmd/go-mirofish-examples --example product-launch-war-room --profile medium
go run ./gateway/cmd/go-mirofish-examples --compare docs/bundled-benchmarks/product-launch__small__latest.json,docs/bundled-benchmarks/literary-sim__small__latest.json
```

Outputs are written under:

- `examples/*/artifacts/<profile>/`
- `benchmark/results/` (local runs; often gitignored)
- `docs/bundled-benchmarks/` — committed short-name JSON for the in-app benchmark report (see `docs/bundled-benchmarks/README.md`)
