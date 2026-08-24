# Release gate: unit tests + health + cassava delivery smoke
# Usage: powershell -File scripts/release_gate.ps1
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$base = "http://127.0.0.1:18080/api/v1"
$cfg = "configs/erp.gate.yaml"
$report = "docs/GATE_SIGN_OFF.md"
$script:proc = $null

function Invoke-Api($method, $path, $body, $token) {
  $headers = @{ "Content-Type" = "application/json" }
  if ($token) { $headers["Authorization"] = "Bearer $token" }
  $params = @{ Method = $method; Uri = "$base$path"; Headers = $headers }
  if ($null -ne $body) { $params.Body = ($body | ConvertTo-Json -Depth 10 -Compress) }
  return Invoke-RestMethod @params
}

function Assert-Ok($name, $resp) {
  if ($null -eq $resp -or $resp.code -ne 1) {
    throw "FAIL $name : $($resp | ConvertTo-Json -Compress -Depth 6)"
  }
  Write-Host "OK  $name"
}

function Assert-Fail($name, $resp) {
  if ($null -eq $resp -or $resp.code -eq 1) {
    throw "FAIL $name : expected error, got $($resp | ConvertTo-Json -Compress -Depth 4)"
  }
  Write-Host "OK  $name (rejected)"
}

function Stop-GateApi {
  try {
    $conns = Get-NetTCPConnection -LocalPort 18080 -State Listen -ErrorAction SilentlyContinue
    foreach ($c in $conns) {
      Stop-Process -Id $c.OwningProcess -Force -ErrorAction SilentlyContinue
    }
  } catch {}
  if ($script:proc -and -not $script:proc.HasExited) {
    Stop-Process -Id $script:proc.Id -Force -ErrorAction SilentlyContinue
  }
  Start-Sleep -Seconds 1
}

Write-Host "== 1) go test =="
go test ./internal/biz/ -count=1
if ($LASTEXITCODE -ne 0) { throw "go test failed" }

Write-Host "== 2) start api (demo:false, strong JWT) =="
Stop-GateApi
New-Item -ItemType Directory -Force -Path data | Out-Null
if (Test-Path "data/erp_gate.db") { Remove-Item "data/erp_gate.db" -Force }

$script:proc = Start-Process -FilePath "go" -ArgumentList @("run", "./cmd/erp-api", "-config", $cfg) -PassThru -NoNewWindow -WorkingDirectory $Root
$ready = $false
for ($i = 0; $i -lt 45; $i++) {
  Start-Sleep -Seconds 1
  try {
    $r = Invoke-RestMethod -Uri "$base/live" -TimeoutSec 2
    if ($r.code -eq 1) { $ready = $true; break }
  } catch {}
}
if (-not $ready) { throw "api failed to become live" }

try {
  Write-Host "== 3) live/ready/health/metrics =="
  Assert-Ok "live" (Invoke-RestMethod -Uri "$base/live")
  $rdy = Invoke-RestMethod -Uri "$base/ready"
  Assert-Ok "ready" $rdy
  if ($rdy.data.db -ne "up") { throw "ready db not up" }
  Assert-Ok "health" (Invoke-RestMethod -Uri "$base/health")
  $metrics = Invoke-WebRequest -Uri "$base/metrics" -UseBasicParsing
  if ($metrics.Content -notmatch "erp_http_requests_total") { throw "metrics missing counter" }
  Write-Host "OK  metrics"

  Write-Host "== 4) login + cassava scope lists =="
  $login = Invoke-Api POST "/auth/login" @{ login_name = "admin"; password = "admin123"; client_type = "web" } $null
  Assert-Ok "login" $login
  $token = $login.data.access_token

  foreach ($p in @(
    "/purchase/weigh-tickets",
    "/purchase/farmers",
    "/inventory/balances",
    "/inventory/availability",
    "/production/station-flow-logs",
    "/production/process-wip",
    "/production/trace-productions",
    "/finance/cost-accountings",
    "/finance/cost-traces",
    "/report/dashboards/production",
    "/report/dashboards/live",
    "/report/dashboards/warehouse",
    "/report/daily",
    "/report/stock-ledger",
    "/report/payroll-reconcile",
    "/report/cost-period-summary"
  )) {
    Assert-Ok $p (Invoke-Api GET $p $null $token)
  }

  Write-Host "== 5) removed domains return FEATURE_REMOVED =="
  Assert-Fail "sales removed" (Invoke-Api GET "/sales/orders" $null $token)
  Assert-Fail "crm removed" (Invoke-Api GET "/crm/customers" $null $token)
  Assert-Fail "finance voucher removed" (Invoke-Api GET "/finance/vouchers" $null $token)
  Assert-Fail "boss dashboard removed" (Invoke-Api GET "/report/dashboards/boss" $null $token)

  Write-Host "== 6) delivery smokes =="
  $env:API_BASE = "$base"
  go run ./cmd/mobile_delivery_smoke
  if ($LASTEXITCODE -ne 0) { throw "mobile_delivery_smoke failed" }
  Write-Host "OK  mobile_delivery_smoke"
  go run ./cmd/station_pass_smoke
  if ($LASTEXITCODE -ne 0) { throw "station_pass_smoke failed" }
  Write-Host "OK  station_pass_smoke"

  Write-Host "== 7) unauthorized =="
  $denied = $false
  try {
    Invoke-WebRequest -Method GET -Uri "$base/finance/cost-accountings" -UseBasicParsing | Out-Null
  } catch {
    $denied = $true
  }
  if (-not $denied) { throw "expected 401 without token" }
  Write-Host "OK  unauthorized without token"

  Write-Host "== 8) sign-off =="
  $now = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
  $lines = @(
    "# Release Gate Sign-off",
    "",
    "Generated: $now",
    "Config: ``$cfg`` (seed.demo=false, strong JWT, CORS locked)",
    "",
    "| Check | Result |",
    "|-------|--------|",
    "| go test ./internal/biz | PASS |",
    "| /live /ready(db=up) /health /metrics | PASS |",
    "| login + cassava scope API lists | PASS |",
    "| removed sales/crm/full-finance rejected | PASS |",
    "| station_pass_smoke | PASS |",
    "| mobile_delivery_smoke | PASS |",
    "| request without token rejected | PASS |",
    "",
    "Sign: _______________  Date: _______________",
    ""
  )
  $utf8 = New-Object System.Text.UTF8Encoding $false
  [System.IO.File]::WriteAllLines((Join-Path $Root $report), $lines, $utf8)
  Write-Host "GATE_OK -> $report"
}
finally {
  Stop-GateApi
}
