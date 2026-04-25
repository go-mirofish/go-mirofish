SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
# Default `make` = Docker gateway (API) on :3000. UI: `npm run dev` (Vite :5173). No Python on host.
.DEFAULT_GOAL := up

ROOT := $(shell cd "$(dir $(lastword $(MAKEFILE_LIST)))" && pwd)

# Rebuild + start: use `make up` (already runs docker compose up -d --build). Do not use `make up --build` — make will error.
.PHONY: up up-release build down logs bootstrap dev gateway gateway-build frontend frontend-build \
        test test-all vet \
        benchmark benchmark-live benchmark-load benchmark-stress benchmark-soak benchmark-smoke \
        benchmark-run benchmark-stress-only benchmark-stress-heavy \
        server-up server-bench server-bench-wait \
        api-wiring-report \
        release release-notes release-changelog \
        clean install

# ─── DOCKER STACK ─────────────────────────────────────────────────────────────

up:
	@cd "$(ROOT)" && docker compose up -d --build

# Build images only (no containers started). To rebuild and run the stack: `make up`.
build:
	@cd "$(ROOT)" && docker compose build

# Optional all-in-one image (static UI in container). Default dev: `make up` + `npm run dev`.
up-release:
	@cd "$(ROOT)" && docker compose -f docker-compose.release.yml up -d --build

down:
	@cd "$(ROOT)" && docker compose down

logs:
	@cd "$(ROOT)" && docker compose logs -f

# ─── LOCAL BOOTSTRAP ──────────────────────────────────────────────────────────

bootstrap:
	@bash "$(ROOT)/scripts/dev/bootstrap.sh"

install: bootstrap

# ─── LOCAL DEV (frontend only; run `make up` first for the gateway in Docker) ─

dev:
	@bash "$(ROOT)/scripts/dev/frontend.sh"

# Non-default: run the Go gateway on the host (debugging). Canonical API path is `make up` (Docker).
gateway:
	@bash "$(ROOT)/scripts/dev/gateway.sh"

gateway-build:
	@bash "$(ROOT)/scripts/dev/gateway.sh" build

frontend:
	@bash "$(ROOT)/scripts/dev/frontend.sh"

frontend-build:
	@bash "$(ROOT)/scripts/dev/frontend.sh" build

# ─── TESTS ────────────────────────────────────────────────────────────────────

test:
	@bash "$(ROOT)/scripts/dev/test.sh"

test-all:
	@cd "$(ROOT)/gateway" && go vet ./...
	@bash "$(ROOT)/scripts/dev/test.sh"

vet:
	@cd "$(ROOT)/gateway" && go vet ./...

# ─── BENCHMARKS ───────────────────────────────────────────────────────────────

# Run the Go benchmark tool against a live gateway.
# Requires the gateway to be running (`make up`; or `make gateway` only for non-Docker gateway debugging).
benchmark-run:
	@mkdir -p "$(ROOT)/benchmark/results/benchmarks"
	@BENCH_BASE_URL="$${BENCH_BASE_URL:-http://127.0.0.1:3000}"; \
	 cd "$(ROOT)/gateway" && go run ./cmd/benchmark \
	   --base-url "$$BENCH_BASE_URL" \
	   --out "$(ROOT)/benchmark/results/benchmarks/benchmark.json" \
	   --release "$${RELEASE:-dev}"

# Stress profile only (faster; good for a focused stress test).
benchmark-stress-only:
	@mkdir -p "$(ROOT)/benchmark/results/benchmarks"
	@BENCH_BASE_URL="$${BENCH_BASE_URL:-http://127.0.0.1:3000}"; \
	 cd "$(ROOT)/gateway" && go run ./cmd/benchmark \
	   --base-url "$$BENCH_BASE_URL" \
	   --stress-only \
	   --out "$(ROOT)/benchmark/results/benchmarks/benchmark-stress-only.json" \
	   --release "$${RELEASE:-dev}"

# Heavier stress: doubles stress RPS and concurrency.
benchmark-stress-heavy:
	@mkdir -p "$(ROOT)/benchmark/results/benchmarks"
	@BENCH_BASE_URL="$${BENCH_BASE_URL:-http://127.0.0.1:3000}"; \
	 cd "$(ROOT)/gateway" && go run ./cmd/benchmark \
	   --base-url "$$BENCH_BASE_URL" \
	   --heavy \
	   --out "$(ROOT)/benchmark/results/benchmarks/benchmark-heavy.json" \
	   --release "$${RELEASE:-dev}"

# Docker stack: create .env if missing, ensure data dirs, start gateway.
server-up:
	@test -f "$(ROOT)/.env" || (echo "Copying .env.example -> .env (edit API keys as needed)" && cp "$(ROOT)/.env.example" "$(ROOT)/.env")
	@mkdir -p "$(ROOT)/data/projects" "$(ROOT)/data/reports" "$(ROOT)/data/tasks" "$(ROOT)/data/simulations"
	@cd "$(ROOT)" && docker compose up -d --build

# Wait for http://127.0.0.1:3000/health (up to 90s) — use after server-up.
server-bench-wait:
	@bash -c 'for i in $$(seq 1 90); do curl -sf http://127.0.0.1:3000/health >/dev/null && echo "[server-bench] gateway ready" && exit 0; sleep 1; done; echo "[server-bench] timeout waiting for gateway" >&2; exit 1'

# Full: bring Docker up, wait for health, run load+stress+soak → benchmark/results/benchmarks/benchmark.json
server-bench: server-up server-bench-wait
	@mkdir -p "$(ROOT)/benchmark/results/benchmarks"
	@cd "$(ROOT)/gateway" && go run ./cmd/benchmark \
	  --base-url "http://127.0.0.1:3000" \
	  --out "$(ROOT)/benchmark/results/benchmarks/benchmark.json" \
	  --release "$${RELEASE:-local}"

# Boot the standardized stack locally, run all benchmark profiles, write artifacts.
benchmark-live:
	@cd "$(ROOT)/gateway" && go run ./cmd/mirofish-hybrid live-benchmark

benchmark-load:
	@bash "$(ROOT)/scripts/dev/benchmark.sh" benchmark --profile load

benchmark-stress:
	@bash "$(ROOT)/scripts/dev/benchmark.sh" benchmark --profile stress

benchmark-soak:
	@bash "$(ROOT)/scripts/dev/benchmark.sh" benchmark --profile soak

benchmark-smoke:
	@bash "$(ROOT)/scripts/dev/benchmark.sh" smoke

# Default benchmark target: runs the Go tool against a live gateway.
benchmark: benchmark-run

# API wiring report: enumerate every route, verify ownership and HTTP contract.
# Requires the gateway to be running.
api-wiring-report:
	@mkdir -p "$(ROOT)/benchmark/results/api-wiring"
	@BENCH_BASE_URL="$${BENCH_BASE_URL:-http://127.0.0.1:3000}"; \
	 cd "$(ROOT)/gateway" && go run ./cmd/api-wiring-report \
	   --base-url "$$BENCH_BASE_URL" \
	   --out "$(ROOT)/benchmark/results/api-wiring/api-wiring-report.json"

# ─── RELEASE ──────────────────────────────────────────────────────────────────

release:
	@bash "$(ROOT)/scripts/release/release.sh"

release-notes:
	@bash "$(ROOT)/scripts/release/release.sh" notes

release-changelog:
	@bash "$(ROOT)/scripts/release/release.sh" changelog

# ─── CLEAN ────────────────────────────────────────────────────────────────────

clean:
	@bash "$(ROOT)/scripts/dev/clean.sh"
