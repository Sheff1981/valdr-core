$ErrorActionPreference = "Stop"
$PSNativeCommandUseErrorActionPreference = $true
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$Deps = Join-Path $Root ".deps"
$Rx = Join-Path $Deps "RandomX"
$Build = Join-Path $Root "build"

New-Item -ItemType Directory -Force -Path $Deps | Out-Null

if (-not (Test-Path (Join-Path $Rx ".git"))) {
  git clone --depth 1 --branch v1.2.3 https://github.com/tevador/RandomX.git $Rx
}

cmake -S $Rx -B (Join-Path $Rx "build") -DCMAKE_BUILD_TYPE=Release
cmake --build (Join-Path $Rx "build") --config Release --parallel

$RxTests = Join-Path $Rx "build/Release/randomx-tests.exe"
if (-not (Test-Path $RxTests)) { $RxTests = Join-Path $Rx "build/randomx-tests.exe" }
& $RxTests

cmake -S $Root -B $Build -DCMAKE_BUILD_TYPE=Release "-DRANDOMX_DIR=$Rx"
cmake --build $Build --config Release --parallel

$Vector = Join-Path $Build "Release/valdr-randomx-vector.exe"
if (-not (Test-Path $Vector)) { $Vector = Join-Path $Build "valdr-randomx-vector.exe" }
& $Vector

$Bench = Join-Path $Rx "build/Release/randomx-benchmark.exe"
if (-not (Test-Path $Bench)) { $Bench = Join-Path $Rx "build/randomx-benchmark.exe" }
Write-Host "=== VALDR RandomX verification benchmark ==="
& $Bench --verify --auto --nonces 2000

Write-Host "=== VALDR RandomX full-memory mining benchmark (1 thread) ==="
& $Bench --mine --auto --threads 1 --nonces 200
