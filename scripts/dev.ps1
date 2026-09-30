param(
    [int]$ServerPort = 8080,
    [string]$FrontendDirectory = 'frontend'
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot

Write-Host 'Starting the EduPilot site server on port' $ServerPort
$server = Start-Process -PassThru -NoNewWindow -FilePath 'go' -ArgumentList @('run', './cmd/server', '-addr', ":$ServerPort") -WorkingDirectory $root

try {
    Push-Location (Join-Path $root $FrontendDirectory)
    $env:VITE_API_BASE_URL = "http://127.0.0.1:$ServerPort"
    npm run dev
}
finally {
    Pop-Location
    if (-not $server.HasExited) {
        $server.Kill()
    }
}
