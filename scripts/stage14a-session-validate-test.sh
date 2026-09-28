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
test -s "$tmp/out/ci-g1-m1/SHA256SUMS"

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
assert all(item["common_tip_observed"] for item in result["distributed_sessions"])
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

echo "Stage14A grouped multi-machine evidence tooling smoke passed"
