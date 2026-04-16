# Installation

This page is the **Getting Started → Installation** chapter for [go.mirofish.ai](https://go.mirofish.ai). It assumes you have never run MiroFish.

## System requirements

| Requirement | Minimum | Notes |
| --- | --- | --- |
| OS | Windows, macOS, or Linux | 64-bit recommended |
| RAM | 4GB+ | 8GB+ for comfortable multi-agent runs |
| Python | 3.11–3.12 | Used by the backend |
| Node.js | 18+ | **Only for the current dev workflow** (`npm run dev`) |
| [uv](https://docs.astral.sh/uv/) | Latest | Recommended Python env + runner for the backend |
| Docker (optional) | Docker Engine + Compose plugin | Alternative to local Node/Python setup |

You also need valid **LLM** and **Zep** credentials unless you use a fully local LLM path (see [Ollama setup](../configuration/ollama.md)).

## 1. Get the code

```bash
git clone https://github.com/go-mirofish/go-mirofish.git
cd go-mirofish
```

## 2. Configure environment variables

```bash
cp .env.example .env
```

Edit `.env`. At minimum set:

- **`LLM_API_KEY`** — key for your OpenAI-compatible LLM endpoint  
- **`ZEP_API_KEY`** — key for [Zep Cloud](https://www.getzep.com/) graph memory  

Optional keys in `.env.example` (for example **`LLM_BOOST_*`**) are **acceleration** slots. If you do not use them, **omit those lines** entirely so the backend does not try to read empty boost config.

## 3. Run with Docker Compose

From the repo root (same folder as `docker-compose.yml`):

```bash
docker compose up -d
```

- Frontend: [http://localhost:3000](http://localhost:3000)  
- Backend API: [http://localhost:5001](http://localhost:5001)  

Compose reads `.env` from the project root. Adjust ports in `docker-compose.yml` only if they clash with other services.

## 4. Run from source (development)

Install dependencies once:

```bash
npm run setup:all
```

Start backend + frontend together:

```bash
npm run dev
```

**URLs** are the same as in the Docker section.

### Run services separately

```bash
npm run backend
```

```bash
npm run frontend
```

## 5. Planned one-command startup (`./start.sh`)

The go-mirofish roadmap includes a **single shell entrypoint** that starts the **Go gateway** and **Python** stack using a **prebuilt Go binary** (no local Go toolchain, no Node for the supported production-style path). That script is **not guaranteed to exist yet** in every checkout; until it appears in the repo root, treat **Docker** or **`npm run dev`** as the supported paths above.

## Verify

1. Open the UI at [http://localhost:3000](http://localhost:3000).  
2. Confirm the API responds, for example by loading the app’s graph or health views (exact checks depend on the release you are on).  

## Next steps

- [Ollama setup (local LLM)](../configuration/ollama.md)  
- Full **.env** reference and cloud LLM providers (coming on go.mirofish.ai under **Configuration**)
