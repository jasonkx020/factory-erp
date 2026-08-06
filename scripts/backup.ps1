#!/usr/bin/env pwsh
# 备份 SQLite / 提示 MySQL 备份
# 用法: powershell -File scripts/backup.ps1 [-SqlitePath data/erp.db] [-OutDir backups]
param(
  [string]$SqlitePath = "",
  [string]$OutDir = "backups"
)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$stamp = Get-Date -Format "yyyyMMdd_HHmmss"

if (-not $SqlitePath) {
  if (Test-Path "data/erp.db") { $SqlitePath = "data/erp.db" }
  elseif (Test-Path "data/erp_dev.db") { $SqlitePath = "data/erp_dev.db" }
}

if ($SqlitePath -and (Test-Path $SqlitePath)) {
  $dest = Join-Path $OutDir ("erp_sqlite_$stamp.db")
  Copy-Item $SqlitePath $dest -Force
  Write-Host "SQLite backup OK -> $dest"
} else {
  Write-Host "No local SQLite found. For MySQL run:"
  Write-Host '  mysqldump -u erp -perp erp_factory > backups/erp_mysql_' + $stamp + '.sql'
}

Write-Host "Rollback:"
Write-Host "  1) Stop API"
Write-Host "  2) Restore DB file / mysql < backup.sql"
Write-Host "  3) docker compose up -d  (or previous image tag)"
