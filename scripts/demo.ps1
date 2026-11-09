param([string]$Distro = 'Ubuntu-22.04')
$ErrorActionPreference = 'Stop'
$scriptPath = Join-Path $PSScriptRoot 'demo.sh'
$wslPath = (& wsl -d $Distro -- wslpath -a $scriptPath).Trim()
if ($LASTEXITCODE -ne 0) { throw 'Cannot resolve script path in WSL' }
& wsl -d $Distro -- bash $wslPath
if ($LASTEXITCODE -ne 0) { throw 'WSL demo startup failed' }
