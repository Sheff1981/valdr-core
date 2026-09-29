#!/usr/bin/env sh
set -eu

BENCH="${1:-./randomx-benchmark}"
OUT="${2:-valdr-randomx-real-cpu.txt}"

if [ ! -x "$BENCH" ]; then
  echo "benchmark executable not found: $BENCH" >&2
  exit 2
fi

OS_NAME=$(uname -s 2>/dev/null || echo unknown)

{
  echo "VALDR_RANDOMX_REAL_CPU_V1"
  echo "date_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "os=$(uname -a)"

  case "$OS_NAME" in
    Darwin)
      CPU=$(sysctl -n machdep.cpu.brand_string 2>/dev/null || echo unknown)
      MEM=$(sysctl -n hw.memsize 2>/dev/null || echo 0)
      LOGICAL=$(sysctl -n hw.logicalcpu 2>/dev/null || echo 1)
      echo "cpu=$CPU"
      echo "memory_bytes=$MEM"
      ;;
    Linux)
      CPU=$(grep -m1 'model name' /proc/cpuinfo 2>/dev/null | sed 's/^[^:]*: *//' || true)
      [ -n "$CPU" ] || CPU=$(lscpu 2>/dev/null | awk -F: '/Model name/ {sub(/^[ 	]+/, "", $2); print $2; exit}' || true)
      [ -n "$CPU" ] || CPU=unknown
      MEM=$(grep MemTotal /proc/meminfo 2>/dev/null | awk '{print $2}' || echo 0)
      LOGICAL=$(getconf _NPROCESSORS_ONLN 2>/dev/null || nproc 2>/dev/null || echo 1)
      echo "cpu=$CPU"
      echo "memory_kib=$MEM"
      ;;
    *)
      CPU=unknown
      LOGICAL=$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 1)
      echo "cpu=$CPU"
      ;;
  esac

  echo "logical_cpus=$LOGICAL"
  echo

  for T in 1 2 4 "$LOGICAL"; do
    [ "$T" -gt "$LOGICAL" ] && continue
    echo "=== threads=$T ==="
    "$BENCH" --mine --auto --threads "$T" --nonces 1000
    echo
  done
} | tee "$OUT"

echo "Saved: $OUT"
