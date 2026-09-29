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

echo.
echo ============================================
echo BENCHMARK FINISHED
echo ============================================
echo Result file:
echo %~dp0valdr-randomx-real-cpu.txt
echo.
echo Send valdr-randomx-real-cpu.txt to the VALDR project chat.
echo.
pause
