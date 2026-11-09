#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
docker run --rm --network host -v "$(pwd):/src" -w /src -e GOFLAGS=-buildvcs=false \
 -e "ARENALAB_UID=$(id -u)" -e "ARENALAB_GID=$(id -g)" \
 -e 'ARENALAB_DSN=arena:arenalab-demo-local-only@tcp(127.0.0.1:23306)/arenalab_demo?parseTime=true' \
 -v arenalab-go-build:/root/.cache/go-build golang:1.25 bash -c 'trap '\''chown -R "$ARENALAB_UID:$ARENALAB_GID" artifacts/raw/browser 2>/dev/null || true'\'' EXIT; go run -tags reference ./cmd/demo-check'
