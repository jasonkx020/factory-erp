# 默认只生成合并 SQL；加 -Apply 才调用 mysql 导入
# 示例：
#   .\install.ps1
#   .\install.ps1 -Apply -MysqlArgs "-uroot","-pYourPassword"
param(
  [switch]$Apply,
  [string[]]$MysqlArgs = @("-uroot", "-p")
)

$ErrorActionPreference = "Stop"
$root = $PSScriptRoot
$out = Join-Path $root "erp_factory_all.sql"

$files = @(
  "schema\00_init.sql",
  "schema\01_common.sql",
  "schema\02_iam.sql",
  "schema\03_product_inventory.sql",
  "schema\04_production_payroll.sql",
  "schema\05_hr.sql",
  "schema\06_crm_sales_purchase.sql",
  "schema\07_finance_asset.sql",
  "schema\08_approval_system_report.sql",
  "seed\01_iam_seed.sql"
)

$sb = New-Object System.Text.StringBuilder
foreach ($rel in $files) {
  $path = Join-Path $root $rel
  if (-not (Test-Path $path)) { throw "Missing: $path" }
  [void]$sb.AppendLine("-- ===== FILE: $rel =====")
  [void]$sb.AppendLine((Get-Content -Path $path -Raw -Encoding UTF8))
  [void]$sb.AppendLine()
}
[System.IO.File]::WriteAllText($out, $sb.ToString(), [System.Text.UTF8Encoding]::new($false))
Write-Host "Generated: $out"
Write-Host ("CREATE TABLE count: " + ([regex]::Matches((Get-Content $out -Raw), '(?m)^CREATE TABLE')).Count)

if (-not $Apply) {
  Write-Host "Skip import (pass -Apply to import). Example:"
  Write-Host "  mysql -uroot -p --default-character-set=utf8mb4 < `"$out`""
  exit 0
}

$mysql = Get-Command mysql -ErrorAction SilentlyContinue
if (-not $mysql) { throw "mysql client not found in PATH" }
Get-Content $out -Raw -Encoding UTF8 | & mysql @MysqlArgs --default-character-set=utf8mb4
Write-Host "Import done."
