#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
START_SCRIPT="$ROOT_DIR/scripts/start-devnet.sh"

HOST="${VALDR_DEVNET_HOST:-127.0.0.1}"
NODE_A_RPC="${VALDR_NODE_A_RPC_PORT:-7332}"
EXPLORER_PORT="${VALDR_EXPLORER_PORT:-7631}"
WAIT_TIMEOUT="${VALDR_EXPLORER_TEST_TIMEOUT:-25}"

OWN_RUNTIME=0
if [[ -z "${VALDR_DEVNET_DIR:-}" ]]; then
  VALDR_DEVNET_DIR="$(mktemp -d "${TMPDIR:-/tmp}/valdr-v0.2-explorer.XXXXXX")"
  export VALDR_DEVNET_DIR
  OWN_RUNTIME=1
fi

BIN_DIR="$VALDR_DEVNET_DIR/bin"
WALLET_DIR="$VALDR_DEVNET_DIR/wallets"
VALDR_CLI="$BIN_DIR/valdr-cli"
VALDR_MINER="$BIN_DIR/valdr-miner"
VALDR_EXPLORER="$BIN_DIR/valdr-explorer"
EXPLORER_INDEX="$VALDR_DEVNET_DIR/explorer/index.json"
EXPLORER_LOG="$VALDR_DEVNET_DIR/explorer.log"
EXPLORER_PID=""

log() {
  printf '[VALDR EXPLORER] %s\n' "$*"
}

cleanup() {
  if [[ -n "$EXPLORER_PID" ]] && kill -0 "$EXPLORER_PID" 2>/dev/null; then
    kill -TERM "$EXPLORER_PID" 2>/dev/null || true
    wait "$EXPLORER_PID" 2>/dev/null || true
  fi
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

wait_url() {
  local url="$1"
  local deadline=$((SECONDS + WAIT_TIMEOUT))
  while (( SECONDS < deadline )); do
    if curl --fail --silent --show-error "$url" >/dev/null 2>&1; then
      return 0
    fi
    if [[ -n "$EXPLORER_PID" ]] && ! kill -0 "$EXPLORER_PID" 2>/dev/null; then
      log "explorer exited; log follows"
      cat "$EXPLORER_LOG" >&2 || true
      return 1
    fi
    sleep 0.1
  done
  log "timeout waiting for $url"
  cat "$EXPLORER_LOG" >&2 || true
  return 1
}

start_explorer() {
  mkdir -p "$(dirname "$EXPLORER_INDEX")"
  "$VALDR_EXPLORER" \
    --node "http://$HOST:$NODE_A_RPC" \
    --listen "$HOST:$EXPLORER_PORT" \
    --index-file "$EXPLORER_INDEX" \
    >"$EXPLORER_LOG" 2>&1 &
  EXPLORER_PID=$!
  wait_url "http://$HOST:$EXPLORER_PORT/healthz"
}

stop_explorer() {
  if [[ -n "$EXPLORER_PID" ]] && kill -0 "$EXPLORER_PID" 2>/dev/null; then
    kill -TERM "$EXPLORER_PID"
    wait "$EXPLORER_PID"
  fi
  EXPLORER_PID=""
}

log "starting isolated three-node devnet"
"$START_SCRIPT" start

log "building valdr-explorer"
(
  cd "$ROOT_DIR"
  go build -o "$VALDR_EXPLORER" ./cmd/valdr-explorer
)

mkdir -p "$WALLET_DIR"
wallet_json="$("$VALDR_CLI" wallet create --dir "$WALLET_DIR" --name explorer-wallet)"
address="$(printf '%s\n' "$wallet_json" | json_field address)"
if [[ -z "$address" ]]; then
  log "failed to parse wallet address"
  exit 1
fi

log "mining one live block for explorer"
"$VALDR_MINER" start \
  --node "http://$HOST:$NODE_A_RPC" \
  --reward-address "$address" \
  --blocks 1 \
  --interval 0s \
  --pid-file "$VALDR_DEVNET_DIR/explorer-miner.pid" >/dev/null

block_json="$("$VALDR_CLI" block get --node "http://$HOST:$NODE_A_RPC" 1)"
txid="$(printf '%s\n' "$block_json" | json_field transaction_id)"
if [[ -z "$txid" ]]; then
  log "failed to parse coinbase transaction ID"
  exit 1
fi

log "starting explorer against live valdrd"
start_explorer

health="$(curl --fail --silent --show-error "http://$HOST:$EXPLORER_PORT/healthz")"
printf '%s\n' "$health" | grep -q '"status":"ok"'
printf '%s\n' "$health" | grep -q '"height":1'

overview="$(curl --fail --silent --show-error "http://$HOST:$EXPLORER_PORT/")"
printf '%s\n' "$overview" | grep -q 'VALDR Explorer'
printf '%s\n' "$overview" | grep -q 'valdr-devnet-1'

block_page="$(curl --fail --silent --show-error "http://$HOST:$EXPLORER_PORT/block/1")"
printf '%s\n' "$block_page" | grep -q 'Block 1'
printf '%s\n' "$block_page" | grep -q "$txid"

tx_page="$(curl --fail --silent --show-error "http://$HOST:$EXPLORER_PORT/tx/$txid")"
printf '%s\n' "$tx_page" | grep -q "$txid"
printf '%s\n' "$tx_page" | grep -q 'Confirmed in'

address_page="$(curl --fail --silent --show-error "http://$HOST:$EXPLORER_PORT/address/$address")"
printf '%s\n' "$address_page" | grep -q '50 VDR'
printf '%s\n' "$address_page" | grep -q 'Confirmed activity'
printf '%s\n' "$address_page" | grep -q "$txid"

if [[ ! -s "$EXPLORER_INDEX" ]]; then
  log "persistent explorer index was not created"
  exit 1
fi

log "restarting explorer from persistent index"
stop_explorer
start_explorer

address_after_restart="$(curl --fail --silent --show-error "http://$HOST:$EXPLORER_PORT/address/$address")"
printf '%s\n' "$address_after_restart" | grep -q '50 VDR'
printf '%s\n' "$address_after_restart" | grep -q "$txid"

log "PASS: live node + block + tx + address history + persistent explorer restart"
