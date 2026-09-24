#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
node_pid=""

cleanup() {
  if [[ -n "$node_pid" ]]; then
    kill "$node_pid" >/dev/null 2>&1 || true
    wait "$node_pid" >/dev/null 2>&1 || true
  fi
  rm -rf "$tmp"
}
trap cleanup EXIT

cd "$root"
go build -o "$tmp/valdrd" ./cmd/valdrd

mkfifo "$tmp/managed.stdin"

"$tmp/valdrd" start \
  --network testnet \
  --data "$tmp/data" \
  --node-id desktop-outbound-smoke \
  --outbound-only \
  --managed-stdin-shutdown \
  --p2p-port 29333 \
  --rpc-host 127.0.0.1 \
  --rpc-port 29332 \
  <"$tmp/managed.stdin" >"$tmp/node.out" 2>"$tmp/node.err" &
node_pid=$!

# Keep exactly one writer in the parent. Closing fd 3 below must be the
# event that produces EOF for valdrd; the child must not inherit a writer.
exec 3>"$tmp/managed.stdin"

for _ in $(seq 1 40); do
  if "$tmp/valdrd" status --node http://127.0.0.1:29332 >"$tmp/status.json" 2>/dev/null; then
    break
  fi
  sleep 0.25
done

python3 - "$tmp/status.json" <<'PY'
import json, sys
with open(sys.argv[1]) as f:
    status=json.load(f)
assert status["network"] == "testnet", status
assert status["chain_id"] == "valdr-testnet-1", status
assert status["height"] == 0, status
PY

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

exec 3>&-
wait "$node_pid"
node_pid=""

grep -q "stopping on managed stdin close" "$tmp/node.err"

"$tmp/valdrd" verify-db   --data "$tmp/data"   --network testnet >"$tmp/verify.json"

grep -q '"valid": true' "$tmp/verify.json"

echo "VALDR Desktop outbound-only node smoke: PASS"
