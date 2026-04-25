#!/usr/bin/env bash

repo_root() {
  cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd
}

run_in_root() {
  local root
  root="$(repo_root)"
  cd "$root"
  "$@"
}

have_cmd() {
  command -v "$1" >/dev/null 2>&1
}

log() {
  printf '[%s] %s\n' "$1" "$2" >&2
}
