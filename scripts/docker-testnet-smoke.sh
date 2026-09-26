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
         and n["height"] == expected and n["tip_hash"] and n["chainwork"] for n in nodes)
same_tip = len({n["tip_hash"] for n in nodes}) == 1
same_work = len({n["chainwork"] for n in nodes}) == 1
raise SystemExit(0 if ok and same_tip and same_work else 1)
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

# Cross the active Testnet2 retarget boundary with two independent miners.
# Heights 3..9 retain the normal target; height 10 must apply the 10-block
# retarget using the frozen Testnet2 clamp and PoW limit.
for height in $(seq 3 10); do
  if (( height % 2 == 0 )); then
    miner_service=node2
    miner_address="$miner2_address"
  else
    miner_service=node1
    miner_address="$reward_address"
  fi
  "${compose[@]}" exec -T "$miner_service" valdr-miner start \
    --node http://127.0.0.1:17332 \
    --reward-address "$miner_address" \
    --blocks 1 --interval 0 >/dev/null
  wait_same_tip "$height"
done

block9=$("${compose[@]}" exec -T node1 valdr-cli block get \
  --node http://127.0.0.1:17332 9)
block10=$("${compose[@]}" exec -T node1 valdr-cli block get \
  --node http://127.0.0.1:17332 10)
python3 - "$block9" "$block10" <<'PY'
import json, sys
block9=json.loads(sys.argv[1])
block10=json.loads(sys.argv[2])
initial="0000031b5d43afe99ee43470e1337c3642e9d9254926038fdf6d1a2e57aaa21f"
pow_limit="000003ffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
assert block9["height"] == 9 and block9["chain_id"] == "valdr-testnet-2", block9
assert block10["height"] == 10 and block10["chain_id"] == "valdr-testnet-2", block10
assert block9["target"] == initial, block9
assert block10["target"] == pow_limit, block10
assert block10["target"] != block9["target"], (block9, block10)
PY

# Exercise two independent valdr-miner processes against two different
# Testnet2 nodes at the same height. Equal-work siblings may temporarily leave
# each node on its locally-found tip; one additional block must resolve the
# fork by cumulative chainwork and converge all nodes without manual repair.
miner1_out=$(mktemp)
miner2_out=$(mktemp)
"${compose[@]}" exec -T node1 valdr-miner start \
  --node http://127.0.0.1:17332 \
  --reward-address "$reward_address" \
  --blocks 1 --interval 0 >"$miner1_out" &
miner1_pid=$!
"${compose[@]}" exec -T node2 valdr-miner start \
  --node http://127.0.0.1:17332 \
  --reward-address "$miner2_address" \
  --blocks 1 --interval 0 >"$miner2_out" &
miner2_pid=$!

wait "$miner1_pid"
wait "$miner2_pid"
miner1_json=$(cat "$miner1_out")
miner2_json=$(cat "$miner2_out")
rm -f "$miner1_out" "$miner2_out"

python3 - "$miner1_json" "$miner2_json" "$reward_address" "$miner2_address" <<'PY'
import json, sys
m1=json.loads(sys.argv[1])
m2=json.loads(sys.argv[2])
assert m1["height"] == 11, m1
assert m2["height"] == 11, m2
assert m1["reward_address"] == sys.argv[3], m1
assert m2["reward_address"] == sys.argv[4], m2
assert m1["block_hash"] and m2["block_hash"], (m1, m2)
assert m1["block_hash"] != m2["block_hash"], (m1, m2)
assert m1["hashes_tried"] > 0 and m2["hashes_tried"] > 0, (m1, m2)
assert m1["hashrate_hps"] > 0 and m2["hashrate_hps"] > 0, (m1, m2)
PY

# Extend node1's active branch. Its greater cumulative chainwork must cause
# node2/node3 to converge even if height 11 was a temporary equal-work split.
"${compose[@]}" exec -T node1 valdr-miner start \
  --node http://127.0.0.1:17332 \
  --reward-address "$reward_address" \
  --blocks 1 --interval 0 >/dev/null
wait_same_tip 12

# Stage 14A preflight: prove node3 learned node2 before removing bootstrap.
peers3=$("${compose[@]}" exec -T node3 valdr-cli peers --node http://127.0.0.1:17332)
python3 - "$peers3" <<'PY'
import json, sys
peers=json.loads(sys.argv[1])
assert any(p.get("node_id") == "testnet-node-2" for p in peers), peers
PY

# Remove the original bootstrap node. Restart node3 while node1 remains down;
# node3 must reconnect using its persisted learned-peer cache and continue with
# node2. This is the local preflight for Stage 14A peer-cache/bootstrap-loss.
"${compose[@]}" stop node1
"${compose[@]}" restart node3
wait_healthy node3

wait_peer_count node2
wait_peer_count node3

peers3_after=$("${compose[@]}" exec -T node3 valdr-cli peers --node http://127.0.0.1:17332)
python3 - "$peers3_after" <<'PY'
import json, sys
peers=json.loads(sys.argv[1])
assert any(p.get("node_id") == "testnet-node-2" for p in peers), peers
PY

"${compose[@]}" exec -T node2 valdr-miner start \
  --node http://127.0.0.1:17332 \
  --reward-address "$miner2_address" \
  --blocks 1 --interval 0 >/dev/null

wait_pair_tip() {
  local expected_height="$1"
  local status2 status3
  for _ in $(seq 1 60); do
    status2=$("${compose[@]}" exec -T node2 valdrd status --node http://127.0.0.1:17332)
    status3=$("${compose[@]}" exec -T node3 valdrd status --node http://127.0.0.1:17332)
    if python3 - "$expected_height" "$status2" "$status3" <<'PY'
import json, sys
expected=int(sys.argv[1])
nodes=[json.loads(sys.argv[2]), json.loads(sys.argv[3])]
ok=all(n["network"]=="testnet2" and n["chain_id"]=="valdr-testnet-2"
       and n["height"]==expected and n["tip_hash"] and n["chainwork"] for n in nodes)
same_tip=len({n["tip_hash"] for n in nodes})==1
same_work=len({n["chainwork"] for n in nodes})==1
raise SystemExit(0 if ok and same_tip and same_work else 1)
PY
    then
      return 0
    fi
    sleep 1
  done
  echo "node2/node3 failed to converge without bootstrap node1 at height $expected_height" >&2
  "${compose[@]}" logs node2 node3 >&2
  return 1
}

wait_pair_tip 13
"${compose[@]}" start node1
wait_healthy node1
wait_same_tip 13

# Verify the stopped follower database after the retarget, restart it, and
# require persisted Testnet2 state to converge without DB deletion or repair.
"${compose[@]}" stop node3
verified=$("${compose[@]}" run --rm --no-deps node3 verify-db \
  --data /var/lib/valdr --network testnet2)
python3 - "$verified" <<'PY'
import json, sys
result = json.loads(sys.argv[1])
assert result["valid"] is True and result["height"] == 13, result
PY
"${compose[@]}" start node3
wait_healthy node3
wait_same_tip 13

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
assert status["status"]["height"] == 13, status
assert status["index_height"] == 13, status
PY

echo "VALDR Docker Testnet2 Core smoke: PASS"
