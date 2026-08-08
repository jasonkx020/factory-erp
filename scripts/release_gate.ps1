# Release gate: unit tests + health + pilot/finance smoke + unauthorized
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

  Write-Host "== 4) login + pilot/finance lists =="
  $login = Invoke-Api POST "/auth/login" @{ login_name = "admin"; password = "admin123"; client_type = "web" } $null
  Assert-Ok "login" $login
  $token = $login.data.access_token

  foreach ($p in @(
    "/purchase/weigh-tickets",
    "/inventory/balances",
    "/production/report-works",
    "/production/dispatches",
    "/finance/vouchers",
    "/finance/account-subjects",
    "/finance/fund-accounts"
  )) {
    Assert-Ok $p (Invoke-Api GET $p $null $token)
  }

  Write-Host "== 5) finance closed loop =="
  $codeS = "G" + (Get-Random -Maximum 99999)
  $codeF = "F" + (Get-Random -Maximum 99999)
  $subj = Invoke-Api POST "/finance/account-subjects" @{ code = $codeS; name = "gate-cash"; subject_type = "asset" } $token
  Assert-Ok "create subject" $subj
  $fund = Invoke-Api POST "/finance/fund-accounts" @{ code = $codeF; name = "gate-fund"; currency = "CNY"; balance = 0 } $token
  Assert-Ok "create fund" $fund
  Assert-Ok "ledger" (Invoke-Api POST "/finance/ledger-entries" @{
    direction = "in"; amount = 100; account_id = [int64]$fund.data.id; subject_id = [int64]$subj.data.id; remark = "gate"
  } $token)

  $period = Get-Date -Format "yyyy-MM"
  $sid = [int64]$subj.data.id
  $vch = Invoke-Api POST "/finance/vouchers" @{
    summary = "gate voucher"; period = $period
    lines = @(
      @{ subject_id = $sid; debit = 100; credit = 0 },
      @{ subject_id = $sid; debit = 0; credit = 100 }
    )
  } $token
  Assert-Ok "voucher create" $vch
  $vid = [int64]$vch.data.id
  Assert-Ok "voucher approve" (Invoke-Api POST "/finance/vouchers/$vid/approve" @{} $token)
  Assert-Ok "voucher post" (Invoke-Api POST "/finance/vouchers/$vid/post" @{} $token)

  $rejected = $false
  try {
    $badV = Invoke-Api POST "/finance/vouchers" @{
      summary = "bad"; period = $period
      lines = @(@{ subject_id = $sid; debit = 50; credit = 0 })
    } $token
    $badId = [int64]$badV.data.id
    $postBad = Invoke-Api POST "/finance/vouchers/$badId/post" @{} $token
    if ($postBad.code -eq 1) { throw "unbalanced post unexpectedly ok" }
    $rejected = $true
  } catch {
    if ($_.Exception.Message -match "unexpectedly ok") { throw }
    $rejected = $true
  }
  if (-not $rejected) { throw "unbalanced post not rejected" }
  Write-Host "OK  unbalanced post rejected"

  $wo = Invoke-Api POST "/finance/receipt-writeoffs" @{
    customer_id = 1; amount = 10; fund_account_id = [int64]$fund.data.id; sales_order_id = 1; line_amount = 10
  } $token
  Assert-Ok "writeoff" $wo
  Assert-Ok "writeoff confirm" (Invoke-Api POST "/finance/receipt-writeoffs/$([int64]$wo.data.id)/confirm" @{} $token)

  $y = [int](Get-Date -Format "yyyy"); $mo = [int](Get-Date -Format "MM")
  Assert-Ok "month close" (Invoke-Api POST "/finance/month-closes" @{ year = $y; month = $mo } $token)
  Assert-Ok "statements" (Invoke-Api POST "/finance/statements/generate" @{ period = $period } $token)

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
    Invoke-WebRequest -Method GET -Uri "$base/finance/vouchers" -UseBasicParsing | Out-Null
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
    "| login + weigh/inventory/report/finance lists | PASS |",
    "| station_pass_smoke (App过站+Admin写拒绝) | PASS |",
    "| delivery_loop (mobile+station smoke) | PASS |",
    "| finance loop subject->fund/ledger->voucher post->writeoff->month close->statements | PASS |",
    "| unbalanced voucher post rejected | PASS |",
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
