param(
  [string]$Benchmark = ".\\randomx-benchmark.exe",
  [string]$Output = "valdr-randomx-real-cpu.txt"
)

$ErrorActionPreference = "Stop"
$PSNativeCommandUseErrorActionPreference = $true

if (-not (Test-Path $Benchmark)) {
  throw "benchmark executable not found: $Benchmark"
}

$cpu = (Get-CimInstance Win32_Processor | Select-Object -First 1)
$mem = (Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory
$logical = [Environment]::ProcessorCount

"VALDR_RANDOMX_REAL_CPU_V1" | Tee-Object -FilePath $Output
"date_utc=$([DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ'))" | Tee-Object -FilePath $Output -Append
"os=$([Environment]::OSVersion.VersionString)" | Tee-Object -FilePath $Output -Append
"cpu=$($cpu.Name)" | Tee-Object -FilePath $Output -Append
"memory_bytes=$mem" | Tee-Object -FilePath $Output -Append
"logical_cpus=$logical" | Tee-Object -FilePath $Output -Append
"" | Tee-Object -FilePath $Output -Append

$threads = @(1,2,4,$logical) | Sort-Object -Unique
foreach ($t in $threads) {
  if ($t -gt $logical) { continue }
  "=== threads=$t ===" | Tee-Object -FilePath $Output -Append
  & $Benchmark --mine --auto --threads $t --nonces 1000 2>&1 |
    Tee-Object -FilePath $Output -Append
  "" | Tee-Object -FilePath $Output -Append
}

Write-Host "Saved: $Output"
