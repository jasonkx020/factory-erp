# Delivery loop: run all gate checks until green.
# Usage: powershell -File scripts/delivery_loop.ps1
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$report = "docs/DELIVERY_LOOP_REPORT.md"
$steps = @()
$fail = 0
$ts = Get-Date -Format "yyyy-MM-dd HH:mm:ss"

function Step($name, [scriptblock]$block) {
  Write-Host "== $name =="
  try {
    & $block
    $script:steps += "| $name | PASS |"
    Write-Host "OK  $name"
  } catch {
    $script:steps += "| $name | FAIL |"
    Write-Host "FAIL $name : $_"
    $script:fail++
  }
}

Step "go test ./internal/biz" { go test ./internal/biz/ -count=1 | Out-Null; if ($LASTEXITCODE -ne 0) { throw "go test failed" } }
Step "release_gate.ps1" { & "$PSScriptRoot/release_gate.ps1" }
Step "mobile_delivery_smoke" { go run ./cmd/mobile_delivery_smoke | Out-Null; if ($LASTEXITCODE -ne 0) { throw "mobile smoke failed" } }
Step "station_pass_smoke" { go run ./cmd/station_pass_smoke | Out-Null; if ($LASTEXITCODE -ne 0) { throw "station smoke failed" } }
Step "openapi-coverage" { go run ./cmd/erp-tools openapi-coverage | Out-Null; if ($LASTEXITCODE -ne 0) { throw "coverage failed" } }

$manual = @(
  "",
  "## Manual walkthrough (required before sign-off)",
  "",
  "- [ ] u_piece: 过站 → 确认 → 我的核对",
  "- [ ] u_fixed: 过站（无计件金额）",
  "- [ ] u_purchase/u_qc: 过磅收货",
  "- [ ] u_warehouse: 待入库",
  "- [ ] u_foreman: 班组（无每箱派工）",
  "- [ ] admin: 生产 Hub 无报工创建表单"
)

$lines = @(
  "# Delivery Loop Report",
  "",
  "Generated: $ts",
  "",
  "| Step | Result |",
  "|------|--------|"
) + $steps + $manual + @("")

$utf8 = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllLines((Join-Path $Root $report), $lines, $utf8)

if ($fail -gt 0) {
  Write-Host ""
  Write-Host "DELIVERY_LOOP_FAIL count=$fail (see $report)"
  exit 1
}
Write-Host ""
Write-Host "DELIVERY_LOOP_OK -> $report"
