#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/dev/common.sh
source "$SCRIPT_DIR/common.sh"

ROOT_DIR="$(repo_root)"
mode="${1:-sprint}"
if [[ $# -gt 0 ]]; then
  shift
fi

cd "$ROOT_DIR"

profile="${PROFILE:-${MIROFISH_BENCHMARK_PROFILE:-}}"

case "$mode" in
  live)
    ( cd "$ROOT_DIR/gateway" && exec go run ./cmd/mirofish-hybrid live-benchmark "$@" )
    ;;
  merge-bundled)
    ( cd "$ROOT_DIR/gateway" && exec go run ./cmd/mirofish-hybrid merge-bundled "$@" )
    ;;
  examples)
    exec go run ./gateway/cmd/go-mirofish-examples --all --bench-only --profile "${profile:-medium}" "$@"
    ;;
  smoke)
    exec go run ./gateway/cmd/go-mirofish-examples --all --smoke-only --profile "${profile:-small}" "$@"
    ;;
  benchmark)
    BENCH_BASE_URL="${BENCH_BASE_URL:-http://127.0.0.1:3000}"
    cd "$ROOT_DIR/gateway"
    exec go run ./cmd/benchmark --base-url "$BENCH_BASE_URL" "$@"
    ;;
  *)
    echo "usage: $0 [live|merge-bundled|examples|smoke|benchmark]" >&2
    exit 1
    ;;
esac
