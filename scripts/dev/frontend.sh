#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/dev/common.sh
source "$SCRIPT_DIR/common.sh"

case "${1:-dev}" in
  dev)
    ROOT_DIR="$(repo_root)"
    cd "$ROOT_DIR"
    exec npm run dev --prefix frontend
    ;;
  build)
    ROOT_DIR="$(repo_root)"
    cd "$ROOT_DIR"
    exec npm run build --prefix frontend
    ;;
  preview)
    ROOT_DIR="$(repo_root)"
    cd "$ROOT_DIR"
    exec npm run preview --prefix frontend
    ;;
  *)
    echo "usage: $0 [dev|build|preview]" >&2
    exit 1
    ;;
esac
