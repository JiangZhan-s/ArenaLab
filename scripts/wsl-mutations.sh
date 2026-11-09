#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
docker compose -f deploy/compose.verify.yaml up -d --wait
docker run --rm --network host -v "$(pwd):/src" -v arenalab-go-build:/root/.cache/go-build -w /src \
 -e GOFLAGS=-buildvcs=false -e ARENALAB_REDIS_ADDR=127.0.0.1:16379 \
 -e 'ARENALAB_ADMIN_DSN=root:arenalab-local-test-only@tcp(127.0.0.1:13306)/?parseTime=true' \
 golang:1.25 go run ./cmd/mutation
