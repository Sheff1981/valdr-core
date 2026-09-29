#!/usr/bin/env sh
set -eu

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
DEPS="$ROOT/.deps"
RX="$DEPS/RandomX"
BUILD="$ROOT/build"

mkdir -p "$DEPS"
if [ ! -d "$RX/.git" ]; then
  git clone --no-checkout https://github.com/tevador/RandomX.git "$RX"
  git -C "$RX" checkout 12f2c2ffe2108d6cf54c391fee33c8bc3646cdab
fi

cmake -S "$RX" -B "$RX/build" -DCMAKE_BUILD_TYPE=Release
cmake --build "$RX/build" --config Release --parallel

"$RX/build/randomx-tests"

cmake -S "$ROOT" -B "$BUILD" -DCMAKE_BUILD_TYPE=Release -DRANDOMX_DIR="$RX"
cmake --build "$BUILD" --config Release --parallel

"$BUILD/valdr-randomx-vector"

# Verification-mode benchmark avoids allocating the ~2 GiB full mining dataset in CI/local smoke.
echo "=== VALDR RandomX verification benchmark ==="
"$RX/build/randomx-benchmark" --verify --auto --nonces 2000

echo "=== VALDR RandomX full-memory mining benchmark (1 thread) ==="
"$RX/build/randomx-benchmark" --mine --auto --threads 1 --nonces 200
