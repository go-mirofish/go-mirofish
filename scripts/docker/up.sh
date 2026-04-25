#!/usr/bin/env bash
# Start the Go gateway in Docker (API on :3000). UI is not built into this image — use `npm run dev`.
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"
exec docker compose up -d --build "$@"
