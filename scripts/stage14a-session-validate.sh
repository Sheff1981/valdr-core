#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
observer="$script_dir/stage14a-soak-observe.sh"

node="http://127.0.0.1:17332"
data_dir="./data"
duration_seconds=10800
interval_seconds=60
output_dir="./stage14a-evidence"
session_id=""
operator_id=""
machine_id=""
source_commit=""
bootstrap_route=""

usage() {
  cat <<'EOF'
usage: stage14a-session-validate.sh [options]

Records one Stage 14A local Testnet2 validation session. This tool does not
declare Stage 14A passed; final acceptance still requires the independent
multi-machine evidence defined by the active Master-TZ.

Options:
  --session-id ID         session identifier; default UTC timestamp
  --operator ID           operator label recorded in evidence
  --machine-id ID         stable non-secret machine/client label recorded in evidence
  --source-commit SHA     tested source commit; default current git HEAD
  --bootstrap-route TEXT  bootstrap route label/address used for this session
  --node URL              local VALDR RPC endpoint (default http://127.0.0.1:17332)
  --data PATH             local node data directory (default ./data)
  --duration-seconds N    observation duration (default 10800 = 3 hours; 0 for one snapshot)
  --interval-seconds N    delay between snapshots (default 60)
  --output-dir PATH       evidence directory (default ./stage14a-evidence)
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --session-id) session_id="$2"; shift 2 ;;
    --operator) operator_id="$2"; shift 2 ;;
    --machine-id) machine_id="$2"; shift 2 ;;
    --source-commit) source_commit="$2"; shift 2 ;;
    --bootstrap-route) bootstrap_route="$2"; shift 2 ;;
    --node) node="$2"; shift 2 ;;
    --data) data_dir="$2"; shift 2 ;;
    --duration-seconds) duration_seconds="$2"; shift 2 ;;
    --interval-seconds) interval_seconds="$2"; shift 2 ;;
    --output-dir) output_dir="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

[[ "$duration_seconds" =~ ^[0-9]+$ ]] || { echo "duration must be a non-negative integer" >&2; exit 2; }
[[ "$interval_seconds" =~ ^[1-9][0-9]*$ ]] || { echo "interval must be a positive integer" >&2; exit 2; }

if [[ -z "$session_id" ]]; then
  session_id="$(date -u +%Y%m%dT%H%M%SZ)"
fi
[[ "$session_id" =~ ^[A-Za-z0-9._-]+$ ]] || { echo "session-id may contain only A-Z a-z 0-9 . _ -" >&2; exit 2; }

command -v valdrd >/dev/null || { echo "valdrd not found in PATH" >&2; exit 1; }
command -v valdr-cli >/dev/null || { echo "valdr-cli not found in PATH" >&2; exit 1; }
command -v python3 >/dev/null || { echo "python3 not found in PATH" >&2; exit 1; }
[[ -f "$observer" ]] || { echo "observer script not found: $observer" >&2; exit 1; }

if [[ -z "$source_commit" ]]; then
  source_commit="$(git -C "$script_dir/.." rev-parse HEAD 2>/dev/null || true)"
fi
[[ -n "$source_commit" ]] || { echo "unable to determine source commit; pass --source-commit" >&2; exit 2; }

session_dir="$output_dir/$session_id"
mkdir -p "$session_dir"
chmod 700 "$session_dir" 2>/dev/null || true

snapshots="$session_dir/snapshots.jsonl"
manifest="$session_dir/manifest.json"
summary="$session_dir/summary.json"
checksums="$session_dir/SHA256SUMS"

started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
valdrd_version="$(valdrd version | head -n 1)"

python3 - "$manifest" "$session_id" "$operator_id" "$machine_id" "$source_commit" "$bootstrap_route" "$node" "$duration_seconds" "$interval_seconds" "$started_at" "$valdrd_version" <<'PY'
import json, sys
(path, session_id, operator_id, machine_id, source_commit, bootstrap_route, node,
 duration, interval, started_at, valdrd_version) = sys.argv[1:]
obj = {
    "schema": "valdr-stage14a-session-v1",
    "session_id": session_id,
    "operator": operator_id or None,
    "machine_id": machine_id or None,
    "source_commit": source_commit,
    "bootstrap_route": bootstrap_route or None,
    "rpc_endpoint": node,
    "network": "testnet2",
    "chain_id": "valdr-testnet-2",
    "planned_duration_seconds": int(duration),
    "interval_seconds": int(interval),
    "started_at": started_at,
    "valdrd_version": valdrd_version,
    "acceptance_scope": "local-session-evidence-only"
}
with open(path, "w", encoding="utf-8") as f:
    json.dump(obj, f, indent=2, sort_keys=True)
    f.write("\n")
PY
chmod 600 "$manifest" 2>/dev/null || true

bash "$observer"   --node "$node"   --data "$data_dir"   --duration-seconds "$duration_seconds"   --interval-seconds "$interval_seconds"   --output "$snapshots"

ended_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

python3 - "$snapshots" "$summary" "$session_id" "$source_commit" "$started_at" "$ended_at" <<'PY'
import json, sys
snapshots_path, summary_path, session_id, source_commit, started_at, ended_at = sys.argv[1:]
records = []
with open(snapshots_path, "r", encoding="utf-8") as f:
    for line in f:
        if line.strip():
            records.append(json.loads(line))
if not records:
    raise SystemExit("no Stage14A snapshots were recorded")
first = records[0]["status"]
last = records[-1]["status"]
summary = {
    "schema": "valdr-stage14a-session-summary-v1",
    "session_id": session_id,
    "source_commit": source_commit,
    "started_at": started_at,
    "ended_at": ended_at,
    "snapshot_count": len(records),
    "network": last["network"],
    "chain_id": last["chain_id"],
    "start_height": first["height"],
    "end_height": last["height"],
    "start_tip_hash": first["tip_hash"],
    "end_tip_hash": last["tip_hash"],
    "start_chainwork": first["chainwork"],
    "end_chainwork": last["chainwork"],
    "result": "local_observation_complete",
    "stage14a_pass": False,
    "note": "Stage 14A requires consolidated independent multi-machine session evidence."
}
with open(summary_path, "w", encoding="utf-8") as f:
    json.dump(summary, f, indent=2, sort_keys=True)
    f.write("\n")
PY
chmod 600 "$snapshots" "$summary" 2>/dev/null || true

if command -v sha256sum >/dev/null; then
  (
    cd "$session_dir"
    sha256sum manifest.json snapshots.jsonl summary.json > SHA256SUMS
  )
elif command -v shasum >/dev/null; then
  (
    cd "$session_dir"
    shasum -a 256 manifest.json snapshots.jsonl summary.json > SHA256SUMS
  )
else
  echo "sha256sum or shasum is required" >&2
  exit 1
fi
chmod 600 "$checksums" 2>/dev/null || true

echo "Stage14A local session evidence complete: $session_dir"
echo "This is not a Stage14A PASS; independent multi-machine evidence is still required."
