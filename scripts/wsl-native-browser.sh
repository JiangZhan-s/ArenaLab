#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
runtime_dir=$(mktemp -d /tmp/arenalab-browser.XXXXXX)
printf 'Isolated browser runtime: %s\n' "$runtime_dir"
npm install --prefix "$runtime_dir" --ignore-scripts playwright@1.55.1
node --check "$runtime_dir/node_modules/playwright/lib/program.js"
export PLAYWRIGHT_BROWSERS_PATH="$runtime_dir/browsers"
node "$runtime_dir/node_modules/playwright/cli.js" install chromium
export ARENALAB_PLAYWRIGHT_MODULE="$runtime_dir/node_modules/playwright/index.mjs"
node scripts/browser-check.mjs
bash scripts/check-demo-db.sh
