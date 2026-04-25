#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

mode="${1:-package}"
if [[ $# -gt 0 ]]; then
  shift
fi

case "$mode" in
  package)
    if command -v python3 >/dev/null 2>&1; then
      exec python3 scripts/release/package_gateway_release.py "$@"
    elif command -v python >/dev/null 2>&1; then
      exec python scripts/release/package_gateway_release.py "$@"
    else
      echo "[release] python not found; running gateway go test directly" >&2
      (cd "$ROOT_DIR/gateway" && exec go test ./...)
    fi
    ;;
  notes)
    exec node scripts/release/extract-release-notes.cjs "$@"
    ;;
  changelog)
    exec node scripts/release/update-changelog.cjs "$@"
    ;;
  *)
    echo "usage: $0 [package|notes|changelog]" >&2
    exit 1
    ;;
esac
