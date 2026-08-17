# PostgreSQL backup helper (factory-erp)
# Usage:
#   .\scripts\backup.ps1
#   $env:ERP_DATABASE_DSN='postgres://erp:erp@127.0.0.1:5432/erp_factory?sslmode=disable'; .\scripts\backup.ps1

$ErrorActionPreference = "Stop"
$dsn = $env:ERP_DATABASE_DSN
if (-not $dsn) {
  $dsn = "postgres://erp:erp@127.0.0.1:5432/erp_factory?sslmode=disable"
}
$outDir = Join-Path $PSScriptRoot ".." "backups"
New-Item -ItemType Directory -Force -Path $outDir | Out-Null
$stamp = Get-Date -Format "yyyyMMdd_HHmmss"
$out = Join-Path $outDir "erp_factory_$stamp.dump"
Write-Host "pg_dump -> $out"
pg_dump --dbname=$dsn -Fc -f $out
Write-Host "done. restore with: pg_restore --clean --if-exists --dbname=<dsn> $out"
