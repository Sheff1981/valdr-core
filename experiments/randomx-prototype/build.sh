#!/usr/bin/env sh
set -eu

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
DEPS="$ROOT/.deps"
RX="$DEPS/RandomX"
BUILD="$ROOT/build"

mkdir -p "$DEPS"
if [ ! -d "$RX/.git" ]; then
  git clone --depth 1 --branch v1.2.3 https://github.com/tevador/RandomX.git "$RX"
fi

cmake -S "$RX" -B "$RX/build" -DCMAKE_BUILD_TYPE=Release
cmake --build "$RX/build" --config Release --parallel

"$RX/build/randomx-tests"

cmake -S "$ROOT" -B "$BUILD" -DCMAKE_BUILD_TYPE=Release -DRANDOMX_DIR="$RX"
cmake --build "$BUILD" --config Release --parallel

"$BUILD/valdr-randomx-vector"

# Verification-mode benchmark avoids allocating the ~2 GiB full mining dataset in CI/local smoke.
"$RX/build/randomx-benchmark" --verify --auto --nonces 2000
