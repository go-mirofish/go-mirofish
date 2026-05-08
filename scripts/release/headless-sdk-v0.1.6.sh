#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

echo "[headless-sdk] staging SDK-only files for release candidate"
git add -- \
  README.md \
  gateway/sdk/headless/doc.go \
  gateway/sdk/headless/headless.go \
  gateway/sdk/headless/headless_test.go \
  gateway/sdk/headless/example_test.go \
  gateway/sdk/headless/README.md \
  docs/report/headless-sdk-v0.1.0.md \
  docs/report/headless-sdk-release-checklist.md \
  docs/report/headless-sdk-release-candidate.md \
  docs/report/headless-sdk-v0.1.6-release-notes.md \
  scripts/release/README.md \
  scripts/release/headless-sdk-v0.1.6.sh

echo "[headless-sdk] running SDK package verification"
(
  cd gateway
  GOCACHE="${GOCACHE:-/tmp/go-build-cache}" \
  GOMODCACHE="${GOMODCACHE:-/tmp/go-mod-cache}" \
  go test ./sdk/headless
)

echo "[headless-sdk] running supporting package verification"
(
  cd gateway
  GOCACHE="${GOCACHE:-/tmp/go-build-cache}" \
  GOMODCACHE="${GOMODCACHE:-/tmp/go-mod-cache}" \
  go test ./internal/http/app ./internal/http/report ./internal/http/prepare ./internal/http/simulation ./internal/provider ./internal/report ./internal/graph
)

cat <<'EOF'

[headless-sdk] release prep complete

Suggested next steps:
1. Review staged diff
2. Paste the changelog block from:
   docs/report/headless-sdk-v0.1.6-release-notes.md
3. Run:
   npm run release
4. Commit with:
   git commit -m "feat(sdk): add headless sdk surface"

Release message:
Headless SDK v0.1.0 introduced in go-mirofish v0.1.6

EOF
