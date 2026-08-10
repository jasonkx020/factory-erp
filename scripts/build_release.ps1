# Build web into embed dir and compile erp-api.exe (Windows).
# Usage (from repo root):
#   powershell -File scripts/build_release.ps1
#   powershell -File scripts/build_release.ps1 -SkipWeb
param(
  [switch]$SkipWeb,
  [string]$OutDir = "dist-release"
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$embedDist = Join-Path $Root "internal\webui\dist"
$webDist = Join-Path $Root "web\dist"

if (-not $SkipWeb) {
  Write-Host "→ npm run build:dist"
  Push-Location (Join-Path $Root "web")
  npm run build:dist
  if ($LASTEXITCODE -ne 0) { throw "build:dist failed" }
  Pop-Location
}

if (-not (Test-Path (Join-Path $webDist "index.html"))) {
  throw "missing web/dist/index.html — run without -SkipWeb or npm run build:dist first"
}

Write-Host "→ sync web/dist → internal/webui/dist"
if (Test-Path $embedDist) {
  Remove-Item -Recurse -Force $embedDist
}
New-Item -ItemType Directory -Path $embedDist | Out-Null
Copy-Item -Path (Join-Path $webDist "*") -Destination $embedDist -Recurse -Force

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$exe = Join-Path $OutDir "erp-api.exe"
Write-Host "→ go build -o $exe ./cmd/erp-api"
go build -o $exe ./cmd/erp-api
if ($LASTEXITCODE -ne 0) { throw "go build failed" }

Write-Host ""
Write-Host "Done: $exe"
Write-Host "Run with empty web_root to use embedded UI, or set server.web_root / ERP_WEB_ROOT for external."
Write-Host "  .\$OutDir\erp-api.exe -config configs\erp.prod.example.yaml"
