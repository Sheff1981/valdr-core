#!/usr/bin/env bash
set -euo pipefail

compose=(docker compose -f deploy/docker-compose.testnet.yml)
cleanup() {
  "${compose[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

"${compose[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
"${compose[@]}" build
"${compose[@]}" up -d

wait_healthy() {
  local service="$1"
  local id health
  for _ in $(seq 1 60); do
    id=$("${compose[@]}" ps -q "$service")
    if [[ -n "$id" ]]; then
      health=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$id")
      if [[ "$health" == "healthy" || "$health" == "running" ]]; then
        return 0
      fi
    fi
    sleep 1
  done
  "${compose[@]}" ps
  "${compose[@]}" logs
  return 1
}

for service in node1 node2 node3 explorer; do
  wait_healthy "$service"
done

status1=$("${compose[@]}" exec -T node1 valdrd status --node http://127.0.0.1:17332)
status2=$("${compose[@]}" exec -T node2 valdrd status --node http://127.0.0.1:17332)
status3=$("${compose[@]}" exec -T node3 valdrd status --node http://127.0.0.1:17332)

python3 - "$status1" "$status2" "$status3" <<'PY'
import json, sys
for idx, raw in enumerate(sys.argv[1:], 1):
    status=json.loads(raw)
    assert status["chain_id"] == "valdr-testnet-2", (idx, status)
    assert status["network"] == "testnet2", (idx, status)
PY

wait_peer_count() {
  local service="$1"
  local status peers
  for _ in $(seq 1 30); do
    status=$("${compose[@]}" exec -T "$service" valdrd status --node http://127.0.0.1:17332)
    peers=$(python3 - "$status" <<'PY'
import json, sys
print(json.loads(sys.argv[1])["peer_count"])
PY
)
    if [[ "$peers" -ge 1 ]]; then
      return 0
    fi
    sleep 1
  done
  echo "P2P convergence failed for $service" >&2
  "${compose[@]}" logs node1 node2 node3 >&2
  return 1
}

wait_peer_count node1
wait_peer_count node2
wait_peer_count node3

wallet_json=$("${compose[@]}" exec -T node1 sh -ec '
  umask 077
  printf "docker-smoke-passphrase\n" > /tmp/valdr-pass
  exec 3</tmp/valdr-pass
  valdr-cli wallet create --dir /var/lib/valdr/wallets --name docker-smoke --password-fd 3
')
reward_address=$(python3 - "$wallet_json" <<'PY'
import json, sys
print(json.loads(sys.argv[1])["address"])
PY
)

"${compose[@]}" exec -T node1 valdr-miner start   --node http://127.0.0.1:17332   --reward-address "$reward_address"   --blocks 1   --interval 0 >/tmp/valdr-docker-miner.json

wait_height() {
  local service="$1"
  local status height
  for _ in $(seq 1 30); do
    status=$("${compose[@]}" exec -T "$service" valdrd status --node http://127.0.0.1:17332)
    height=$(python3 - "$status" <<'PY'
import json, sys
print(json.loads(sys.argv[1])["height"])
PY
)
    if [[ "$height" -ge 1 ]]; then
      return 0
    fi
    sleep 1
  done
  return 1
}

wait_height node1
wait_height node2
wait_height node3

explorer_status=$(curl --fail --silent --show-error http://127.0.0.1:8080/api/v1/status)
python3 - "$explorer_status" <<'PY'
import json, sys
status=json.loads(sys.argv[1])
assert status["status"]["chain_id"] == "valdr-testnet-2", status
assert status["status"]["height"] >= 1, status
assert status["index_height"] >= 1, status
PY

echo "VALDR Docker Testnet smoke: PASS"
