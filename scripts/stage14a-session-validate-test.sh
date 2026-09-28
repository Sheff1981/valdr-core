#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
target="$repo_root/scripts/stage14a-session-validate.sh"
verifier="$repo_root/scripts/stage14a-evidence-verify.py"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

mkdir -p "$tmp/bin" "$tmp/data" "$tmp/out"

cat >"$tmp/bin/valdrd" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in
  version)
    echo "VALDR VDR valdrd 0.2.0-dev"
    ;;
  status)
    cat <<'JSON'
{"network":"testnet2","chain_id":"valdr-testnet-2","height":42,"tip_hash":"0000000000000000000000000000000000000000000000000000000000000042","chainwork":"0000000000000000000000000000000000000000000000000000000000001234","target":"0000031b5d43afe99ee43470e1337c3642e9d9254926038fdf6d1a2e57aaa21f"}
JSON
    ;;
  *)
    echo "unsupported fake valdrd command" >&2
    exit 2
    ;;
esac
EOF

cat >"$tmp/bin/valdr-cli" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
case "${1:-} ${2:-}" in
  "mining info")
    cat <<'JSON'
{"height":42,"current_target":"0000031b5d43afe99ee43470e1337c3642e9d9254926038fdf6d1a2e57aaa21f","hashrate":0}
JSON
    ;;
  *)
    if [[ "${1:-}" == "peers" ]]; then
      echo '{"peers":[]}'
    else
      echo "unsupported fake valdr-cli command" >&2
      exit 2
    fi
    ;;
esac
EOF

chmod +x "$tmp/bin/valdrd" "$tmp/bin/valdr-cli"

commit=0123456789abcdef0123456789abcdef01234567

# Three distributed validation sessions, each with three independent machine labels.
for group in 1 2 3; do
  for machine in 1 2 3; do
    evidence_id="ci-g${group}-m${machine}"
    PATH="$tmp/bin:$PATH" bash "$target"       --session-id "$evidence_id"       --session-group "ci-group-$group"       --operator ci       --machine-id "ci-machine-$machine"       --source-commit "$commit"       --bootstrap-route ci-fake-bootstrap       --node http://127.0.0.1:17332       --data "$tmp/data"       --duration-seconds 0       --interval-seconds 1       --output-dir "$tmp/out"
  done
done

test -s "$tmp/out/ci-g1-m1/manifest.json"
test -s "$tmp/out/ci-g1-m1/snapshots.jsonl"
test -s "$tmp/out/ci-g1-m1/summary.json"
test -f "$tmp/out/ci-g1-m1/events.jsonl"
test -s "$tmp/out/ci-g1-m1/SHA256SUMS"

if PATH="$tmp/bin:$PATH" bash "$target"   --session-id ci-g1-m1   --session-group ci-group-1   --operator ci   --machine-id ci-machine-1   --source-commit "$commit"   --bootstrap-route ci-fake-bootstrap   --node http://127.0.0.1:17332   --data "$tmp/data"   --duration-seconds 0   --interval-seconds 1   --output-dir "$tmp/out" >/dev/null 2>&1; then
  echo "duplicate Stage14A session id unexpectedly overwrote evidence" >&2
  exit 1
fi

python3 - "$tmp/out/ci-g1-m1/manifest.json" "$tmp/out/ci-g1-m1/summary.json" <<'PY'
import json, sys
manifest = json.load(open(sys.argv[1], encoding="utf-8"))
summary = json.load(open(sys.argv[2], encoding="utf-8"))
assert manifest["network"] == "testnet2"
assert manifest["chain_id"] == "valdr-testnet-2"
assert manifest["source_commit"] == "0123456789abcdef0123456789abcdef01234567"
assert manifest["machine_id"] == "ci-machine-1"
assert manifest["session_group_id"] == "ci-group-1"
assert summary["session_group_id"] == "ci-group-1"
assert summary["snapshot_count"] == 1
assert summary["start_height"] == 42
assert summary["end_height"] == 42
assert summary["result"] == "local_observation_complete"
assert summary["stage14a_pass"] is False
PY

