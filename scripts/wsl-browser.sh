#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
docker run --rm --init --network host --shm-size=1g \
  -v "$(pwd):/src" -w /src \
  -v arenalab-browser-modules:/src/node_modules \
  mcr.microsoft.com/playwright:v1.55.1-noble \
  bash -lc 'npm ci --ignore-scripts && node scripts/browser-check.mjs'
bash scripts/check-demo-db.sh
