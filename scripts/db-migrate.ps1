# Build and run erp-db migration CLI
# Usage:
#   .\scripts\db-migrate.ps1 status
#   .\scripts\db-migrate.ps1 upgrade --all
#   .\scripts\db-migrate.ps1 validate

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
  go build -o bin/erp-db.exe ./cmd/erp-db
  & "$root\bin\erp-db.exe" @args
} finally {
  Pop-Location
}