python3 "$repo_root/scripts/stage14a-event.py" "$tmp/out/ci-g1-m1" peer_exchange --result pass --note "CI synthetic operator event" >/dev/null
python3 "$repo_root/scripts/stage14a-event.py" "$tmp/out/ci-g1-m2" mining --result pass >/dev/null
python3 "$repo_root/scripts/stage14a-event.py" "$tmp/out/ci-g1-m3" transaction --result pass >/dev/null
python3 "$repo_root/scripts/stage14a-event.py" "$tmp/out/ci-g3-m1" mining --result pass >/dev/null
python3 "$repo_root/scripts/stage14a-event.py" "$tmp/out/ci-g2-m1" restart --result pass >/dev/null
python3 "$repo_root/scripts/stage14a-event.py" "$tmp/out/ci-g2-m2" db_verify --result pass >/dev/null
python3 "$repo_root/scripts/stage14a-event.py" "$tmp/out/ci-g2-m3" bootstrap_loss --result pass >/dev/null

python3 "$verifier" "$tmp/out" >"$tmp/consolidated.json"

python3 - "$tmp/consolidated.json" <<'PY'
import json, sys
result = json.load(open(sys.argv[1], encoding="utf-8"))
assert result["schema"] == "valdr-stage14a-consolidated-check-v2"
assert result["automated_evidence_ready"] is True
assert result["stage14a_pass"] is False
assert result["human_review_required"] is True
assert result["session_count"] == 3
assert result["evidence_count"] == 9
assert result["machine_count"] == 3
assert result["checks"]["at_least_three_distributed_sessions"] is True
assert result["checks"]["each_session_has_at_least_three_independent_machine_labels"] is True
assert result["checks"]["single_exact_source_commit"] is True
assert result["checks"]["bootstrap_route_recorded_each_session"] is True
assert result["checks"]["common_tip_observed_each_session"] is True
assert result["checks"]["same_final_tip_and_chainwork_each_session"] is True
assert all(len(item["end_heights"]) == 1 for item in result["distributed_sessions"])
assert result["checks"]["required_operator_scenarios_passed"] is True
assert result["checks"]["mining_passed_on_at_least_two_machine_labels"] is True
assert all(item["common_tip_observed"] for item in result["distributed_sessions"])
assert all(item["observed_duration_seconds"] >= 0 for item in result["evidence"])
coverage = set(result["operator_recorded_scenario_coverage"])
passed = set(result["operator_recorded_passed_scenarios"])
assert {"peer_exchange", "mining", "transaction", "restart", "db_verify", "bootstrap_loss"} <= coverage
assert {"peer_exchange", "mining", "transaction", "restart", "db_verify", "bootstrap_loss"} <= passed
assert len(result["mining_pass_machine_labels"]) >= 2
PY

cp -R "$tmp/out/ci-g1-m1" "$tmp/bad-hash-format"
python3 - "$tmp/bad-hash-format" <<'PY'
import hashlib, json, pathlib, sys
root=pathlib.Path(sys.argv[1])
snapshots=root/"snapshots.jsonl"
rows=[json.loads(x) for x in snapshots.read_text(encoding="utf-8").splitlines() if x.strip()]
rows[0]["status"]["tip_hash"]="ABCDEF"
snapshots.write_text("\n".join(json.dumps(x, sort_keys=True) for x in rows)+"\n", encoding="utf-8")
summary=root/"summary.json"
s=json.loads(summary.read_text(encoding="utf-8"))
s["start_tip_hash"]="ABCDEF"
s["end_tip_hash"]="ABCDEF"
summary.write_text(json.dumps(s, indent=2, sort_keys=True)+"\n", encoding="utf-8")
names=["manifest.json","snapshots.jsonl","summary.json","events.jsonl"]
(root/"SHA256SUMS").write_text(
    "\n".join(f"{hashlib.sha256((root/n).read_bytes()).hexdigest()}  {n}" for n in names)+"\n",
    encoding="utf-8",
)
PY
if python3 "$verifier" "$tmp/bad-hash-format" >/dev/null 2>&1; then
  echo "Stage14A evidence with malformed tip hash unexpectedly verified" >&2
  exit 1
