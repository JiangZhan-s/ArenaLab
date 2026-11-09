$ErrorActionPreference = 'Stop'
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    & go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'vet failed' }
    & go test ./... -timeout=60s
    if ($LASTEXITCODE -ne 0) { throw 'starter tests failed' }
    & go run ./cmd/course check
    if ($LASTEXITCODE -ne 0) { throw 'course structure failed' }
} finally { Pop-Location }
