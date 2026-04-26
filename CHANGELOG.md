# Changelog

## v0.1.5 - 2026-04-26

Changes since `v0.1.0`.

### Added

- add README.md
- add public
- add seo
- add sitemap.xml
- add robots.txt
- add og-image.png
- add .env.example
- add vercel.json
- add pnpm-workspace.yaml

### Changed

- adjust CHANGELOG.md
- adjust skills-lock.json
- adjust pnpm-lock.yaml
- adjust main.js
- adjust App.vue
- adjust package.json
- adjust index.html
- adjust .gitignore
- adjust README.md

### Chore

- update BenchCharts.vue (select } from 'd3-selection'-] [-import { sca…
- update pnpm-lock.yaml (which@2.0.2: → vite@7.3.2:)


All notable changes to `go-mirofish` will be documented in this file.

## v0.1.4 - 2026-04-26

Changes since `v0.1.0`.

### Added

- add public
- add seo
- add sitemap.xml
- add robots.txt
- add og-image.png
- add .env.example
- add vercel.json
- add pnpm-workspace.yaml

### Changed

- adjust skills-lock.json
- adjust pnpm-lock.yaml
- adjust main.js
- adjust App.vue
- adjust package.json
- adjust index.html
- adjust .gitignore
- adjust README.md

### Chore

- update BenchCharts.vue (select } from 'd3-selection'-] [-import { sca…
- update pnpm-lock.yaml (which@2.0.2: → vite@7.3.2:)

## v0.1.3 - 2026-04-25

Changes since `v0.1.0`.

### Added

- add up.sh
- add up.bat
- add Screenshot(8).png
- add Screenshot(7).png
- add release.sh
- add docker
- add test.sh
- add gateway.sh
- add frontend.sh
- add common.sh
- add clean.sh
- add bootstrap.sh
- add benchmark.sh
- add scheduler.go
- add native_test.go
- add native_helpers.go
- add native.go
- add telemetry
- add store_test.go
- add service_test.go
- add runinstructions
- add artifactcontract
- add mirofish-hybrid
- add benchmark-report
- add api-wiring-report
- add benchmark.exe
- add benchmark-report.exe
- add useClientHardwarePreview.js
- add DocsRoadmap.vue
- add SystemTerminalSplit.vue
- add ReportPreparingPanel.vue
- add Screenshot(8).png
- add Screenshot(7).png
- add showcase.md
- add roadmap.md
- add report
- add docker-compose.release.yml
- add ScriptsActivate.ps1
- add Makefile
- add MIGRATION.md
- add live-stack__hybrid__latest.json
- update ontology
- add skills-lock.json
- add runtime_sprint.py
- add merge_hybrid_stack_into_benchmarks.py
- add frontend_sprint.cjs
- add run-gateway.cjs
- add internal
- add go.sum
- add go-mirofish-examples
- add backend
- add docs
- add docs
- add static
- add examples
- add bundled-benchmarks
- add Screenshot(6).png
- add Screenshot(5).png
- add Screenshot(4).png
- add Screenshot(3).png
- add Screenshot(2).png
- add Screenshot(1).png
- add extract-release-notes.cjs
- add run_stress_probe.py
- add run_live_benchmark.sh
- add generate_benchmark_report.py
- add commit
- add runtimeTarget.js
- add playgroundMode.js
- add PlaygroundModePanel.vue
- add playground
- add hybrid
- add release-notes.yml

### Changed

- adjust build-gateway.cjs
- adjust renovate.json
- adjust errors.go
- adjust contract.go
- adjust store_test.go
- adjust store_test.go
- adjust service.go
- adjust go.sum
- adjust go.mod
- adjust SimulationView.vue
- adjust SimulationRunView.vue
- adjust ReportView.vue
- adjust MainView.vue
- adjust InteractionView.vue
- adjust benchmarkRunLabel.js
- adjust package.json
- adjust package-lock.json
- adjust pre-push
- adjust pre-commit
- adjust .gitignore
- adjust PULL_REQUEST_TEMPLATE.md
- adjust .env.example
- adjust App.vue
- adjust package.json
- adjust handler_test.go
- adjust README.md
- adjust start.sh
- adjust mirofish-gateway
- adjust go.mod
- adjust index.js
- adjust package.json
- adjust package-lock.json
- adjust __init__.py
- adjust .gitignore
- adjust mirofish-gateway
- adjust main_test.go
- adjust main.go
- adjust README.md
- adjust doc-layout.css
- adjust SiteFooter.vue
- adjust simulation.js
- adjust README.md
- adjust llm_client.py
- adjust simulation_manager.py
- adjust simulation.py
- adjust graph.py

### Fixed

- remove not existing redirect files

### Chore

- remove start.sh
- remove start.bat
- remove verify_contract_matrix.py
- remove runtime_sprint.py
- remove run_stress_probe.py
- remove run_live_benchmark.sh
- remove run_benchmark_smoke.py
- remove merge_hybrid_stack_into_benchmarks.py
- remove generate_benchmark_report.py
- remove frontend_sprint.cjs
- remove setup-backend.cjs
- update run-gateway.cjs (gatewayDir, → root,)
- remove run-backend.cjs
- update package.json ("npm install && npm install --prefix frontend",…
- update en.json (:3000 → :5173)
- update client_test.go ("agent unavailable", → &errText,)
- update client.go ("encoding/json"-] "fmt" "os" "path/filepath" " → =…
- update bridge_test.go ("context"-] "encoding/json" "errors" "os" [- "…
- update bridge.go ((-] [- "context"-] "encoding/json"[-"fmt"-] [- "os…
- update store.go ("os/exec"-] "path/filepath" "sort" [- "strconv"- → "…
- update store.go (s.SaveMeta(reportID, → s.saveMetaLocked(reportID,)
- update store.go (raw, → if)
- update store_test.go (project → task)
- update store.go (return os.WriteFile(s.taskPath(task.TaskID), → tmpPa…
- update service.go (intValue(item["round"]) → actionRound(item))
- update ingestion.go (len(episodes)) → len(chunk)))
- update handlers.go (simulation.NormalizeRunStatus(simulationID, runSt…
- update handler_test.go ("state.json")); → "control_state.json"));)
- update handler_test.go (Windows → t.TempDir cleanup)
- update handler.go (name → path)
- update handler_test.go ("io" → "context")
- update handler.go (BackendURL string-] [- BackendProxy *htt → New(fro…
- update service_test.go (missingProjectsDir → blocker)
- update main_test.go ("bytes"-] "context" "encoding/csv" "encoding/js…
- update main.go (backendURL *url.URL-] frontendDevURL *url.U → apphttp…
- update vite.config.js ('../docs') -] [- } → '../docs'),)
- update DocsView.vue ('DocsShowcase', → 'DocsShowcase' || entry?.compo…
- update doc-layout.css (strokes — → strokes;)
- update index.js ('contributing'] → 'contributing', 'roadmap'])
- update manifest.js ('docs/hybrid/benchmark-report.md', → 'docs/report…
- update BenchmarkRunCombobox.vue ('—' → '-')
- update DocsShowcase.vue (— → -)
- update DocsBenchmarks.vue (ref → ref, watch)
- update benchmarkRunLabel.js ((`scenario__profile__variant.json` — `__…
- update Step5Interaction.vue (<!-- Loading State -->-] <div v-els → cl…
- update Step4Report.vue (class="loading-state"> → class="section-loadi…
- update Step3Simulation.vue (<!-- Bottom Info / Logs -->-] [- <div cla…
- update Step2EnvSetup.vue (<!-- Bottom Info / Logs -->-] [- <div class…
- update Step1GraphBuild.vue (accent">{{ $t('step1.inProgress') → succe…
- update PlaygroundModePanel.vue ('—' → '-')
- update GraphPanel.vue (class="graph-state"> → class="graph-state" :cl…
- update simulation.js (* @param {Object} data - { project_id, graph_id…
- update report.js (* @param {Object} data - { simulation_id, force_re…
- update index.js (// 5-minute timeout; ontology generation can take →…
- update graph.js (@returns {Promise} → Upload seed documents and gener…
- remove raspberry-pi-validation.md
- remove go-parity-matrix.md
- remove go-migration-plan.md
- remove contract-matrix.md
- remove benchmark-report.md
- update installation.md (For the public app ([go-mirofish.vercel.app](h…
- update commit-messages.md ((Conventional Commits) -] [- -] [-Commits…
- update README.md (Hub for contributor docs. The repo root **[CONTRIB…
- update README.md (UI) → UI + fixtures))
- update gateway.Dockerfile (GOARCH=amd64-] go build -o /out/go-mirofis…
- remove backend.Dockerfile
- update docker-compose.yml (backend:-] [- build:-] [- context: .-] [-…
- remove uv.lock
- remove test_profile_format.py
- remove run_twitter_simulation.py
- remove run_reddit_simulation.py
- remove run_parallel_simulation.py
- remove run.py
- remove requirements.txt
- remove pyproject.toml
- remove zep_paging.py
- remove retry.py
- remove locale.py
- remove llm_provider.py
- remove llm_client.py
- remove file_parser.py
- remove __init__.py
- remove zep_tools.py
- remove zep_graph_memory_updater.py
- remove zep_entity_reader.py
- remove text_processor.py
- remove simulation_runner.py
- remove simulation_manager.py
- remove simulation_ipc.py
- remove simulation_config_generator.py
- remove report_agent.py
- remove oasis_profile_generator.py
- remove graph_builder.py
- remove __init__.py
- remove task.py
- remove project.py
- remove __init__.py
- remove llm_provider.py
- remove config.py
- remove __init__.py
- update README.md ([![Python](https://img.shields.io/badge/Python-3.1…
- update CONTRIBUTING.md (**go-mirofish**. This repository focuses on →…
- sync pnpm lockfile
- update package-lock.json ("^9.0.0" → "^9.0.0",)
- update bridge_test.go ("/bin/sh") → posixShellForTest(t)))
- update store.go (exec.Command("python3", → exec.Command(pythonForExec…
- update handler_test.go (exec.Command("python3", → exec.Command(python…
- update handler_test.go (wantFragment: "no such file or directory", →…
- update main_test.go ("/bin/sh", → workerTestShell(t),)
- update DocsBenchmarks.vue (})) → }))
- update benchmarkRunLabel.js (rest.join('--') → rest.join('__'))
- update run_benchmark_smoke.py (report_generate["data"]["report_id"] →…
- update generate_benchmark_report.py (Backend health service: `{data.g…
- update package.json ("https://go.mirofish.ai", → "https://go-mirofish.…
- update package-lock.json ("^9.0.0" → "^9.0.0",)
- update en.json ("HTTP", → "HTTP :3000 (Vite dev UI)",)
- update main_test.go ("net/url"-] "strings" "testing" {+"time"+} ) t →…
- update main.go ("bytes"-] [- "encoding/json"-] "errors" [- "io"-] → a…
- update vite.config.js ('http://localhost:5001', → process.env.VITE_GA…
- update DocsView.vue (class="home-container doc-site docs-view">-] [-…
- update doc-layout.css (12px; → 0;)
- update runtimeTarget.js ('http://localhost:5001' → '')
- update Step5Interaction.vue (var(--doc-muted); → var(--doc-text);)
- update DocsArchitectureDiagram.vue (560 256" → 520 300")
- update index.html (href="https://go.mirofish.ai/" → href="https://gom…
- remove showcase.md
- update go-parity-matrix.md (Python Flask → Go)
- update benchmark-report.md (v0.1.0-] benchmark report {+## Executive…
- update installation.md ([go.mirofish.ai](https://go.mirofish.ai), → t…
- update ollama.md (go.mirofish.ai → [go-mirofish.vercel.app](https://go…
- update run_twitter_simulation.py (print(f"Error: → raise FileNotFound…
- update run_reddit_simulation.py (print(f"Error: → raise FileNotFoundE…
- update run_parallel_simulation.py (self.logger.warning(f"Failed → log…
- update llm_client.py (import random-] from typing import Optional, Di…
- update report_agent.py (*]\s+', content, re.MULTILINE))+} {+ quote_ →…
- update task.py (Optional → Optional, List)
- update simulation.py
- update report.py
- update graph.py
- update __init__.py
- update README.md (src="./static/image/go-mirofish-thumbnail.png" → sr…
- remove .gitkeep
- update Step5Interaction.vue (450px; → 0;)
- update Step4Report.vue (0 → 12px)
- update run_benchmark_smoke.py (request_json(url: → request_json()
- update package.json ("changeset" → "changeset",)
- update en.json ("JustineDevs" → "JustineDevs",)
- update SimulationView.vue (class="main-view"> → class="doc-workbench">)
- update SimulationRunView.vue (class="main-view"> → class="doc-workben…
- update ReportView.vue (class="main-view"> → class="doc-workbench">)
- update Process.vue (stroke="#000" → stroke="currentColor")
- update MainView.vue (class="main-view doc-workbench"> → class="doc-wo…
- update InteractionView.vue (class="main-view"> → class="doc-workbench">)
- update Home.vue (class="doc-aside-kicker">{{ $t('home.docSectionPla →…
- update Step5Interaction.vue (stroke="#E5E7EB"></circle> → stroke="var…
- update Step4Report.vue (stroke="#E5E7EB"></circle> → stroke="var(--do…
- update Step3Simulation.vue ((res.success) → (res.success && res.data))
- update Step1GraphBuild.vue (#fff; → var(--doc-tooltip-pill-fg, #fafaf…
- update HistoryDatabase.vue (v-if="projects.length > 0 || loading" cla…
- update GraphPanel.vue ('#C0C0C0') → 'var(--doc-graph-link-stroke, #b8…
- update index.js (import.meta.env.VITE_API_BASE_URL || 'http://local →…
- update App.vue (#4b5563; → var(--doc-console-scroll, #4b5563);)
- update providers.md (**Gemini and Anthropic:** → **Anthropic:**)
- update run_parallel_simulation.py (logger.info(f"Truncated → log_info…
- update simulation_runner.py ([RunnerStatus.RUNNING, RunnerStatus.PAUS…
- update README.md ([![Discord](https://img.shields.io/badge/Discord-J…
- update .gitignore (docs/hybrid/* → docs/hybrid/capture-checklist.md)
- update .env.example (Alibaba DashScope qwen-plus → Google Gemini Open…

This project uses a curated, release-oriented changelog:
- human-written release notes for important product changes
- conventional-commit history as supporting input
- version tags as the source of release boundaries

## v0.1.0 - 2026-04-24

Initial public `go-mirofish` release.

### Added

- Added a Go gateway layer with route aliases, static asset serving, health endpoint support, and gateway tests.
- Added hybrid runtime entrypoints with `start.sh`, `start.bat`, organized Dockerfiles, and a single canonical `docker-compose.yml`.
- Added benchmark and verification tooling, including a seed fixture, contract matrix, contract verifier, benchmark smoke runner, and release/showcase documentation.
- Added local OpenAI-compatible provider support for setups like `llama.cpp` and Ollama without requiring a real cloud API key for localhost deployments.
- Added contributor and release tooling including Husky, Commitlint, Changesets, Renovate, release workflows, and the release changelog generator.

### Changed

- Reworked the repo around the `go-mirofish` product identity across docs, package metadata, runtime names, gateway binaries, and health/service labels.
- Centralized the container/runtime layout so Docker orchestration and gateway packaging live in one organized surface.
- Improved the source-development workflow for Windows and Git Bash by replacing shell-fragile commands with cross-platform Node wrappers.
- Shifted the checked-in application surface to English-first source strings and collapsed the locale registry to English for the current public release.

### Fixed

- Fixed gateway compatibility for aliased API routes used by the frontend and hybrid runtime.
- Fixed local startup script resolution for Windows venv layouts and renamed gateway binaries.
- Fixed multiple backend/runtime text, parser, and script issues uncovered during the English-only cleanup and hybrid migration work.
- Hardened LLM client retry behavior for transient OpenAI-compatible provider failures.

### Documentation

- Rewrote installation, provider, Ollama, hybrid runtime, showcase, and Raspberry Pi validation docs for the `go-mirofish` release line.
- Removed misleading upstream screenshot/demo framing from the active release surface and replaced it with first-party proof expectations.

### Notes

- Raspberry Pi support is currently positioned as `ARM64-ready` and pending on-device validation.
- First-party screenshots and demo recordings are still expected to be captured from a browser-capable runtime session and added separately.
