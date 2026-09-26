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
  printf "docker-smoke-passphrase\\n" > /tmp/valdr-pass
  exec 3</tmp/valdr-pass
  valdr-cli wallet create --dir /var/lib/valdr/wallets --name docker-smoke --password-fd 3
')
reward_address=$(python3 - "$wallet_json" <<'PY'
import json, sys
print(json.loads(sys.argv[1])["address"])
PY
)

miner2_wallet_json=$("${compose[@]}" exec -T node2 sh -ec '
  umask 077
  printf "docker-miner2-passphrase\\n" > /tmp/valdr-miner2-pass
  exec 3</tmp/valdr-miner2-pass
  valdr-cli wallet create --dir /var/lib/valdr/wallets --name docker-miner-2 --password-fd 3
')
miner2_address=$(python3 - "$miner2_wallet_json" <<'PY'
import json, sys
print(json.loads(sys.argv[1])["address"])
PY
)

recipient_wallet_json=$("${compose[@]}" exec -T node3 sh -ec '
  umask 077
  printf "docker-recipient-passphrase\\n" > /tmp/valdr-recipient-pass
  exec 3</tmp/valdr-recipient-pass
  valdr-cli wallet create --dir /var/lib/valdr/wallets --name docker-recipient --password-fd 3
')
recipient_address=$(python3 - "$recipient_wallet_json" <<'PY'
import json, sys
print(json.loads(sys.argv[1])["address"])
PY
)

"${compose[@]}" exec -T node1 valdr-miner start \
  --node http://127.0.0.1:17332 \
  --reward-address "$reward_address" \
  --blocks 1 --interval 0 >/tmp/valdr-docker-miner.json

wait_height() {
  local service="$1"
  local expected_height="$2"
  local status height
  for _ in $(seq 1 60); do
    status=$("${compose[@]}" exec -T "$service" valdrd status --node http://127.0.0.1:17332)
    height=$(python3 - "$status" <<'PY'
import json, sys
print(json.loads(sys.argv[1])["height"])
PY
)
    if [[ "$height" -ge "$expected_height" ]]; then
      return 0
    fi
    sleep 1
  done
  echo "height convergence failed for $service at expected height $expected_height" >&2
  "${compose[@]}" logs node1 node2 node3 >&2
  return 1
}

wait_same_tip() {
  local expected_height="$1"
  local statuses
  for _ in $(seq 1 60); do
    statuses=()
    for service in node1 node2 node3; do
      statuses+=("$("${compose[@]}" exec -T "$service" valdrd status --node http://127.0.0.1:17332)")
    done
    if python3 - "$expected_height" "${statuses[@]}" <<'PY'
import json, sys
expected = int(sys.argv[1])
nodes = [json.loads(raw) for raw in sys.argv[2:]]
ok = all(n["network"] == "testnet2" and n["chain_id"] == "valdr-testnet-2"
         and n["height"] == expected and n["tip_hash"] for n in nodes)
raise SystemExit(0 if ok and len({n["tip_hash"] for n in nodes}) == 1 else 1)
PY
    then
      return 0
    fi
    sleep 1
  done
  echo "Testnet2 three-node tip convergence failed at height $expected_height" >&2
  "${compose[@]}" logs node1 node2 node3 >&2
  return 1
}

wait_height node1 1
wait_height node2 1
wait_height node3 1
wait_same_tip 1

# Exercise the real Testnet2 transaction path before the second block:
# encrypted wallet -> local signing -> RPC -> mempool -> P2P relay.
send_json=$("${compose[@]}" exec -T node1 sh -ec '
  umask 077
  printf "docker-smoke-passphrase\\n" > /tmp/valdr-pass
  exec 3</tmp/valdr-pass
  valdr-cli send \
    --node http://127.0.0.1:17332 \
    --wallet-dir /var/lib/valdr/wallets \
    --from docker-smoke \
    --to "$1" \
    --amount 0.25 \
    --password-fd 3
' sh "$recipient_address")
txid=$(python3 - "$send_json" <<'PY'
import json, sys
value=json.loads(sys.argv[1])
txid=value["transaction_id"]
assert isinstance(txid, str) and len(txid) == 64, value
print(txid)
PY
)

wait_mempool_tx() {
  local service="$1"
  local wanted_txid="$2"
  local raw
  for _ in $(seq 1 60); do
    raw=$("${compose[@]}" exec -T "$service" valdr-cli mempool --node http://127.0.0.1:17332)
    if python3 - "$wanted_txid" "$raw" <<'PY'
import json, sys
wanted=sys.argv[1]
txs=json.loads(sys.argv[2])
raise SystemExit(0 if any(tx.get("transaction_id") == wanted for tx in txs) else 1)
PY
    then
      return 0
    fi
    sleep 1
  done
  echo "transaction $wanted_txid did not reach $service mempool" >&2
  "${compose[@]}" logs node1 node2 node3 >&2
  return 1
}

wait_mempool_tx node1 "$txid"
wait_mempool_tx node2 "$txid"
wait_mempool_tx node3 "$txid"

unconfirmed=$("${compose[@]}" exec -T node3 valdr-cli tx get \
  --node http://127.0.0.1:17332 "$txid")
python3 - "$txid" "$unconfirmed" <<'PY'
import json, sys
wanted=sys.argv[1]
result=json.loads(sys.argv[2])
assert result["confirmed"] is False, result
assert result["transaction"]["transaction_id"] == wanted, result
assert result["transaction"]["chain_id"] == "valdr-testnet-2", result
PY

explorer_mempool=$(curl --fail --silent --show-error http://127.0.0.1:8080/api/v1/mempool)
python3 - "$txid" "$explorer_mempool" <<'PY'
import json, sys
wanted=sys.argv[1]
txs=json.loads(sys.argv[2])
assert any(tx.get("transaction_id") == wanted for tx in txs), txs
PY

# Mine the relayed transaction on a different node with a different reward
# wallet, then require all three independent Badger stores to converge.
"${compose[@]}" exec -T node2 valdr-miner start \
  --node http://127.0.0.1:17332 \
  --reward-address "$miner2_address" \
  --blocks 1 --interval 0 >/dev/null
wait_same_tip 2

for service in node1 node2 node3; do
  confirmed=$("${compose[@]}" exec -T "$service" valdr-cli tx get \
    --node http://127.0.0.1:17332 "$txid")
  python3 - "$txid" "$confirmed" <<'PY'
import json, sys
wanted=sys.argv[1]
result=json.loads(sys.argv[2])
assert result["confirmed"] is True, result
assert result["block_height"] == 2, result
assert result["transaction"]["transaction_id"] == wanted, result
assert result["transaction"]["chain_id"] == "valdr-testnet-2", result
PY

  status=$("${compose[@]}" exec -T "$service" valdrd status --node http://127.0.0.1:17332)
  python3 - "$status" <<'PY'
import json, sys
status=json.loads(sys.argv[1])
assert status["mempool_count"] == 0, status
PY

  balance=$("${compose[@]}" exec -T "$service" valdr-cli balance \
    --node http://127.0.0.1:17332 "$recipient_address")
  python3 - "$recipient_address" "$balance" <<'PY'
import json, sys
address=sys.argv[1]
balance=json.loads(sys.argv[2])
assert balance["address"] == address, balance
assert balance["balance_val"] == 25_000_000, balance
PY
done

explorer_tx=$(curl --fail --silent --show-error "http://127.0.0.1:8080/api/v1/tx/$txid")
python3 - "$txid" "$explorer_tx" <<'PY'
import json, sys
wanted=sys.argv[1]
result=json.loads(sys.argv[2])
assert result["confirmed"] is True and result["block_height"] == 2, result
assert result["transaction"]["transaction_id"] == wanted, result
PY

explorer_address=$(curl --fail --silent --show-error \
  "http://127.0.0.1:8080/api/v1/address/$recipient_address")
python3 - "$recipient_address" "$explorer_address" <<'PY'
import json, sys
address=sys.argv[1]
result=json.loads(sys.argv[2])
assert result["balance"]["address"] == address, result
assert result["balance"]["balance_val"] == 25_000_000, result
assert result["utxos"], result
assert result["history"], result
PY

# Verify the stopped follower database, restart it, and require persisted
# Testnet2 state to converge without deleting or repairing the database.
"${compose[@]}" stop node3
verified=$("${compose[@]}" run --rm --no-deps node3 verify-db \
  --data /var/lib/valdr --network testnet2)
python3 - "$verified" <<'PY'
import json, sys
result = json.loads(sys.argv[1])
assert result["valid"] is True and result["height"] == 2, result
PY
"${compose[@]}" start node3
wait_healthy node3
wait_same_tip 2

restart_balance=$("${compose[@]}" exec -T node3 valdr-cli balance \
  --node http://127.0.0.1:17332 "$recipient_address")
python3 - "$restart_balance" <<'PY'
import json, sys
balance=json.loads(sys.argv[1])
assert balance["balance_val"] == 25_000_000, balance
PY

explorer_status=$(curl --fail --silent --show-error http://127.0.0.1:8080/api/v1/status)
python3 - "$explorer_status" <<'PY'
import json, sys
status=json.loads(sys.argv[1])
assert status["status"]["chain_id"] == "valdr-testnet-2", status
assert status["status"]["network"] == "testnet2", status
assert status["status"]["height"] == 2, status
assert status["index_height"] == 2, status
PY

echo "VALDR Docker Testnet2 Core smoke: PASS"