fi

cp -R "$tmp/out" "$tmp/divergent-final-out"
python3 - "$tmp/divergent-final-out/ci-g2-m3" <<'PY'
import hashlib, json, pathlib, sys
root=pathlib.Path(sys.argv[1])
snapshots_path=root/"snapshots.jsonl"
summary_path=root/"summary.json"
rows=[json.loads(x) for x in snapshots_path.read_text(encoding="utf-8").splitlines() if x.strip()]
old=rows[0]
new=json.loads(json.dumps(old))
new["observed_epoch"]=int(old.get("observed_epoch", 0))+1
new["status"]["height"]=43
new["status"]["tip_hash"]="f"*64
new["status"]["chainwork"]="0"*60+"9999"
snapshots_path.write_text("\n".join(json.dumps(x, sort_keys=True) for x in (old, new))+"\n", encoding="utf-8")
summary=json.loads(summary_path.read_text(encoding="utf-8"))
summary["snapshot_count"]=2
summary["successful_snapshot_count"]=2
summary["observation_error_count"]=0
summary["end_height"]=43
summary["end_tip_hash"]=new["status"]["tip_hash"]
summary["end_chainwork"]=new["status"]["chainwork"]
summary_path.write_text(json.dumps(summary, indent=2, sort_keys=True)+"\n", encoding="utf-8")
names=["manifest.json","snapshots.jsonl","summary.json","events.jsonl"]
(root/"SHA256SUMS").write_text(
    "\n".join(f"{hashlib.sha256((root/n).read_bytes()).hexdigest()}  {n}" for n in names)+"\n",
    encoding="utf-8",
)
PY
if python3 "$verifier" "$tmp/divergent-final-out" >"$tmp/divergent-final.json"; then
  echo "Stage14A evidence with divergent final chain unexpectedly became ready" >&2
  exit 1
fi
python3 - "$tmp/divergent-final.json" <<'PY'
import json, sys
result=json.load(open(sys.argv[1], encoding="utf-8"))
assert result["automated_evidence_ready"] is False
assert result["checks"]["same_final_tip_and_chainwork_each_session"] is False
assert any(not x["end_converged"] for x in result["distributed_sessions"])
PY

cp -R "$tmp/out" "$tmp/one-miner-out"
python3 - "$tmp/one-miner-out/ci-g3-m1" <<'PY'
import hashlib, json, pathlib, sys
root=pathlib.Path(sys.argv[1])
events=root/"events.jsonl"
rows=[json.loads(x) for x in events.read_text(encoding="utf-8").splitlines() if x.strip()]
rows=[x for x in rows if x.get("event_type") != "mining"]
events.write_text("\n".join(json.dumps(x, sort_keys=True) for x in rows)+("\n" if rows else ""), encoding="utf-8")
names=["manifest.json","snapshots.jsonl","summary.json","events.jsonl"]
(root/"SHA256SUMS").write_text(
    "\n".join(f"{hashlib.sha256((root/n).read_bytes()).hexdigest()}  {n}" for n in names)+"\n",
    encoding="utf-8",
)
PY
if python3 "$verifier" "$tmp/one-miner-out" >"$tmp/one-miner.json"; then
  echo "Stage14A evidence with only one mining machine unexpectedly became ready" >&2
  exit 1
fi
python3 - "$tmp/one-miner.json" <<'PY'
import json, sys
result=json.load(open(sys.argv[1], encoding="utf-8"))
assert result["automated_evidence_ready"] is False
assert result["checks"]["mining_passed_on_at_least_two_machine_labels"] is False
PY

# Guard against the previous weak interpretation: one machine per distributed
# session is not enough even if three different machines exist overall.
mkdir -p "$tmp/weak"
for group in 1 2 3; do
  cp -R "$tmp/out/ci-g${group}-m${group}" "$tmp/weak/"
done
if python3 "$verifier" "$tmp/weak" >"$tmp/weak.json"; then
  echo "weak Stage14A evidence unexpectedly passed" >&2
  exit 1
