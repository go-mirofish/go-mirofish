<div align="center">

<img src="./static/image/go-mirofish-thumbnail (transparent).png" alt="go-mirofish logo" width="55%"/>

**go-mirofish, lightweight and local-first**

[![GitHub Stars](https://img.shields.io/github/stars/go-mirofish/go-mirofish?style=flat-square&color=DAA520)](https://github.com/go-mirofish/go-mirofish/stargazers)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg?style=flat-square)](./LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg?style=flat-square)](https://go.dev/)
[![Python](https://img.shields.io/badge/Python-3.11.x-blue.svg?style=flat-square)](https://www.python.org/)
[![GitHub Sponsors](https://img.shields.io/github/sponsors/justinedevs?style=flat-square&logo=github&label=Sponsor)](https://github.com/sponsors/go-mirofish)
[![Buy Me A Coffee](https://img.shields.io/badge/Buy%20Me%20A%20Coffee-ffdd00?style=flat-square&logo=buy-me-a-coffee&logoColor=black)](https://www.buymeacoffee.com/justinedevs)
[![X](https://img.shields.io/badge/X-Follow-000000?style=flat-square&logo=x&logoColor=white)](https://x.com/trader2g)

</div>

Upload documents, describe what you want to predict, and get a full simulation report **on your laptop**.

**Public preview (Vercel):** [gomirofish.vercel.app](https://gomirofish.vercel.app) — custom domain **go.mirofish.ai** is pending (subdomain access with the domain holder is still in progress).

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

   **Or** from source (Node 18+ and **Python 3.11.x**; [uv](https://docs.astral.sh/uv/) is optional; see [Installation](docs/getting-started/installation.md#python-uv-and-venv)):

   ```bash
   npm run setup:all && npm run dev
   ```

   - App UI: [http://localhost:3000](http://localhost:3000)
   - API: [http://localhost:5001](http://localhost:5001)

> [!IMPORTANT]
> You need **`LLM_API_KEY`** and **`ZEP_API_KEY`** for the default cloud path. For **local LLMs** or other OpenAI-compatible providers, see [Ollama setup](docs/configuration/ollama.md) and [OpenAI-compatible providers](docs/configuration/providers.md).

> [!NOTE]
> This fork now includes additive hybrid entrypoints:
> - **`compose.yaml`** for the gateway + backend container topology
> - **`./start.sh`** / **`start.bat`** for the local prebuilt-gateway path
>
> Build the gateway binary into **`gateway/bin/`** first, then use the hybrid path described in [Installation](docs/getting-started/installation.md).

## How it works (5 steps)

1. **Graph building:** upload seed documents; build the knowledge graph  
2. **Environment setup:** extract entities, personas, and agent configuration  
3. **Simulation:** run the multi-agent social simulation  
4. **Report generation:** produce an analysis report from the simulated world  
5. **Deep interaction:** chat with agents and the report assistant  

## Showcase Proof

The proof surface is now split in two:

- live-stack benchmark proof for the real Go gateway + backend runtime
- example-suite benchmark proof for the five plug-and-play local templates

Current captured result:

| Proof surface | Status | Evidence |
| --- | --- | --- |
| Backend boot | `PASS` | backend health responded as `go-mirofish-backend` |
| Go gateway boot | `PASS` | gateway health responded as `go-mirofish-gateway` |
| Bounded stress pass | `PASS` | `80/80` requests succeeded |
| Latency envelope | `PASS` | p50 `4.55ms`, p95 `24.59ms`, max `29.0ms` |
| Full benchmark flow | `PASS` | project, graph, simulation, and report IDs were all captured |
| Report artifact quality | `PASS` | benchmark completed with `report_non_empty=true` in the latest captured run |

Current benchmark phase evidence:

| Phase | Status | Evidence |
| --- | --- | --- |
| Ontology generation | `PASS` | `project_id=proj_57e3f0da8a33` |
| Graph build | `PASS` | `graph_id=go_mirofish_20fe3c512dea4b74` |
| Simulation create | `PASS` | `simulation_id=sim_553008e20bc1` |
| Simulation run | `PASS` | `simulation_status=completed` |
| Report generation | `PASS` | `report_id=report_6238622d5fd4`, `report_status=completed` |
| Report content | `PASS` | `report_non_empty=true` |

What that means:

- the hybrid go-mirofish stack is benchmark-verified end to end
- the Go gateway is part of the measured request path, so this is not just inherited MiroFish behavior
- the latest captured run now completes with a non-empty report artifact

Current example-suite benchmark results:

| Example | Profile | Status | Startup | Runtime | Artifact |
| --- | --- | --- | ---: | ---: | --- |
| Product Launch PR War Room | `medium` | `PASS` | `10.93ms` | `19.81ms` | `risk_report.json` |
| Hyper-Local Urban Planning | `medium` | `PASS` | `17.58ms` | `34.34ms` | `coalition_highway.json`, `coalition_park.json` |
| Zero-Day Cyber Incident Drill | `medium` | `PASS` | `10.70ms` | `20.50ms` | `incident_report.json` |
| De-Fi Sentiment Stress-Test | `medium` | `PASS` | `6.92ms` | `15.97ms` | `liquidation_cascade_forecast.json` |
| Lost Ending Literary Simulator | `medium` | `PASS` | `15.04ms` | `26.59ms` | `draft_ending.json`, `draft_ending.txt` |

What the example suite proves:

- `product-launch-war-room` proves concurrent multi-agent crisis analysis with a concrete risk report
- `hyperlocal-urban-planning` proves edge-friendly local coalition modeling with two scenario variants
- `zero-day-incident-drill` proves privacy-preserving internal/external rumor modeling without a cloud dependency
- `defi-sentiment-stress-test` proves offline-first sentiment cascade modeling with explicit panic threshold output
- `lost-ending-literary-simulator` proves structured creative simulation with consistency scoring, not just enterprise-only flows

Read the evidence in detail:

- [Benchmark report](./docs/hybrid/benchmark-report.md)
- [Go parity matrix](./docs/hybrid/go-parity-matrix.md)
- [Go migration plan](./docs/hybrid/go-migration-plan.md)
- [Showcase policy](./docs/hybrid/showcase.md)
- `benchmark/results/examples-benchmark-suite.json` (local capture; path may be gitignored)
- `benchmark/results/smoke/latest.json`
- `docs/bundled-benchmarks/*.json` — short names for the in-app benchmark report (committed)

## Examples & Benchmarks

List examples:

```bash
go run ./gateway/cmd/go-mirofish-examples --list
```

Run one example:

```bash
go run ./gateway/cmd/go-mirofish-examples --example product-launch-war-room --profile medium
```

Run smoke validation for all examples:

```bash
go run ./gateway/cmd/go-mirofish-examples --all --smoke-only --profile small
```

Run the benchmark suite:

```bash
go run ./gateway/cmd/go-mirofish-examples --all --bench-only --profile medium
```

Compare two benchmark runs:

```bash
go run ./gateway/cmd/go-mirofish-examples --compare docs/bundled-benchmarks/product-launch__small__latest.json,docs/bundled-benchmarks/literary-sim__small__latest.json
```

## 🌐 Live Demo

- Static playground (zero-cost replay): [gomirofish.vercel.app](https://gomirofish.vercel.app)

## 📸 Screenshots

<div align="center">
  <table>
    <tr>
      <td align="center">
        <img src="static/image/Screenshot/Screenshot(1).png" width="520" />
        <br />
        <sub><b>Home / entry</b></sub>
      </td>
      <td align="center">
        <img src="static/image/Screenshot/Screenshot(2).png" width="520" />
        <br />
        <sub><b>Simulation run</b></sub>
      </td>
    </tr>
    <tr>
      <td align="center">
        <img src="static/image/Screenshot/Screenshot(3).png" width="520" />
        <br />
        <sub><b>Report generation</b></sub>
      </td>
      <td align="center">
        <img src="static/image/Screenshot/Screenshot(4).png" width="520" />
        <br />
        <sub><b>Report timeline / tools</b></sub>
      </td>
    </tr>
    <tr>
      <td align="center">
        <img src="static/image/Screenshot/Screenshot(5).png" width="520" />
        <br />
        <sub><b>Simulation history</b></sub>
      </td>
      <td align="center">
        <img src="static/image/Screenshot/Screenshot(6).png" width="520" />
        <br />
        <sub><b>Deep interaction</b></sub>
      </td>
    </tr>
  </table>
</div>

## Production Split

- Docs / landing / showcase: static
- Interactive playground: fixture-driven and precomputed, with no shared live inference
- Real product: local / self-hosted
- Optional advanced mode: BYOK

The homepage now follows that split:

- public visitors get a zero-cost static playground replay
- real runs happen only after connecting a local backend
- advanced users can point the backend at their own provider keys or local OpenAI-compatible models

## Go Migration

Current state:

- Go owns the public control plane, public routes, runner CLI, example suite, benchmark suite, provider layer, memory layer, and route orchestration
- Python no longer owns the public API surface
- Python remains as the simulation worker/runtime boundary and private worker internals

## Hardware compatibility

| Device | RAM | Works? |
| --- | ---: | --- |
| Desktop / laptop | 8GB | Yes |
| Desktop / laptop | 4GB | Yes (smaller simulations) |
| Raspberry Pi 5 | 4GB | ARM64-ready; pending on-device validation |
| Raspberry Pi 4 | 4GB | ARM64-ready; likely tight headroom, pending on-device validation |

> [!WARNING]
> Large graphs, long simulations, or heavy models can exceed **4GB** systems. Start with short runs and smaller seeds.

> [!NOTE]
> Hardware and runtime claims should be read together with the inline benchmark summary above, the detailed report in [docs/hybrid/benchmark-report.md](./docs/hybrid/benchmark-report.md), and the validation policy in [docs/hybrid/raspberry-pi-validation.md](./docs/hybrid/raspberry-pi-validation.md).

## Contributing

Issues and PRs are welcome. Use this repo for **go-mirofish** changes; upstream product discussion stays with [MiroFish](https://github.com/666ghj/MiroFish). Start with **[CONTRIBUTING.md](CONTRIBUTING.md)** and **[docs/contributing/README.md](docs/contributing/README.md)** (6-layer PR planning, Husky, Commitlint, Changesets, Renovate). Longer guides and the Phase 1–6 roadmap will also live on **[gomirofish.vercel.app](https://gomirofish.vercel.app)** (and on **go.mirofish.ai** once the custom domain is connected).

## License

[AGPL-3.0](./LICENSE).

## Acknowledgments

Derived from **[MiroFish](https://github.com/666ghj/MiroFish)**. Simulation is powered by **[OASIS](https://github.com/camel-ai/oasis)**. Thanks to the CAMEL-AI team.
