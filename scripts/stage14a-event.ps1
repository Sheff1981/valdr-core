param(
    [Parameter(Mandatory = $true)][string]$SessionDir,
    [Parameter(Mandatory = $true)]
    [ValidateSet("peer_exchange","bootstrap_loss","restart","db_verify","mining","transaction","mempool","desktop_sync","package_start","wallet_backup_restore","reorg_observation","crash_error","note")]
    [string]$EventType,
    [ValidateSet("pass","fail","observed","not_applicable")][string]$Result = "observed",
    [string]$Note = ""
)

$ErrorActionPreference = "Stop"
$manifestPath = Join-Path $SessionDir "manifest.json"
$eventsPath = Join-Path $SessionDir "events.jsonl"
if (-not (Test-Path -LiteralPath $manifestPath)) { throw "missing manifest: $manifestPath" }

$manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
if ([string]::IsNullOrWhiteSpace([string]$manifest.session_group_id)) { throw "manifest missing session_group_id" }
if ([string]::IsNullOrWhiteSpace([string]$manifest.machine_id)) { throw "manifest missing machine_id" }

$event = [ordered]@{
    schema = "valdr-stage14a-event-v1"
    recorded_at = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    session_group_id = $manifest.session_group_id
    machine_id = $manifest.machine_id
    event_type = $EventType
    result = $Result
    note = $Note
    evidence_kind = "operator_recorded"
}
Add-Content -LiteralPath $eventsPath -Value ($event | ConvertTo-Json -Depth 10 -Compress) -Encoding utf8

$names = @("manifest.json", "snapshots.jsonl", "summary.json", "events.jsonl")
$missing = @($names | Where-Object { -not (Test-Path -LiteralPath (Join-Path $SessionDir $_)) })
if ($missing.Count -gt 0) { return }
$lines = foreach ($name in $names) {
    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $SessionDir $name)).Hash.ToLowerInvariant()
    "$hash  $name"
}
$lines | Set-Content -LiteralPath (Join-Path $SessionDir "SHA256SUMS") -Encoding ascii
$event | ConvertTo-Json -Depth 10 -Compress
