#!/usr/bin/env sh
set -eu

BENCH="${1:-./randomx-benchmark}"
OUT="${2:-valdr-randomx-real-cpu.txt}"

if [ ! -x "$BENCH" ]; then
  echo "benchmark executable not found: $BENCH" >&2
  exit 2
fi

{
  echo "VALDR_RANDOMX_REAL_CPU_V1"
  echo "date_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "os=$(uname -a)"
  if command -v sysctl >/dev/null 2>&1; then
    sysctl -n machdep.cpu.brand_string 2>/dev/null | sed 's/^/cpu=/' || true
    sysctl -n hw.memsize 2>/dev/null | sed 's/^/memory_bytes=/' || true
    LOGICAL=$(sysctl -n hw.logicalcpu 2>/dev/null || echo 1)
  else
    grep -m1 'model name' /proc/cpuinfo 2>/dev/null | sed 's/^[^:]*: */cpu=/' || true
    grep MemTotal /proc/meminfo 2>/dev/null | awk '{print "memory_kib="$2}' || true
    LOGICAL=$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 1)
  fi
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
