#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
docker run --rm --network host -v "$(pwd):/src" -w /src -e GOFLAGS=-buildvcs=false \
 -e 'ARENALAB_DSN=arena:arenalab-demo-local-only@tcp(127.0.0.1:23306)/arenalab_demo?parseTime=true' \
 -v arenalab-go-build:/root/.cache/go-build golang:1.25 go run -tags reference ./cmd/demo-check
