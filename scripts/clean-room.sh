#!/usr/bin/env bash
# Run only when the demo ports are free; never stop somebody else's project.
set -euo pipefail
cd "$(dirname "$0")/.."
source_root=$(pwd)
clean_dir=$(mktemp -d /tmp/arenalab-clean.XXXXXX)
git -c safe.directory="$source_root" clone --no-hardlinks "$source_root" "$clean_dir/repo"
cd "$clean_dir/repo"
printf 'Clean checkout: %s\n' "$clean_dir/repo"
git rev-parse HEAD
test -z "$(git status --porcelain)"
bash scripts/wsl-reference.sh
bash scripts/wsl-mutations.sh
export ARENALAB_DEMO_PROJECT="arenalab-clean-$(date -u +%Y%m%d%H%M%S)"
bash scripts/demo.sh
bash scripts/wsl-browser.sh
mkdir -p "$source_root/artifacts/raw"
tar -czf "$source_root/artifacts/raw/$ARENALAB_DEMO_PROJECT.tar.gz" artifacts/raw
printf 'PASS clean checkout. Evidence: %s/artifacts/raw/%s.tar.gz\n' "$source_root" "$ARENALAB_DEMO_PROJECT"
printf 'Demo remains running. Stop (keeps data): docker compose -p %s -f %s/deploy/compose.yaml stop\n' "$ARENALAB_DEMO_PROJECT" "$clean_dir/repo"
