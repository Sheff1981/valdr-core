#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
desktop_pid=""
seed_pid=""

cleanup() {
  if [[ -n "$desktop_pid" ]]; then
    kill "$desktop_pid" >/dev/null 2>&1 || true
    wait "$desktop_pid" >/dev/null 2>&1 || true
  fi
  if [[ -n "$seed_pid" ]]; then
    kill "$seed_pid" >/dev/null 2>&1 || true
    wait "$seed_pid" >/dev/null 2>&1 || true
  fi
  rm -rf "$tmp"
}
trap cleanup EXIT

cd "$root"
go build -o "$tmp/valdrd" ./cmd/valdrd
go build -o "$tmp/valdr-miner" ./cmd/valdr-miner

# Start a real listening Testnet peer with one mined block. The Desktop node
# must discover it only through an outbound seed connection and synchronize
# the block without opening its own inbound P2P listener.
"$tmp/valdrd" start \
  --network testnet2 \
  --data "$tmp/seed-data" \
  --node-id desktop-outbound-seed \
  --p2p-host 127.0.0.1 \
  --p2p-port 29433 \
  --rpc-host 127.0.0.1 \
  --rpc-port 29432 \
  >"$tmp/seed.out" 2>"$tmp/seed.err" &
seed_pid=$!

seed_ready=0
for _ in $(seq 1 60); do
  if "$tmp/valdrd" status --node http://127.0.0.1:29432 >"$tmp/seed-status.json" 2>/dev/null; then
    seed_ready=1
    break
  fi
  if ! kill -0 "$seed_pid" >/dev/null 2>&1; then
    cat "$tmp/seed.err" >&2 || true
    exit 1
  fi
  sleep 0.25
done
if [[ "$seed_ready" -ne 1 ]]; then
  cat "$tmp/seed.err" >&2 || true
  exit 1
fi

"$tmp/valdr-miner" start \
  --node http://127.0.0.1:29432 \
  --reward-address VDR1NGF6UY64ISRUIZR76FBJV2QQQQW7E63LY2C6SUA \
  --blocks 1 \
  --interval 0s \
  --pid-file "$tmp/seed-miner.pid" \
  >"$tmp/seed-miner.json"

"$tmp/valdrd" status --node http://127.0.0.1:29432 >"$tmp/seed-after-mine.json"
python3 - "$tmp/seed-after-mine.json" <<'PY'
import json, sys
with open(sys.argv[1]) as f:
    status=json.load(f)
assert status["network"] == "testnet2", status
assert status["chain_id"] == "valdr-testnet-2", status
assert status["height"] >= 1, status
assert status["tip_hash"], status
PY

mkfifo "$tmp/managed.stdin"

"$tmp/valdrd" start \
  --network testnet2 \
  --data "$tmp/desktop-data" \
  --node-id desktop-outbound-smoke \
  --outbound-only \
  --managed-stdin-shutdown \
  --p2p-port 29333 \
  --rpc-host 127.0.0.1 \
  --rpc-port 29332 \
  --seed 127.0.0.1:29433 \
  <"$tmp/managed.stdin" >"$tmp/desktop.out" 2>"$tmp/desktop.err" &
desktop_pid=$!

# Keep exactly one writer in the parent. Closing fd 3 below must be the
# event that produces EOF for valdrd; the child must not inherit a writer.
exec 3>"$tmp/managed.stdin"

desktop_ready=0
for _ in $(seq 1 60); do
  if "$tmp/valdrd" status --node http://127.0.0.1:29332 >"$tmp/desktop-status.json" 2>/dev/null; then
    desktop_ready=1
    break
  fi
  if ! kill -0 "$desktop_pid" >/dev/null 2>&1; then
    cat "$tmp/desktop.err" >&2 || true
    exit 1
  fi
  sleep 0.25
done
if [[ "$desktop_ready" -ne 1 ]]; then
  cat "$tmp/desktop.err" >&2 || true
  exit 1
fi

# Verify the Desktop node is outbound-only at the socket boundary.
python3 - <<'PY'
import socket
sock=socket.socket()
sock.settimeout(0.5)
try:
    result=sock.connect_ex(("127.0.0.1", 29333))
finally:
    sock.close()
assert result != 0, "outbound-only Desktop node unexpectedly opened inbound P2P port"
PY

# Acceptance gate: a real Testnet block must synchronize from the listening
# seed to the outbound-only Desktop node, with the same active tip.
synced=0
for _ in $(seq 1 80); do
  "$tmp/valdrd" status --node http://127.0.0.1:29332 >"$tmp/desktop-status.json"
  if python3 - "$tmp/desktop-status.json" "$tmp/seed-after-mine.json" <<'PY'
import json, sys
with open(sys.argv[1]) as f:
    desktop=json.load(f)
with open(sys.argv[2]) as f:
    seed=json.load(f)
ok = (
    desktop["network"] == "testnet2"
    and desktop["chain_id"] == "valdr-testnet-2"
    and desktop["height"] == seed["height"]
    and desktop["tip_hash"] == seed["tip_hash"]
    and desktop["height"] >= 1
    and desktop["peer_count"] >= 1
)
raise SystemExit(0 if ok else 1)
PY
  then
    synced=1
    break
  fi
  sleep 0.25
done
if [[ "$synced" -ne 1 ]]; then
  echo "Desktop outbound-only Testnet synchronization did not converge" >&2
  cat "$tmp/desktop-status.json" >&2 || true
  cat "$tmp/desktop.err" >&2 || true
  exit 1
fi

exec 3>&-
wait "$desktop_pid"
desktop_pid=""

grep -q "stopping on managed stdin close" "$tmp/desktop.err"

"$tmp/valdrd" verify-db \
  --data "$tmp/desktop-data" \
  --network testnet2 >"$tmp/verify.json"

grep -q '"valid": true' "$tmp/verify.json"
grep -q '"height": 1' "$tmp/verify.json"

echo "VALDR Desktop outbound-only Testnet sync smoke: PASS"
