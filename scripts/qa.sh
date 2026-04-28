#!/usr/bin/env bash
# QA bring-up: builds specd, seeds tmp/qa on first run, starts Vite (HMR)
# in the background, then runs Air to live-rebuild specd.
#
# Why this is a real bash script and not inline `cmds:` in Taskfile.yml:
# Task uses mvdan/sh, which rejects bare signal names ("INT"/"TERM") in
# the `trap` builtin. We need real bash so cleanup-on-exit works reliably.

set -euo pipefail

cd "$(dirname "$0")/.."

# 1. Build Go once so `specd init` is available for the seed step below.
CGO_ENABLED=0 go build -o tmp/specd .

# 2. Seed tmp/qa on first run; reuse afterwards so seeded data survives.
#    Force a clean slate with: rm -rf tmp/qa
if [ ! -f tmp/qa/.specd.json ]; then
  rm -rf tmp/qa
  mkdir -p tmp/qa
  (cd tmp/qa && ../../tmp/specd init --dir specd --username qa --skip-skills)
else
  echo "  Reusing existing tmp/qa project (delete it to re-init)"
fi

# 3. Ensure frontend deps exist.
if [ ! -d frontend/node_modules ]; then
  (cd frontend && pnpm install)
fi

# 4. Start Vite (HMR) on 5173 in the background. Track its PID so we can
#    tear it down on exit / Ctrl-C / SIGTERM.
(cd frontend && pnpm dev --host 127.0.0.1) &
VITE_PID=$!

cleanup() {
  kill "$VITE_PID" 2>/dev/null || true
  wait "$VITE_PID" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo ""
echo "  Vite dev:  http://localhost:5173 (direct, HMR)"
echo "  specd:     http://localhost:8000 (Go + spa-proxy → 5173)"
echo ""

# 5. Air rebuilds Go on .go changes; UI changes go through Vite HMR
#    via specd's --spa-proxy without a Go rebuild.
exec air -c .air-qa.toml
