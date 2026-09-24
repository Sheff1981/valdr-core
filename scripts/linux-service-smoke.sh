#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
cleanup() {
  if [[ -n "${explorer_pid:-}" ]]; then kill "$explorer_pid" >/dev/null 2>&1 || true; fi
  if [[ -n "${node_pid:-}" ]]; then kill "$node_pid" >/dev/null 2>&1 || true; fi
  wait "${explorer_pid:-}" >/dev/null 2>&1 || true
  wait "${node_pid:-}" >/dev/null 2>&1 || true
  rm -rf "$tmp"
}
trap cleanup EXIT

cd "$root"
mkdir -p "$tmp/bin" "$tmp/data"
go build -o "$tmp/bin/valdrd" ./cmd/valdrd
go build -o "$tmp/bin/valdr-explorer" ./cmd/valdr-explorer

if command -v systemd-analyze >/dev/null 2>&1; then
  user=$(id -un)
  group=$(id -gn)
  sed     -e "s#User=valdr#User=$user#"     -e "s#Group=valdr#Group=$group#"     -e "s#/usr/local/bin/valdrd#$tmp/bin/valdrd#g"     deploy/systemd/valdrd.service > "$tmp/valdrd.service"
  sed     -e "s#User=valdr#User=$user#"     -e "s#Group=valdr#Group=$group#"     -e "s#/usr/local/bin/valdr-explorer#$tmp/bin/valdr-explorer#g"     deploy/systemd/valdr-explorer.service > "$tmp/valdr-explorer.service"
  systemd-analyze verify "$tmp/valdrd.service" "$tmp/valdr-explorer.service"
fi

start_node() {
  "$tmp/bin/valdrd" start     --network testnet     --data "$tmp/data"     --node-id linux-smoke     --p2p-host 127.0.0.1     --p2p-port 27333     --rpc-host 127.0.0.1     --rpc-port 27332     >"$tmp/node.out" 2>"$tmp/node.err" &
  node_pid=$!
}

wait_rpc() {
  for _ in $(seq 1 40); do
    if "$tmp/bin/valdrd" status --node http://127.0.0.1:27332 >"$tmp/status.json" 2>/dev/null; then
      return 0
    fi
    sleep 0.25
  done
  cat "$tmp/node.err" >&2 || true
  return 1
}

start_node
wait_rpc
python3 - "$tmp/status.json" <<'PY'
import json, sys
with open(sys.argv[1]) as f:
    status=json.load(f)
assert status["chain_id"] == "valdr-testnet-1", status
assert status["network"] == "testnet", status
PY

"$tmp/bin/valdr-explorer"   --listen 127.0.0.1:28080   --node http://127.0.0.1:27332   --index-file "$tmp/data/explorer/index.json"   >"$tmp/explorer.out" 2>"$tmp/explorer.err" &
explorer_pid=$!

for _ in $(seq 1 40); do
  if curl --fail --silent http://127.0.0.1:28080/healthz >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done
curl --fail --silent http://127.0.0.1:28080/api/v1/status >/dev/null

kill "$explorer_pid"
wait "$explorer_pid" || true
unset explorer_pid
kill "$node_pid"
wait "$node_pid" || true
unset node_pid

"$tmp/bin/valdrd" verify-db --data "$tmp/data" --network testnet >"$tmp/verify.json"
grep -q '"valid": true' "$tmp/verify.json"

start_node
wait_rpc
kill "$node_pid"
wait "$node_pid" || true
unset node_pid

echo "VALDR Linux service smoke: PASS"
