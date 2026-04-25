#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/dev/common.sh
source "$SCRIPT_DIR/common.sh"

case "${1:-run}" in
  run)
    ROOT_DIR="$(repo_root)"
    cd "$ROOT_DIR"
    exec node scripts/dev/run-gateway.cjs
    ;;
  build)
    ROOT_DIR="$(repo_root)"
    cd "$ROOT_DIR"
    exec node scripts/dev/build-gateway.cjs
    ;;
  *)
    echo "usage: $0 [run|build]" >&2
    exit 1
    ;;
esac
