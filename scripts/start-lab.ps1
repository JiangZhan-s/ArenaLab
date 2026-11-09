param([Parameter(Mandatory)][ValidateRange(0,11)][int]$Lab)
$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
$target = Join-Path $root ('learning/lab{0:d2}' -f $Lab)
New-Item -ItemType Directory -Force -Path $target | Out-Null
foreach ($name in @('prelab','diff-review','experiment','exit-review')) {
    $dest = Join-Path $target ($name + '.md')
    if (-not (Test-Path -LiteralPath $dest)) {
        Copy-Item -LiteralPath (Join-Path $root ('templates/' + $name + '.md')) -Destination $dest
    }
}
Write-Output 'Templates ready. Fill predictions yourself before implementation.'
