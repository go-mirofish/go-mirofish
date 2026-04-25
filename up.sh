#!/usr/bin/env sh
# One-command stack: Go gateway + API + UI (all via Docker; Node not required on host).
cd "$(dirname "$0")" || exit 1
exec bash ./scripts/docker/up.sh
