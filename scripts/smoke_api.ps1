# Smoke test against local erp-api (port 18080)
$ErrorActionPreference = "Stop"
$base = "http://127.0.0.1:18080/api/v1"

function Invoke-Api($method, $path, $body, $token) {
  $headers = @{ "Content-Type" = "application/json" }
  if ($token) { $headers["Authorization"] = "Bearer $token" }
  $params = @{ Method = $method; Uri = "$base$path"; Headers = $headers }
  if ($null -ne $body) { $params.Body = ($body | ConvertTo-Json -Depth 8 -Compress) }
  return Invoke-RestMethod @params
}

Write-Host "== health =="
$h = Invoke-Api GET "/health" $null $null
if ($h.code -ne 1) { throw "health failed" }

Write-Host "== login =="
$login = Invoke-Api POST "/auth/login" @{ login_name = "admin"; password = "admin123"; client_type = "web" } $null
if ($login.code -ne 1) { throw "login failed: $($login.msg)" }
$token = $login.data.access_token

Write-Host "== me =="
$me = Invoke-Api GET "/auth/me" $null $token
if ($me.code -ne 1) { throw "me failed" }

$samples = @(
  @{ m = "GET"; p = "/product/products" },
  @{ m = "GET"; p = "/inventory/balances" },
  @{ m = "GET"; p = "/production/processes" },
  @{ m = "GET"; p = "/iam/roles" },
  @{ m = "GET"; p = "/hr/employees" },
  @{ m = "GET"; p = "/crm/customers" },
  @{ m = "GET"; p = "/sales/orders" },
  @{ m = "GET"; p = "/purchase/suppliers" },
  @{ m = "GET"; p = "/finance/vouchers" },
  @{ m = "GET"; p = "/asset/fixed-assets" },
  @{ m = "GET"; p = "/report/dashboards/boss" }
)
foreach ($s in $samples) {
  Write-Host "== $($s.m) $($s.p) =="
  $r = Invoke-Api $s.m $s.p $null $token
  if ($r.code -ne 1) { throw "$($s.p) failed: $($r.msg)" }
  if ($r.msg -eq "NOT_IMPLEMENTED") { throw "$($s.p) still NOT_IMPLEMENTED" }
}

Write-Host "SMOKE OK"
