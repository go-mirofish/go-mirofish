# go-mirofish Showcase

This page tracks **first-party** proof for `go-mirofish`.

It intentionally excludes inherited MiroFish screenshots and demo assets, because those do not prove the hybrid fork itself.

## What counts as proof here

- benchmark scripts and fixtures owned by this repo
- gateway tests owned by this repo
- hybrid startup path owned by this repo
- contract-matrix verification owned by this repo
- first-party screenshots or recordings captured from the hybrid stack

## First-party proof currently in repo

### 1. Benchmark fixture

- Fixture: [benchmark/seed.txt](https://github.com/go-mirofish/go-mirofish/blob/main/benchmark/seed.txt)
- Benchmark runner: [scripts/hybrid/run_benchmark_smoke.py](https://github.com/go-mirofish/go-mirofish/blob/main/scripts/hybrid/run_benchmark_smoke.py)
- Stress probe: [scripts/hybrid/run_stress_probe.py](https://github.com/go-mirofish/go-mirofish/blob/main/scripts/hybrid/run_stress_probe.py)
- Latest report: [docs/hybrid/benchmark-report.md](https://github.com/go-mirofish/go-mirofish/blob/main/docs/hybrid/benchmark-report.md)

The benchmark flow targets the hybrid API surface:

1. ontology generation
2. graph build
3. simulation prepare
4. simulation run
5. report generation

### 2. Contract verification

- Contract matrix: [docs/hybrid/contract-matrix.md](https://github.com/go-mirofish/go-mirofish/blob/main/docs/hybrid/contract-matrix.md)
- Verification script: [scripts/hybrid/verify_contract_matrix.py](https://github.com/go-mirofish/go-mirofish/blob/main/scripts/hybrid/verify_contract_matrix.py)

Verified checks:

- benchmark seed contract
- backend route inventory
- frontend caller inventory
- gateway alias policy

### 3. Gateway tests

- Gateway implementation: [gateway/cmd/mirofish-gateway/main.go](https://github.com/go-mirofish/go-mirofish/blob/main/gateway/cmd/mirofish-gateway/main.go)
- Gateway tests: [gateway/cmd/mirofish-gateway/main_test.go](https://github.com/go-mirofish/go-mirofish/blob/main/gateway/cmd/mirofish-gateway/main_test.go)

Verified:

```bash
cd gateway
GOCACHE=/tmp/go-build-cache go test ./...
GOCACHE=/tmp/go-build-cache go build ./...
```

### 4. Hybrid startup path

- Local entrypoint: [start.sh](https://github.com/go-mirofish/go-mirofish/blob/main/start.sh)
- Windows entrypoint: [start.bat](https://github.com/go-mirofish/go-mirofish/blob/main/start.bat)
- Compose topology: [docker-compose.yml](https://github.com/go-mirofish/go-mirofish/blob/main/docker-compose.yml)

Verified in this repo:

- backend can boot from `backend/.venv`
- gateway can boot from `gateway/bin/mirofish-gateway`
- gateway `/health` responds
- backend `/health` responds
- live route aliases respond through the gateway

### 5. Current benchmark status

The benchmark harness is real and has now been run against the live `go-mirofish` stack.

Observed current state:

- backend boot: pass
- gateway boot: pass
- health endpoint stress pass: pass
- full benchmark flow: blocked by a live upstream provider rate limit during ontology generation

See the first-party report:

- [docs/hybrid/benchmark-report.md](https://github.com/go-mirofish/go-mirofish/blob/main/docs/hybrid/benchmark-report.md)

### 6. Raspberry Pi support status

Pi support should be described through claim levels, not by assumption.

See:

- [docs/hybrid/raspberry-pi-validation.md](https://github.com/go-mirofish/go-mirofish/blob/main/docs/hybrid/raspberry-pi-validation.md)

## What is still missing

### First-party screenshots

No committed screenshot in this repo is yet a confirmed first-party capture from the `go-mirofish` hybrid stack.

Expected asset directory:

- [static/image/go-mirofish-showcase/](https://github.com/go-mirofish/go-mirofish/tree/main/static/image/go-mirofish-showcase)

Expected screenshot set:

- `home.png`
- `graph-build.png`
- `env-setup.png`
- `simulation-run.png`
- `report-generation.png`
- `deep-interaction.png`

### First-party video demo

No committed video or recording in this repo is yet a confirmed first-party demo of the `go-mirofish` hybrid stack.

Expected asset path:

- `static/image/go-mirofish-showcase/demo.mp4`

## Capture note

Standalone `go-mirofish` screenshots and demo recordings should be captured from the hybrid stack itself and committed only after they are first-party proof artifacts.

## Rule

Until first-party captures are added here, inherited MiroFish visuals should be treated as lineage context, not as `go-mirofish` proof.

Until a real Pi run is committed, Raspberry Pi language should stay at:

- ARM64-ready
- designed for Pi-class 64-bit Linux
- pending on-device validation
