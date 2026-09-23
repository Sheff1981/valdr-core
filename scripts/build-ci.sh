#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UPSTREAM_COMMIT="9be056a8a72b624dae9623b2f7bded92c2a21c91"
SRC="$ROOT/work/bitcoin-core"
rm -rf "$SRC" "$ROOT/release/bin"
mkdir -p "$ROOT/work" "$ROOT/release/bin"

git init "$SRC"
git -C "$SRC" remote add origin https://github.com/bitcoin/bitcoin.git
git -C "$SRC" fetch --depth 1 origin "$UPSTREAM_COMMIT"
git -C "$SRC" checkout --detach FETCH_HEAD
test "$(git -C "$SRC" rev-parse HEAD)" = "$UPSTREAM_COMMIT"

python3 "$ROOT/tools/apply_valdr_testnet.py" "$SRC"

cmake -S "$SRC" -B "$SRC/build"   -DBUILD_GUI=OFF   -DBUILD_TESTS=OFF   -DBUILD_BENCH=OFF   -DENABLE_IPC=OFF   -DWITH_ZMQ=ON   -DCMAKE_BUILD_TYPE=Release

cmake --build "$SRC/build" --target bitcoind bitcoin-cli -j"${JOBS:-2}"

cp "$SRC/build/bin/bitcoind" "$ROOT/release/bin/valdrd"
cp "$SRC/build/bin/bitcoin-cli" "$ROOT/release/bin/valdr-cli"
chmod +x "$ROOT/release/bin/valdrd" "$ROOT/release/bin/valdr-cli"

"$ROOT/release/bin/valdrd" --version
sha256sum "$ROOT/release/bin/valdrd" "$ROOT/release/bin/valdr-cli" | tee "$ROOT/release/bin/SHA256SUMS"
