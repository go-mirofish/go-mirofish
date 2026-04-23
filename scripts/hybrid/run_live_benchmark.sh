#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BENCH_DIR="$ROOT_DIR/benchmark/results"
BENCH_PYTHON="${BENCH_PYTHON:-/tmp/go-mirofish-bench-venv/bin/python}"
BENCH_VENV="${BENCH_VENV:-/tmp/go-mirofish-bench-venv}"
BACKEND_LOG="$BENCH_DIR/backend-live.log"
GATEWAY_LOG="$BENCH_DIR/gateway-live.log"
BENCH_JSON="$BENCH_DIR/v0.1.0-live-benchmark.json"
BENCH_REPORT="$ROOT_DIR/docs/hybrid/benchmark-report.md"
BACKEND_PID=""
GATEWAY_PID=""

cleanup() {
  for pid in "$GATEWAY_PID" "$BACKEND_PID"; do
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      kill "$pid" >/dev/null 2>&1 || true
      wait "$pid" 2>/dev/null || true
    fi
  done
}

wait_for_url() {
  local url="$1"
  local label="$2"
  "$BENCH_PYTHON" - <<PY
import sys, time, urllib.request
url = ${url@Q}
label = ${label@Q}
for _ in range(120):
    try:
        with urllib.request.urlopen(url, timeout=2):
            raise SystemExit(0)
    except Exception:
        time.sleep(1)
print(f"timeout waiting for {label}: {url}", file=sys.stderr)
raise SystemExit(1)
PY
}

ensure_bench_python() {
  export UV_CACHE_DIR="${UV_CACHE_DIR:-/tmp/uv-cache}"
  if [[ ! -x "$BENCH_PYTHON" ]]; then
    uv venv --python 3.11 "$BENCH_VENV"
  fi

  if ! "$BENCH_PYTHON" - <<'PY' >/dev/null 2>&1
mods = ['flask', 'dotenv', 'openai', 'zep_cloud', 'camel', 'oasis']
for mod in mods:
    __import__(mod)
PY
  then
    uv pip install --python "$BENCH_PYTHON" -r "$ROOT_DIR/backend/requirements.txt"
  fi
}

trap cleanup EXIT INT TERM

mkdir -p "$BENCH_DIR"
ensure_bench_python

export GOCACHE="${GOCACHE:-/tmp/go-build-cache}"
export ROOT_DIR
export BENCH_JSON
npm run build:gateway >/dev/null
npm run build --prefix frontend >/dev/null

export FLASK_DEBUG=false
export FLASK_HOST=127.0.0.1
export FLASK_PORT=5001
"$BENCH_PYTHON" "$ROOT_DIR/backend/run.py" >"$BACKEND_LOG" 2>&1 &
BACKEND_PID="$!"
export BACKEND_PID
wait_for_url "http://127.0.0.1:5001/health" "backend health"

export BACKEND_URL="http://127.0.0.1:5001"
export FRONTEND_DIST_DIR="$ROOT_DIR/frontend/dist"
export GATEWAY_BIND_HOST=127.0.0.1
export GATEWAY_PORT=3000
"$ROOT_DIR/gateway/bin/go-mirofish-gateway" >"$GATEWAY_LOG" 2>&1 &
GATEWAY_PID="$!"
export GATEWAY_PID
wait_for_url "http://127.0.0.1:3000/health" "gateway health"

"$BENCH_PYTHON" - <<'PY'
import concurrent.futures
import json
import os
import pathlib
import statistics
import subprocess
import time
import urllib.request

root = pathlib.Path(os.environ["ROOT_DIR"]) if "ROOT_DIR" in os.environ else pathlib.Path.cwd()
json_path = pathlib.Path(os.environ["BENCH_JSON"])
backend_pid = int(os.environ["BACKEND_PID"])
gateway_pid = int(os.environ["GATEWAY_PID"])

def health(url: str) -> dict:
    with urllib.request.urlopen(url, timeout=5) as resp:
        return json.loads(resp.read().decode("utf-8"))

def proc_status(pid: int) -> dict:
    result = {"pid": pid}
    with open(f"/proc/{pid}/status", "r", encoding="utf-8") as fh:
        for line in fh:
            if line.startswith(("VmRSS:", "VmHWM:", "Threads:")):
                key, value = line.split(":", 1)
                result[key] = value.strip()
    return result

def hit(url: str) -> dict:
    started = time.perf_counter()
    try:
        with urllib.request.urlopen(url, timeout=10) as resp:
            body = resp.read()
        return {
            "ok": True,
            "url": url,
            "status": resp.status,
            "elapsed_ms": round((time.perf_counter() - started) * 1000, 2),
            "bytes": len(body),
        }
    except Exception as exc:
        return {
            "ok": False,
            "url": url,
            "elapsed_ms": round((time.perf_counter() - started) * 1000, 2),
            "error": repr(exc),
        }

backend_health = health("http://127.0.0.1:5001/health")
gateway_health = health("http://127.0.0.1:3000/health")

started = time.perf_counter()
bench = subprocess.run(
    [
        "python3",
        str(root / "scripts/hybrid/run_benchmark_smoke.py"),
        "--base-url",
        "http://127.0.0.1:3000",
        "--project-name",
        "v0.1.0 live benchmark",
    ],
    cwd=str(root),
    capture_output=True,
    text=True,
)
duration = time.perf_counter() - started

summary = None
stdout = bench.stdout.strip()
if stdout:
    try:
        summary = json.loads(stdout)
    except Exception:
        summary = {"raw_stdout": stdout}

samples = []
with concurrent.futures.ThreadPoolExecutor(max_workers=16) as executor:
    futures = [executor.submit(hit, url) for _ in range(40) for url in ("http://127.0.0.1:5001/health", "http://127.0.0.1:3000/health")]
    for future in concurrent.futures.as_completed(futures):
        samples.append(future.result())

oks = [sample for sample in samples if sample["ok"]]
failures = [sample for sample in samples if not sample["ok"]]
latencies = sorted(sample["elapsed_ms"] for sample in oks)

payload = {
    "captured_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    "release": "v0.1.0",
    "host": {
        "uname": os.uname().sysname + " " + os.uname().release,
        "cpu_count": os.cpu_count(),
    },
    "backend_health": backend_health,
    "gateway_health": gateway_health,
    "processes": {
        "backend": proc_status(backend_pid),
        "gateway": proc_status(gateway_pid),
    },
    "benchmark": {
        "ok": bench.returncode == 0,
        "duration_seconds": round(duration, 2),
        "returncode": bench.returncode,
        "summary": summary,
        "stderr": bench.stderr.strip(),
    },
    "stress": {
        "request_count": len(samples),
        "success_count": len(oks),
        "failure_count": len(failures),
        "latency_ms": {
            "min": latencies[0] if latencies else None,
            "p50": round(statistics.median(latencies), 2) if latencies else None,
            "p95": round(latencies[max(0, int(len(latencies) * 0.95) - 1)], 2) if latencies else None,
            "max": latencies[-1] if latencies else None,
        },
        "failures": failures[:10],
    },
}

json_path.write_text(json.dumps(payload, indent=2), encoding="utf-8")
print(json.dumps(payload, indent=2))
PY

"$BENCH_PYTHON" "$ROOT_DIR/scripts/hybrid/generate_benchmark_report.py" \
  --input "$BENCH_JSON" \
  --output "$BENCH_REPORT" \
  --title "go-mirofish v0.1.0 benchmark report"
