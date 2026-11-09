#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
project=${ARENALAB_DEMO_PROJECT:-arenalab-demo}
docker compose -p "$project" -f deploy/compose.yaml up --build -d --wait
printf '%s\n' 'Open http://localhost:8080 in two tabs. Choose P0 and P1.' 'Local demo credentials only. MySQL uses the demo-mysql named volume.' 'Stop: docker compose -f deploy/compose.yaml stop (data retained).' 'Never use down -v unless you intend to erase this demo database.'
printf 'Compose project: %s\n' "$project"
if [[ ${1:-} == --attach ]]; then
  exec docker compose -p "$project" -f deploy/compose.yaml logs --follow --tail 5
fi
