#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
target="$repo_root/scripts/stage14a-session-validate.sh"
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

PATH="$tmp/bin:$PATH" bash "$target"   --session-id ci-smoke   --operator ci   --source-commit 0123456789abcdef0123456789abcdef01234567   --bootstrap-route ci-fake-bootstrap   --node http://127.0.0.1:17332   --data "$tmp/data"   --duration-seconds 0   --interval-seconds 1   --output-dir "$tmp/out"

test -s "$tmp/out/ci-smoke/manifest.json"
test -s "$tmp/out/ci-smoke/snapshots.jsonl"
test -s "$tmp/out/ci-smoke/summary.json"
test -s "$tmp/out/ci-smoke/SHA256SUMS"

python3 - "$tmp/out/ci-smoke/manifest.json" "$tmp/out/ci-smoke/summary.json" <<'PY'
import json, sys
manifest = json.load(open(sys.argv[1], encoding="utf-8"))
summary = json.load(open(sys.argv[2], encoding="utf-8"))
assert manifest["network"] == "testnet2"
assert manifest["chain_id"] == "valdr-testnet-2"
assert manifest["source_commit"] == "0123456789abcdef0123456789abcdef01234567"
assert summary["snapshot_count"] == 1
assert summary["start_height"] == 42
assert summary["end_height"] == 42
assert summary["result"] == "local_observation_complete"
assert summary["stage14a_pass"] is False
PY

echo "Stage14A session evidence tooling smoke passed"