fi
python3 - "$tmp/weak.json" <<'PY'
import json, sys
result = json.load(open(sys.argv[1], encoding="utf-8"))
assert result["automated_evidence_ready"] is False
assert result["checks"]["each_session_has_at_least_three_independent_machine_labels"] is False
assert result["stage14a_pass"] is False
PY

mkdir -p "$tmp/flaky-bin"
cat >"$tmp/flaky-bin/valdr-cli" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
case "${1:-} ${2:-}" in
  "mining info")
    cat <<'JSON'
{"height":43,"current_target":"0000031b5d43afe99ee43470e1337c3642e9d9254926038fdf6d1a2e57aaa21f","hashrate":0}
JSON
    ;;
  *)
    if [[ "${1:-}" == "peers" ]]; then
      echo '{"peers":[]}'
    else
      echo "unsupported fake valdr-cli command" >&2
      exit 2
    fi
    ;;
esac
EOF
cat >"$tmp/flaky-bin/valdrd" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in
  version)
    echo "VALDR VDR valdrd 0.2.0-dev"
    ;;
  status)
    counter="${VALDR_FLAKY_COUNTER:?}"
    if [[ ! -e "$counter" ]]; then
      printf '1\n' >"$counter"
      echo "temporary RPC unavailable during restart" >&2
      exit 7
    fi
    cat <<'JSON'
{"network":"testnet2","chain_id":"valdr-testnet-2","height":43,"tip_hash":"0000000000000000000000000000000000000000000000000000000000000043","chainwork":"0000000000000000000000000000000000000000000000000000000000001235","target":"0000031b5d43afe99ee43470e1337c3642e9d9254926038fdf6d1a2e57aaa21f"}
JSON
    ;;
  *)
    echo "unsupported fake valdrd command" >&2
    exit 2
    ;;
esac
EOF
chmod +x "$tmp/flaky-bin/valdrd" "$tmp/flaky-bin/valdr-cli"
VALDR_FLAKY_COUNTER="$tmp/flaky-counter" PATH="$tmp/flaky-bin:$PATH"   bash "$repo_root/scripts/stage14a-soak-observe.sh"     --node http://127.0.0.1:17332     --data "$tmp/data"     --duration-seconds 1     --interval-seconds 1     --output "$tmp/flaky.jsonl"
python3 - "$tmp/flaky.jsonl" <<'PY'
import json, sys
records=[json.loads(line) for line in open(sys.argv[1], encoding="utf-8") if line.strip()]
assert len(records) >= 2, records
errors=[r["observation_error"] for r in records if "observation_error" in r]
assert errors, records
assert any(e["component"] == "valdrd-status" and e["exit_code"] == 7 for e in errors), errors
assert any(r.get("status", {}).get("height") == 43 for r in records), records
PY

cp -R "$tmp/out/ci-g3-m3" "$tmp/tampered"
printf '\n' >>"$tmp/tampered/manifest.json"
if python3 "$verifier" "$tmp/tampered" >/dev/null 2>&1; then
  echo "tampered Stage14A evidence unexpectedly verified" >&2
  exit 1
fi

