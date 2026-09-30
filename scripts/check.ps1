param(
    [switch]$SkipFrontend
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot

Push-Location $root
try {
    Write-Host 'gofmt' -ForegroundColor Cyan
    $unformatted = gofmt -l ./cmd ./internal
    if ($unformatted) {
        Write-Host 'unformatted files:' -ForegroundColor Red
        $unformatted | ForEach-Object { Write-Host "  $_" }
        exit 1
    }

    Write-Host 'go vet' -ForegroundColor Cyan
    go vet ./...
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    Write-Host 'go test' -ForegroundColor Cyan
    go test ./... -count=1
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    if (-not $SkipFrontend) {
        Push-Location (Join-Path $root 'frontend')
        try {
            Write-Host 'frontend typecheck' -ForegroundColor Cyan
            npm run typecheck
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

            Write-Host 'frontend tests' -ForegroundColor Cyan
            npm run test
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        }
        finally {
            Pop-Location
        }
    }

    Write-Host 'All checks passed.' -ForegroundColor Green
}
finally {
    Pop-Location
}
