$ErrorActionPreference = "Stop"
$tmp = Join-Path $env:RUNNER_TEMP "valdr-stage14a-ps-smoke"
Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
$bin = Join-Path $tmp "bin"
$data = Join-Path $tmp "data"
$out = Join-Path $tmp "out"
New-Item -ItemType Directory -Force -Path $bin,$data,$out | Out-Null

@'
@echo off
if "%1"=="version" (
  echo VALDR VDR valdrd 0.2.0-dev
  exit /b 0
)
if "%1"=="status" (
  echo {"network":"testnet2","chain_id":"valdr-testnet-2","height":42,"tip_hash":"0000000000000000000000000000000000000000000000000000000000000042","chainwork":"0000000000000000000000000000000000000000000000000000000000001234","target":"0000031b5d43afe99ee43470e1337c3642e9d9254926038fdf6d1a2e57aaa21f"}
  exit /b 0
)
exit /b 2
'@ | Set-Content -LiteralPath (Join-Path $bin "valdrd.cmd") -Encoding ascii

@'
@echo off
if "%1"=="peers" (
  echo {"peers":[]}
  exit /b 0
)
if "%1"=="mining" if "%2"=="info" (
  echo {"height":42,"current_target":"0000031b5d43afe99ee43470e1337c3642e9d9254926038fdf6d1a2e57aaa21f","hashrate":0}
  exit /b 0
)
exit /b 2
'@ | Set-Content -LiteralPath (Join-Path $bin "valdr-cli.cmd") -Encoding ascii

$env:PATH = "$bin;$env:PATH"
$sessionScript = Join-Path $PSScriptRoot "stage14a-session-validate.ps1"
$eventScript = Join-Path $PSScriptRoot "stage14a-event.ps1"
& $sessionScript -SessionGroup "ci-group-1" -MachineId "ci-win-1" -SessionId "ci-win-session" -Operator "ci" -SourceCommit "0123456789abcdef0123456789abcdef01234567" -BootstrapRoute "ci-fake-bootstrap" -Data $data -DurationSeconds 0 -IntervalSeconds 1 -OutputDir $out

$session = Join-Path $out "ci-win-session"
foreach ($name in @("manifest.json","snapshots.jsonl","summary.json","events.jsonl","SHA256SUMS")) {
    if (-not (Test-Path -LiteralPath (Join-Path $session $name))) { throw "missing $name" }
}
$manifest = Get-Content -LiteralPath (Join-Path $session "manifest.json") -Raw | ConvertFrom-Json
$summary = Get-Content -LiteralPath (Join-Path $session "summary.json") -Raw | ConvertFrom-Json
if ($manifest.network -ne "testnet2") { throw "wrong network" }
if ($manifest.chain_id -ne "valdr-testnet-2") { throw "wrong chain id" }
if ($summary.snapshot_count -ne 1 -or $summary.successful_snapshot_count -ne 1) { throw "wrong snapshot counts" }
if ($summary.stage14a_pass -ne $false) { throw "local session must not claim Stage14A PASS" }

& $eventScript -SessionDir $session -EventType "package_start" -Result "pass" -Note "Windows PowerShell smoke" | Out-Null
$eventLines = @(Get-Content -LiteralPath (Join-Path $session "events.jsonl") | Where-Object { $_ })
if ($eventLines.Count -ne 1) { throw "event was not recorded" }
$event = $eventLines[0] | ConvertFrom-Json
if ($event.event_type -ne "package_start" -or $event.result -ne "pass") { throw "wrong event" }

$duplicateRejected = $false
try {
    & $sessionScript -SessionGroup "ci-group-1" -MachineId "ci-win-1" -SessionId "ci-win-session" -Operator "ci" -SourceCommit "0123456789abcdef0123456789abcdef01234567" -BootstrapRoute "ci-fake-bootstrap" -Data $data -DurationSeconds 0 -IntervalSeconds 1 -OutputDir $out | Out-Null
} catch {
    $duplicateRejected = $true
}
if (-not $duplicateRejected) { throw "duplicate Stage14A session unexpectedly overwrote evidence" }

Write-Output "Stage14A PowerShell evidence tooling smoke passed"
