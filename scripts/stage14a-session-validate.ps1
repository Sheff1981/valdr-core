param(
    [Parameter(Mandatory = $true)][ValidatePattern('^[A-Za-z0-9._-]+$')][string]$SessionGroup,
    [Parameter(Mandatory = $true)][ValidatePattern('^[A-Za-z0-9._-]+$')][string]$MachineId,
    [string]$SessionId = "",
    [string]$Operator = "",
    [Parameter(Mandatory = $true)][ValidatePattern('^[0-9a-fA-F]{40}$')][string]$SourceCommit,
    [string]$BootstrapRoute = "",
    [string]$Node = "http://127.0.0.1:17332",
    [string]$Data = ".\data",
    [ValidateRange(0, 2147483647)][int]$DurationSeconds = 10800,
    [ValidateRange(1, 2147483647)][int]$IntervalSeconds = 60,
    [string]$OutputDir = ".\stage14a-evidence"
)

$ErrorActionPreference = "Stop"

function Get-UtcIso {
    return (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
}

function Get-DirectoryBytes([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) { return [int64]0 }
    $sum = [int64]0
    Get-ChildItem -LiteralPath $Path -File -Recurse -Force -ErrorAction SilentlyContinue | ForEach-Object {
        $sum += [int64]$_.Length
    }
    return $sum
}

function Append-JsonLine([string]$Path, $Object) {
    $line = $Object | ConvertTo-Json -Depth 20 -Compress
    Add-Content -LiteralPath $Path -Value $line -Encoding utf8
}

function Invoke-Captured([scriptblock]$Command) {
    $output = & $Command 2>&1
    $code = $LASTEXITCODE
    return [pscustomobject]@{
        ExitCode = [int]$code
        Text = (($output | Out-String).Trim())
    }
}

if (-not (Get-Command valdrd -ErrorAction SilentlyContinue)) { throw "valdrd not found in PATH" }
if (-not (Get-Command valdr-cli -ErrorAction SilentlyContinue)) { throw "valdr-cli not found in PATH" }

if ([string]::IsNullOrWhiteSpace($SessionId)) {
    $SessionId = "$(Get-Date -AsUTC -Format 'yyyyMMddTHHmmssZ')-$MachineId"
}
if ($SessionId -notmatch '^[A-Za-z0-9._-]+$') { throw "invalid session id" }

$sessionDir = Join-Path $OutputDir $SessionId
if (Test-Path -LiteralPath $sessionDir) {
    $existing = Get-ChildItem -LiteralPath $sessionDir -Force -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($null -ne $existing) { throw "session evidence directory already exists and is not empty: $sessionDir" }
} else {
    New-Item -ItemType Directory -Path $sessionDir -Force | Out-Null
}

$manifestPath = Join-Path $sessionDir "manifest.json"
$snapshotsPath = Join-Path $sessionDir "snapshots.jsonl"
$summaryPath = Join-Path $sessionDir "summary.json"
$eventsPath = Join-Path $sessionDir "events.jsonl"
$sumsPath = Join-Path $sessionDir "SHA256SUMS"

$version = (Invoke-Captured { valdrd version }).Text
$startedAt = Get-UtcIso

$manifest = [ordered]@{
    schema = "valdr-stage14a-session-v1"
    session_id = $SessionId
    session_group_id = $SessionGroup
    operator = $(if ($Operator) { $Operator } else { $null })
    machine_id = $MachineId
    source_commit = $SourceCommit.ToLowerInvariant()
    bootstrap_route = $(if ($BootstrapRoute) { $BootstrapRoute } else { $null })
    rpc_endpoint = $Node
    network = "testnet2"
    chain_id = "valdr-testnet-2"
    planned_duration_seconds = $DurationSeconds
    interval_seconds = $IntervalSeconds
    started_at = $startedAt
    valdrd_version = $version
    acceptance_scope = "local-session-evidence-only"
}
$manifest | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $manifestPath -Encoding utf8
Set-Content -LiteralPath $snapshotsPath -Value "" -NoNewline -Encoding utf8
Set-Content -LiteralPath $eventsPath -Value "" -NoNewline -Encoding utf8

$deadline = [DateTimeOffset]::UtcNow.AddSeconds($DurationSeconds)
$previousHeight = $null
$previousWork = $null

do {
    $observedAt = Get-UtcIso
    $observedEpoch = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    $diskBytes = Get-DirectoryBytes $Data

    $statusCall = Invoke-Captured { valdrd status --node $Node }
    if ($statusCall.ExitCode -ne 0) {
        Append-JsonLine $snapshotsPath ([ordered]@{
            observed_at = $observedAt
            observed_epoch = $observedEpoch
            observation_error = [ordered]@{
                component = "valdrd-status"
                exit_code = $statusCall.ExitCode
                message = $(if ($statusCall.Text.Length -gt 2048) { $statusCall.Text.Substring($statusCall.Text.Length - 2048) } else { $statusCall.Text })
            }
            disk_bytes = $diskBytes
        })
    } else {
        $peersCall = Invoke-Captured { valdr-cli peers --node $Node }
        $miningCall = Invoke-Captured { valdr-cli mining info --node $Node }
        if ($peersCall.ExitCode -ne 0 -or $miningCall.ExitCode -ne 0) {
            $failed = if ($peersCall.ExitCode -ne 0) { $peersCall } else { $miningCall }
            $component = if ($peersCall.ExitCode -ne 0) { "valdr-cli-peers" } else { "valdr-cli-mining-info" }
            Append-JsonLine $snapshotsPath ([ordered]@{
                observed_at = $observedAt
                observed_epoch = $observedEpoch
                observation_error = [ordered]@{
                    component = $component
                    exit_code = $failed.ExitCode
                    message = $failed.Text
                }
                disk_bytes = $diskBytes
            })
        } else {
            $status = $statusCall.Text | ConvertFrom-Json
            $peers = $peersCall.Text | ConvertFrom-Json
            $mining = $miningCall.Text | ConvertFrom-Json

            if ($status.network -ne "testnet2" -or $status.chain_id -ne "valdr-testnet-2") { throw "unexpected network identity" }
            if ([string]::IsNullOrWhiteSpace([string]$status.tip_hash)) { throw "missing tip hash" }
            if ([string]::IsNullOrWhiteSpace([string]$status.chainwork)) { throw "missing chainwork" }
            if ([string]::IsNullOrWhiteSpace([string]$status.target)) { throw "missing target" }
            if ([int64]$mining.height -ne [int64]$status.height) { throw "mining/status height mismatch" }
            if ([string]$mining.current_target -ne [string]$status.target) { throw "mining/status target mismatch" }

            $height = [int64]$status.height
            $work = [System.Numerics.BigInteger]::Parse("0$($status.chainwork)", [System.Globalization.NumberStyles]::AllowHexSpecifier)
            if ($null -ne $previousHeight -and $height -lt $previousHeight) { throw "height regressed" }
            if ($null -ne $previousWork -and $work -lt $previousWork) { throw "chainwork regressed" }

            Append-JsonLine $snapshotsPath ([ordered]@{
                observed_at = $observedAt
                observed_epoch = $observedEpoch
                status = $status
                peers = $peers
                mining = $mining
                disk_bytes = $diskBytes
            })
            $previousHeight = $height
            $previousWork = $work
        }
    }

    if ($DurationSeconds -eq 0) { break }
    if ([DateTimeOffset]::UtcNow -ge $deadline) { break }
    Start-Sleep -Seconds $IntervalSeconds
} while ($true)

$endedAt = Get-UtcIso
$records = @()
Get-Content -LiteralPath $snapshotsPath | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | ForEach-Object {
    $records += ($_ | ConvertFrom-Json)
}
if ($records.Count -eq 0) { throw "no Stage14A snapshots were recorded" }
$success = @($records | Where-Object { $null -ne $_.status })
if ($success.Count -eq 0) { throw "no successful Stage14A status snapshots were recorded" }
$errors = @($records | Where-Object { $null -ne $_.observation_error })
$first = $success[0].status
$last = $success[-1].status

$summary = [ordered]@{
    schema = "valdr-stage14a-session-summary-v1"
    session_id = $SessionId
    session_group_id = $SessionGroup
    source_commit = $SourceCommit.ToLowerInvariant()
    started_at = $startedAt
    ended_at = $endedAt
    snapshot_count = $records.Count
    successful_snapshot_count = $success.Count
    observation_error_count = $errors.Count
    network = $last.network
    chain_id = $last.chain_id
    start_height = $first.height
    end_height = $last.height
    start_tip_hash = $first.tip_hash
    end_tip_hash = $last.tip_hash
    start_chainwork = $first.chainwork
    end_chainwork = $last.chainwork
    result = "local_observation_complete"
    stage14a_pass = $false
    note = "Stage 14A requires consolidated independent multi-machine session evidence."
}
$summary | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $summaryPath -Encoding utf8

$names = @("manifest.json", "snapshots.jsonl", "summary.json", "events.jsonl")
$lines = foreach ($name in $names) {
    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $sessionDir $name)).Hash.ToLowerInvariant()
    "$hash  $name"
}
$lines | Set-Content -LiteralPath $sumsPath -Encoding ascii

Write-Output "Stage14A local session evidence complete: $sessionDir"
Write-Output "This is not a Stage14A PASS; independent multi-machine evidence is still required."
