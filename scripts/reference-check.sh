#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
: "${ARENALAB_ADMIN_DSN:?Use the isolated WSL verification database}"
mkdir -p bin
go version
go run ./cmd/course check
go test ./... -count=1
go build -race -tags reference,testhooks -o bin/arena-server ./cmd/arena-server
go build -race -tags reference -o bin/settlement-worker ./cmd/settlement-worker
go build -race -tags reference,testhooks -o bin/ds-sim ./cmd/ds-sim
go vet -tags reference ./...
go run -tags reference ./cmd/course gate 00
go test -race -tags reference,acceptance ./... -count=1 -timeout=120s
go test -race -tags reference,e2e ./internal/reference/testdriver -count=1 -v -timeout=20m
if grep -R -E 'WARNING: DATA RACE|^panic:' "${ARENALAB_ARTIFACTS:-artifacts/raw/reference}" --include='*.log'; then
  printf '%s\n' 'Child process race/panic detected'; exit 1
fi
