#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
run_id=$(date -u +%Y%m%dT%H%M%SZ)
run_dir="artifacts/raw/runs/$run_id"
mkdir -p "$run_dir"
commit=$(git -c safe.directory="$(pwd)" rev-parse HEAD)
if test -n "$(git -c safe.directory="$(pwd)" status --porcelain)"; then commit="$commit-dirty"; fi
{ printf 'commit=%s\n' "$commit"; uname -a; lscpu; free -m; docker version; docker compose version; } > "$run_dir/environment.txt"
docker compose -f deploy/compose.verify.yaml up -d --wait
docker run --rm --network host \
  -v "$(pwd):/src" \
  -v arenalab-go-mod:/go/pkg/mod \
  -v arenalab-go-build:/root/.cache/go-build \
  -w /src \
  -e GOFLAGS=-buildvcs=false \
  -e "ARENALAB_COMMIT=$commit" \
  -e "ARENALAB_ARTIFACTS=/src/$run_dir/reference" \
  -e ARENALAB_REDIS_ADDR=127.0.0.1:16379 \
  -e 'ARENALAB_ADMIN_DSN=root:arenalab-local-test-only@tcp(127.0.0.1:13306)/?parseTime=true' \
  -e "ARENALAB_GATES=${1:-02,03,04,05,06,07,08,09,10,11,12,13,14}" \
  golang:1.25 bash scripts/reference-check.sh 2>&1 | tee "$run_dir/gates.log"
printf 'Evidence: %s\n' "$run_dir"
docker image inspect golang:1.25 mysql:8.4 redis:7.4 --format '{{.RepoTags}} {{.Id}}' >> "$run_dir/environment.txt"
