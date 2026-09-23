#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
START_SCRIPT="$ROOT_DIR/scripts/start-devnet.sh"

HOST="${VALDR_DEVNET_HOST:-127.0.0.1}"
NODE_A_RPC="${VALDR_NODE_A_RPC_PORT:-7332}"
NODE_B_RPC="${VALDR_NODE_B_RPC_PORT:-7432}"
NODE_C_RPC="${VALDR_NODE_C_RPC_PORT:-7532}"
WAIT_TIMEOUT="${VALDR_FINAL_TEST_TIMEOUT:-25}"

OWN_RUNTIME=0
if [[ -z "${VALDR_DEVNET_DIR:-}" ]]; then
  VALDR_DEVNET_DIR="$(mktemp -d "${TMPDIR:-/tmp}/valdr-v0.1-final.XXXXXX")"
  export VALDR_DEVNET_DIR
  OWN_RUNTIME=1
fi

BIN_DIR="$VALDR_DEVNET_DIR/bin"
DATA_DIR="$VALDR_DEVNET_DIR/data"
WALLET_DIR="$VALDR_DEVNET_DIR/wallets"
VALDR_CLI="$BIN_DIR/valdr-cli"
VALDR_MINER="$BIN_DIR/valdr-miner"
MINER_PID_FILE="$VALDR_DEVNET_DIR/miner.pid"

log() {
  printf '[VALDR FINAL] %s\n' "$*"
}

cleanup() {
  "$START_SCRIPT" stop >/dev/null 2>&1 || true
  if [[ "$OWN_RUNTIME" == "1" ]]; then
    rm -rf "$VALDR_DEVNET_DIR"
  fi
}
trap cleanup EXIT INT TERM

