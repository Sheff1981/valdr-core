@echo off
setlocal
cd /d "%~dp0"
title VALDR RandomX CPU Benchmark

echo ============================================
echo VALDR Mainnet M5 - Real CPU Benchmark
echo ============================================
echo.
echo This test does NOT mine coins and does NOT enable Mainnet.
echo It measures this Windows PC for M5 calibration evidence.
echo.
echo Close heavy programs for a more representative result.
echo The benchmark may use significant CPU and memory.
echo.
pause

powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0benchmark-real-cpu.ps1"
if errorlevel 1 (
  echo.
  echo BENCHMARK FAILED.
  echo Take a photo/screenshot of this window and send it to the VALDR project chat.
  pause
  exit /b 1
)

if exist "%~dp0valdr-m5-ingest.exe" (
  "%~dp0valdr-m5-ingest.exe" -input "%~dp0valdr-randomx-real-cpu.txt" -output "%~dp0valdr-m5-machine.json"
  if errorlevel 1 (
    echo.
    echo WARNING: benchmark finished, but JSON normalization failed.
    echo The TXT result is still valid and should be sent to the VALDR chat.
  )
)

echo.
echo ============================================
echo BENCHMARK FINISHED
echo ============================================
echo Result file:
echo %~dp0valdr-randomx-real-cpu.txt
echo.
echo Send these files to the VALDR project chat:
echo   valdr-randomx-real-cpu.txt
echo   valdr-m5-machine.json  ^(if created^)
echo.
pause