cp -R "$tmp/out/ci-g1-m1" "$tmp/reorg-height-drop"
python3 - "$tmp/reorg-height-drop" "$verifier" <<'PY'
import hashlib, importlib.util, json, pathlib, sys
root=pathlib.Path(sys.argv[1])
verifier_path=pathlib.Path(sys.argv[2])
snapshots_path=root/"snapshots.jsonl"
summary_path=root/"summary.json"
rows=[json.loads(x) for x in snapshots_path.read_text(encoding="utf-8").splitlines() if x.strip()]
first=rows[0]
second=json.loads(json.dumps(first))
first["status"]["height"]=42
first["status"]["tip_hash"]="0"*63+"a"
first["status"]["chainwork"]="0"*60+"1234"
second["observed_epoch"]=int(first.get("observed_epoch", 0))+1
second["status"]["height"]=41
second["status"]["tip_hash"]="0"*63+"b"
second["status"]["chainwork"]="0"*60+"1235"
snapshots_path.write_text("\n".join(json.dumps(x, sort_keys=True) for x in (first, second))+"\n", encoding="utf-8")
summary=json.loads(summary_path.read_text(encoding="utf-8"))
summary["snapshot_count"]=2
summary["successful_snapshot_count"]=2
summary["observation_error_count"]=0
summary["start_height"]=42
summary["end_height"]=41
summary["start_tip_hash"]=first["status"]["tip_hash"]
summary["end_tip_hash"]=second["status"]["tip_hash"]
summary["start_chainwork"]=first["status"]["chainwork"]
summary["end_chainwork"]=second["status"]["chainwork"]
summary_path.write_text(json.dumps(summary, indent=2, sort_keys=True)+"\n", encoding="utf-8")
names=["manifest.json","snapshots.jsonl","summary.json","events.jsonl"]
(root/"SHA256SUMS").write_text(
    "\n".join(f"{hashlib.sha256((root/n).read_bytes()).hexdigest()}  {n}" for n in names)+"\n",
    encoding="utf-8",
)
spec=importlib.util.spec_from_file_location("stage14a_verify", verifier_path)
module=importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
verified=module.verify_session(root)
assert verified["start_height"] == 42
assert verified["end_height"] == 41
PY

cp -R "$tmp/out/ci-g1-m1" "$tmp/bad-summary"
python3 - "$tmp/bad-summary" <<'PY'
import hashlib, json, pathlib, sys
root=pathlib.Path(sys.argv[1])
summary_path=root/"summary.json"
summary=json.loads(summary_path.read_text(encoding="utf-8"))
summary["end_tip_hash"]="f"*64
summary_path.write_text(json.dumps(summary, indent=2, sort_keys=True)+"\n", encoding="utf-8")
names=["manifest.json","snapshots.jsonl","summary.json","events.jsonl"]
(root/"SHA256SUMS").write_text(
    "\n".join(f"{hashlib.sha256((root/n).read_bytes()).hexdigest()}  {n}" for n in names)+"\n",
    encoding="utf-8",
)
PY
if python3 "$verifier" "$tmp/bad-summary" >/dev/null 2>&1; then
  echo "semantically inconsistent Stage14A summary unexpectedly verified" >&2
  exit 1
fi

cp -R "$tmp/out/ci-g1-m1" "$tmp/bad-event"
python3 - "$tmp/bad-event" <<'PY'
import hashlib, json, pathlib, sys
root=pathlib.Path(sys.argv[1])
events=root/"events.jsonl"
rows=[json.loads(x) for x in events.read_text().splitlines() if x.strip()]
rows[0]["event_type"]="made_up_event"
events.write_text("\n".join(json.dumps(x, sort_keys=True) for x in rows)+"\n")
names=["manifest.json","snapshots.jsonl","summary.json","events.jsonl"]
(root/"SHA256SUMS").write_text(
    "\n".join(f"{hashlib.sha256((root/n).read_bytes()).hexdigest()}  {n}" for n in names)+"\n"
)
PY
if python3 "$verifier" "$tmp/bad-event" >/dev/null 2>&1; then
  echo "semantically invalid Stage14A event unexpectedly verified" >&2
  exit 1
fi

cp -R "$tmp/out" "$tmp/failing-out"
python3 "$repo_root/scripts/stage14a-event.py"   "$tmp/failing-out/ci-g3-m1" crash_error --result fail --note "CI synthetic failure" >/dev/null
if python3 "$verifier" "$tmp/failing-out" >"$tmp/failing.json"; then
  echo "Stage14A evidence with operator-recorded failure unexpectedly became ready" >&2
  exit 1
fi
python3 - "$tmp/failing.json" <<'PY'
import json, sys
result=json.load(open(sys.argv[1], encoding="utf-8"))
assert result["automated_evidence_ready"] is False
assert result["checks"]["no_operator_recorded_failures"] is False
assert any(x["operator_recorded_failed_event_count"] > 0 for x in result["distributed_sessions"])
assert result["stage14a_pass"] is False
PY

echo "Stage14A grouped multi-machine evidence tooling smoke passed"
