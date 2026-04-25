#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/dev/common.sh
source "$SCRIPT_DIR/common.sh"

ROOT_DIR="$(repo_root)"

rm -rf "$ROOT_DIR/gateway/bin"
rm -rf "$ROOT_DIR/frontend/dist"
rm -rf "$ROOT_DIR/benchmark/results/runtime-sprint"
rm -rf "$ROOT_DIR/backend/.mirofish"
