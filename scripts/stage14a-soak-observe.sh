#!/usr/bin/env bash
set -euo pipefail

node="http://127.0.0.1:17332"
data_dir="./data"
duration_seconds=86400
interval_seconds=60
output="stage14a-soak.jsonl"

usage() {
  cat <<'EOF'
usage: stage14a-soak-observe.sh [options]

Options:
  --node URL              localhost VALDR RPC endpoint (default http://127.0.0.1:17332)
  --data PATH             local node data directory (default ./data)
  --duration-seconds N    observation duration; 0 = one snapshot (default 86400)
  --interval-seconds N    delay between snapshots (default 60)
  --output PATH           JSONL evidence file (default stage14a-soak.jsonl)
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --node) node="$2"; shift 2 ;;
    --data) data_dir="$2"; shift 2 ;;
    --duration-seconds) duration_seconds="$2"; shift 2 ;;
    --interval-seconds) interval_seconds="$2"; shift 2 ;;
    --output) output="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

[[ "$duration_seconds" =~ ^[0-9]+$ ]] || { echo "duration must be >=0 integer" >&2; exit 2; }
[[ "$interval_seconds" =~ ^[1-9][0-9]*$ ]] || { echo "interval must be positive integer" >&2; exit 2; }

command -v valdrd >/dev/null || { echo "valdrd not found in PATH" >&2; exit 1; }
command -v valdr-cli >/dev/null || { echo "valdr-cli not found in PATH" >&2; exit 1; }
command -v python3 >/dev/null || { echo "python3 not found in PATH" >&2; exit 1; }

mkdir -p "$(dirname "$output")"
touch "$output"
chmod 600 "$output" 2>/dev/null || true

start_epoch=$(date +%s)
deadline=$((start_epoch + duration_seconds))
previous_height=""
previous_chainwork=""

snapshot() {
  local status peers mining disk_bytes now_epoch now_iso
  now_epoch=$(date +%s)
  now_iso=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  status=$(valdrd status --node "$node")
  peers=$(valdr-cli peers --node "$node")
  mining=$(valdr-cli mining info --node "$node")
  disk_bytes=$(du -sk "$data_dir" 2>/dev/null | awk '{print $1 * 1024}' || printf '0')

  python3 - "$now_iso" "$now_epoch" "$status" "$peers" "$mining" "$disk_bytes" "$previous_height" "$previous_chainwork" <<'PY' >>"$output"
import json, sys
ts, epoch, status_raw, peers_raw, mining_raw, disk_raw, prev_h, prev_work = sys.argv[1:]
status=json.loads(status_raw)
peers=json.loads(peers_raw)
mining=json.loads(mining_raw)

assert status["network"] == "testnet2", status
assert status["chain_id"] == "valdr-testnet-2", status
assert status["tip_hash"], status
assert status["chainwork"], status
assert status["target"], status
assert mining["height"] == status["height"], (mining, status)
assert mining["current_target"] == status["target"], (mining, status)

height=int(status["height"])
work=int(status["chainwork"], 16)
if prev_h:
    assert height >= int(prev_h), (prev_h, height)
if prev_work:
    assert work >= int(prev_work, 16), (prev_work, status["chainwork"])

record={
    "observed_at": ts,
    "observed_epoch": int(epoch),
    "status": status,
    "peers": peers,
    "mining": mining,
    "disk_bytes": int(float(disk_raw)),
}
print(json.dumps(record, sort_keys=True, separators=(",", ":")))
PY

  previous_height=$(python3 - "$status" <<'PY'
import json, sys
print(json.loads(sys.argv[1])["height"])
PY
)
  previous_chainwork=$(python3 - "$status" <<'PY'
import json, sys
print(json.loads(sys.argv[1])["chainwork"])
PY
)
}

while :; do
  snapshot
  if (( duration_seconds == 0 )); then
    break
  fi
  now=$(date +%s)
  if (( now >= deadline )); then
    break
  fi
  sleep "$interval_seconds"
done

echo "Stage14A observer complete: $output"
