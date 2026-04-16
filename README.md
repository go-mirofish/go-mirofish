<div align="center">

<img src="./static/image/go-mirofish-thumbnail.png" alt="go-mirofish logo" width="55%"/>

**MiroFish, lightweight and local-first**

[![GitHub Stars](https://img.shields.io/github/stars/go-mirofish/go-mirofish?style=flat-square&color=DAA520)](https://github.com/go-mirofish/go-mirofish/stargazers)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg?style=flat-square)](./LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg?style=flat-square)](https://go.dev/)
[![Python Version](https://img.shields.io/badge/Python-3.11+-blue.svg?style=flat-square)](https://www.python.org/)
[![GitHub Sponsors](https://img.shields.io/github/sponsors/justinedevs?style=flat-square&logo=github&label=Sponsor)](https://github.com/sponsors/go-mirofish)
[![Buy Me A Coffee](https://img.shields.io/badge/Buy%20Me%20A%20Coffee-ffdd00?style=flat-square&logo=buy-me-a-coffee&logoColor=black)](https://www.buymeacoffee.com/justinedevs)
[![Discord](https://img.shields.io/badge/Discord-Join-5865F2?style=flat-square&logo=discord&logoColor=white)](http://discord.gg/ePf5aPaHnA)
[![X](https://img.shields.io/badge/X-Follow-000000?style=flat-square&logo=x&logoColor=white)](https://x.com/mirofish_ai)

[English](./README.md) | [中文文档](./README-ZH.md)

</div>

Upload documents, describe what you want to predict, and get a full simulation report—**on your laptop**.

> [!NOTE]
> **go-mirofish** is a lightweight fork of [MiroFish](https://github.com/666ghj/MiroFish). The AI features and five-step workflow are the same; **only the web and runtime layer** is optimized for local-first, lower-overhead deployment.

## What changed vs MiroFish

| | MiroFish | go-mirofish |
| --- | --- | --- |
| RAM usage | ~500MB–1GB typical | ~350–600MB target |
| Startup | ~5–10s typical | ~1–2s target (hybrid stack) |
| Hardware | 8GB+ RAM comfortable | 4GB+ RAM target |
| Setup | Multi-step dev stack | **One command** (`./start.sh`) on the roadmap; Docker or npm today |

> [!NOTE]
> Targets above depend on workload, model choice, and simulation size. See [Installation](docs/getting-started/installation.md) for what works **today** in this repository.

## Quick start

**Goal:** running UI + API on a fresh machine in a few minutes.

1. **Clone**

   ```bash
   git clone https://github.com/go-mirofish/go-mirofish.git
   cd go-mirofish
   ```

2. **Configure**

   ```bash
   cp .env.example .env
   ```

   Edit `.env` and set at least **`LLM_API_KEY`** and **`ZEP_API_KEY`**.

3. **Run** (pick one)

   ```bash
   docker compose up -d
   ```

   **Or** from source (Node18+, Python 3.11+, [uv](https://docs.astral.sh/uv/) recommended):

   ```bash
   npm run setup:all && npm run dev
   ```

   - App UI: [http://localhost:3000](http://localhost:3000)
   - API: [http://localhost:5001](http://localhost:5001)

> [!IMPORTANT]
> You need **`LLM_API_KEY`** and **`ZEP_API_KEY`** for the default cloud path. For **local LLMs** (no cloud key for the model), see [Ollama setup](docs/configuration/ollama.md).

> [!NOTE]
> A single **`./start.sh`** entrypoint (prebuilt Go gateway + Python, no Node for daily use) is **planned** for this fork. Until it lands in-repo, use **Docker** or **`npm run dev`** as shown—details in [Installation](docs/getting-started/installation.md).

## How it works (5 steps)

1. **Graph building** — upload seed documents; build the knowledge graph  
2. **Environment setup** — extract entities, personas, and agent configuration  
3. **Simulation** — run the multi-agent social simulation  
4. **Report generation** — produce an analysis report from the simulated world  
5. **Deep interaction** — chat with agents and the report assistant  

## Screenshots

<div align="center">
<table>
<tr>
<td><img src="./static/image/Screenshot/运行截图1.png" alt="Screenshot 1" width="100%"/></td>
<td><img src="./static/image/Screenshot/运行截图2.png" alt="Screenshot 2" width="100%"/></td>
</tr>
<tr>
<td><img src="./static/image/Screenshot/运行截图3.png" alt="Screenshot 3" width="100%"/></td>
<td><img src="./static/image/Screenshot/运行截图4.png" alt="Screenshot 4" width="100%"/></td>
</tr>
<tr>
<td><img src="./static/image/Screenshot/运行截图5.png" alt="Screenshot 5" width="100%"/></td>
<td><img src="./static/image/Screenshot/运行截图6.png" alt="Screenshot 6" width="100%"/></td>
</tr>
</table>
</div>

## Hardware compatibility

| Device | RAM | Works? |
| --- | ---: | --- |
| Desktop / laptop | 8GB | Yes |
| Desktop / laptop | 4GB | Yes (smaller simulations) |
| Raspberry Pi 5 | 4GB | Yes (light workloads; validate locally) |
| Raspberry Pi 4 | 4GB | Limited (expect tight headroom) |

> [!WARNING]
> Large graphs, long simulations, or heavy models can exceed **4GB** systems. Start with short runs and smaller seeds.

## Contributing

Issues and PRs are welcome. Use this repo for **go-mirofish** changes; upstream product discussion stays with [MiroFish](https://github.com/666ghj/MiroFish). For **contributing** (6-layer PR planning, Husky, Commitlint, Changesets, Renovate), see **[docs/contributing/README.md](docs/contributing/README.md)** (includes the [6-layer PR planning guide](docs/contributing/github-pr-6-layer.md)). Longer guides and the Phase 1–6 roadmap will also live on **[go.mirofish.ai](https://go.mirofish.ai)** as the docs site grows.

## License

[AGPL-3.0](./LICENSE).

## Acknowledgments

Derived from **[MiroFish](https://github.com/666ghj/MiroFish)**. Simulation is powered by **[OASIS](https://github.com/camel-ai/oasis)**—thanks to the CAMEL-AI team.