json_field() {
  local field="$1"
  awk -v key="\"$field\":" '
    $1 == key {
      gsub(/[",]/, "", $2)
      print $2
      exit
    }
  '
}

status_json() {
  local port="$1"
  "$VALDR_CLI" status --node "http://$HOST:$port"
}

status_field() {
  local port="$1"
  local field="$2"
  status_json "$port" | json_field "$field"
}

wait_status_field() {
  local node="$1"
  local port="$2"
  local field="$3"
  local want="$4"
  local deadline=$((SECONDS + WAIT_TIMEOUT))

  while (( SECONDS < deadline )); do
    local got
    got="$(status_field "$port" "$field" 2>/dev/null || true)"
    if [[ "$got" == "$want" ]]; then
      return 0
    fi
    sleep 0.1
  done

  local got
  got="$(status_field "$port" "$field" 2>/dev/null || printf '?')"
  log "$node $field=$got, want $want"
  return 1
}

balance_val() {
  local port="$1"
  local address="$2"
  "$VALDR_CLI" balance     --node "http://$HOST:$port"     "$address" |
    json_field balance_val
}

assert_balance_all() {
  local address="$1"
  local want="$2"
  local label="$3"

  for spec in     "node-a:$NODE_A_RPC"     "node-b:$NODE_B_RPC"     "node-c:$NODE_C_RPC"; do
    local node="${spec%%:*}"
    local port="${spec##*:}"
    local got
    got="$(balance_val "$port" "$address")"
    if [[ "$got" != "$want" ]]; then
      log "$label balance on $node = $got, want $want"
      return 1
    fi
  done
}

tx_confirmed() {
  local port="$1"
  local txid="$2"
  "$VALDR_CLI" tx get     --node "http://$HOST:$port"     "$txid" |
    json_field confirmed
}

wait_tx_confirmed_all() {
  local txid="$1"
  local want="$2"
  local deadline=$((SECONDS + WAIT_TIMEOUT))

  while (( SECONDS < deadline )); do
    local a b c
    a="$(tx_confirmed "$NODE_A_RPC" "$txid" 2>/dev/null || true)"
    b="$(tx_confirmed "$NODE_B_RPC" "$txid" 2>/dev/null || true)"
    c="$(tx_confirmed "$NODE_C_RPC" "$txid" 2>/dev/null || true)"
    if [[ "$a" == "$want" && "$b" == "$want" && "$c" == "$want" ]]; then
      return 0
    fi
    sleep 0.1
  done
  log "transaction $txid did not reach confirmed=$want on all nodes"
  return 1
}

log "starting three-node VALDR Devnet"
"$START_SCRIPT" start

mkdir -p "$WALLET_DIR"

log "creating wallet A"
wallet_a_json="$("$VALDR_CLI" wallet create --dir "$WALLET_DIR" --name wallet-a)"
wallet_a="$(printf '%s\n' "$wallet_a_json" | json_field address)"
if [[ -z "$wallet_a" ]]; then
  log "failed to parse wallet A address"
  exit 1
fi

log "creating wallet B"
wallet_b_json="$("$VALDR_CLI" wallet create --dir "$WALLET_DIR" --name wallet-b)"
wallet_b="$(printf '%s\n' "$wallet_b_json" | json_field address)"
if [[ -z "$wallet_b" ]]; then
  log "failed to parse wallet B address"
  exit 1
fi

log "wallet A=$wallet_a"
log "wallet B=$wallet_b"

log "mining first 50 VDR block to wallet A"
"$VALDR_MINER" start   --node "http://$HOST:$NODE_A_RPC"   --reward-address "$wallet_a"   --blocks 1   --interval 0s   --pid-file "$MINER_PID_FILE" >/dev/null

wait_status_field node-a "$NODE_A_RPC" height 1
wait_status_field node-b "$NODE_B_RPC" height 1
wait_status_field node-c "$NODE_C_RPC" height 1
assert_balance_all "$wallet_a" 5000000000 "wallet A after first mine"

log "sending 10 VDR from wallet A to wallet B"
send_json="$("$VALDR_CLI" send   --node "http://$HOST:$NODE_A_RPC"   --wallet-dir "$WALLET_DIR"   --from wallet-a   --to "$wallet_b"   --amount 10)"
txid="$(printf '%s\n' "$send_json" | json_field transaction_id)"
if [[ -z "$txid" ]]; then
  log "failed to parse transaction id"
  exit 1
fi
log "txid=$txid"

wait_status_field node-a "$NODE_A_RPC" mempool_count 1
wait_status_field node-b "$NODE_B_RPC" mempool_count 1
wait_status_field node-c "$NODE_C_RPC" mempool_count 1
wait_tx_confirmed_all "$txid" false

log "mining confirming block on Node A"
"$VALDR_MINER" start   --node "http://$HOST:$NODE_A_RPC"   --reward-address "$wallet_a"   --blocks 1   --interval 0s   --pid-file "$MINER_PID_FILE" >/dev/null

wait_status_field node-a "$NODE_A_RPC" height 2
wait_status_field node-b "$NODE_B_RPC" height 2
wait_status_field node-c "$NODE_C_RPC" height 2
wait_status_field node-a "$NODE_A_RPC" mempool_count 0
wait_status_field node-b "$NODE_B_RPC" mempool_count 0
wait_status_field node-c "$NODE_C_RPC" mempool_count 0
wait_tx_confirmed_all "$txid" true

assert_balance_all "$wallet_b" 1000000000 "wallet B"
assert_balance_all "$wallet_a" 9000000000 "wallet A"

tip_before="$(status_field "$NODE_A_RPC" tip_hash)"
if [[ -z "$tip_before" ]]; then
  log "empty tip hash before restart"
  exit 1
fi
for port in "$NODE_B_RPC" "$NODE_C_RPC"; do
  if [[ "$(status_field "$port" tip_hash)" != "$tip_before" ]]; then
    log "tip mismatch before restart"
    exit 1
  fi
done

log "checking persistent blockchain files"
for node in node-a node-b node-c; do
  file="$DATA_DIR/$node/blockchain.json"
  if [[ ! -s "$file" ]]; then
    log "missing persisted chain: $file"
    exit 1
  fi
done

log "stopping all three nodes"
"$START_SCRIPT" stop

log "restarting all three nodes from the same data directories"
"$START_SCRIPT" start

wait_status_field node-a "$NODE_A_RPC" height 2
wait_status_field node-b "$NODE_B_RPC" height 2
wait_status_field node-c "$NODE_C_RPC" height 2

for port in "$NODE_A_RPC" "$NODE_B_RPC" "$NODE_C_RPC"; do
  if [[ "$(status_field "$port" tip_hash)" != "$tip_before" ]]; then
    log "persisted tip mismatch after restart on RPC port $port"
    exit 1
  fi
done

wait_tx_confirmed_all "$txid" true
assert_balance_all "$wallet_b" 1000000000 "wallet B after restart"
assert_balance_all "$wallet_a" 9000000000 "wallet A after restart"

log "PASS: 3 nodes + wallets + mine + send + broadcast + mine + sync + balance + restart + persistence"
