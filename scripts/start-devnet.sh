#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="${1:-start}"

DEVNET_DIR="${VALDR_DEVNET_DIR:-$ROOT_DIR/.valdr-devnet}"
BIN_DIR="$DEVNET_DIR/bin"
RUN_DIR="$DEVNET_DIR/run"
LOG_DIR="$DEVNET_DIR/logs"
DATA_DIR="$DEVNET_DIR/data"

HOST="${VALDR_DEVNET_HOST:-127.0.0.1}"

NODE_A_P2P="${VALDR_NODE_A_P2P_PORT:-7333}"
NODE_A_RPC="${VALDR_NODE_A_RPC_PORT:-7332}"
NODE_B_P2P="${VALDR_NODE_B_P2P_PORT:-7433}"
NODE_B_RPC="${VALDR_NODE_B_RPC_PORT:-7432}"
NODE_C_P2P="${VALDR_NODE_C_P2P_PORT:-7533}"
NODE_C_RPC="${VALDR_NODE_C_RPC_PORT:-7532}"

START_TIMEOUT_SECONDS="${VALDR_DEVNET_START_TIMEOUT:-20}"

VALDRD="$BIN_DIR/valdrd"
VALDR_CLI="$BIN_DIR/valdr-cli"

mkdir -p "$BIN_DIR" "$RUN_DIR" "$LOG_DIR" "$DATA_DIR"

log() {
  printf '[VALDR DEVNET] %s\n' "$*"
}

pid_file() {
  printf '%s/%s.pid\n' "$RUN_DIR" "$1"
}

node_is_running() {
  local node="$1"
  local file
  file="$(pid_file "$node")"
  [[ -f "$file" ]] || return 1

  local pid
  pid="$(cat "$file")"
  [[ "$pid" =~ ^[0-9]+$ ]] || return 1
  kill -0 "$pid" 2>/dev/null
}

assert_not_running() {
  local node="$1"
  if node_is_running "$node"; then
    log "$node is already running with PID $(cat "$(pid_file "$node")")"
    return 1
  fi
  rm -f "$(pid_file "$node")"
}

build_binaries() {
  log "building valdrd and valdr-cli"
  (
    cd "$ROOT_DIR"
    go build -o "$VALDRD" ./cmd/valdrd
    go build -o "$VALDR_CLI" ./cmd/valdr-cli
  )
}

init_node() {
  local node="$1"
  "$VALDRD" init --data "$DATA_DIR/$node" >/dev/null
}

start_node() {
  local node="$1"
  local p2p_port="$2"
  local rpc_port="$3"
  shift 3

  assert_not_running "$node"
  init_node "$node"

  local logfile="$LOG_DIR/$node.log"
  log "starting $node p2p=$HOST:$p2p_port rpc=$HOST:$rpc_port"

  nohup "$VALDRD" start \
    --data "$DATA_DIR/$node" \
    --node-id "$node" \
    --p2p-host "$HOST" \
    --p2p-port "$p2p_port" \
    --rpc-host "$HOST" \
    --rpc-port "$rpc_port" \
    "$@" \
    >"$logfile" 2>&1 &

  local pid=$!
  printf '%s\n' "$pid" >"$(pid_file "$node")"
}

wait_rpc() {
  local node="$1"
  local rpc_port="$2"
  local deadline=$((SECONDS + START_TIMEOUT_SECONDS))

  while (( SECONDS < deadline )); do
    if "$VALDR_CLI" status --node "http://$HOST:$rpc_port" >/dev/null 2>&1; then
      log "$node RPC ready"
      return 0
    fi

    if ! node_is_running "$node"; then
      log "$node exited during startup; see $LOG_DIR/$node.log"
      return 1
    fi
    sleep 0.2
  done

  log "timeout waiting for $node RPC; see $LOG_DIR/$node.log"
  return 1
}

status_json() {
  local rpc_port="$1"
  "$VALDR_CLI" status --node "http://$HOST:$rpc_port"
}

status_field() {
  local rpc_port="$1"
  local field="$2"
  status_json "$rpc_port" |
    awk -v key="\"$field\":" '
      $1 == key {
        gsub(/[",]/, "", $2)
        print $2
        exit
      }
    '
}

wait_peer_count() {
  local node="$1"
  local rpc_port="$2"
  local want="$3"
  local deadline=$((SECONDS + START_TIMEOUT_SECONDS))

  while (( SECONDS < deadline )); do
    local got
    got="$(status_field "$rpc_port" peer_count 2>/dev/null || true)"
    if [[ "$got" == "$want" ]]; then
      log "$node peer_count=$got"
      return 0
    fi
    sleep 0.2
  done

  local got
  got="$(status_field "$rpc_port" peer_count 2>/dev/null || printf '?')"
  log "$node peer_count=$got, want $want"
  return 1
}

verify_topology() {
  log "verifying three-node RPC and P2P topology"

  wait_rpc node-a "$NODE_A_RPC"
  wait_rpc node-b "$NODE_B_RPC"
  wait_rpc node-c "$NODE_C_RPC"

  wait_peer_count node-a "$NODE_A_RPC" 1
  wait_peer_count node-b "$NODE_B_RPC" 2
  wait_peer_count node-c "$NODE_C_RPC" 1

  local chain_a chain_b chain_c
  chain_a="$(status_field "$NODE_A_RPC" chain_id)"
  chain_b="$(status_field "$NODE_B_RPC" chain_id)"
  chain_c="$(status_field "$NODE_C_RPC" chain_id)"

  if [[ "$chain_a" != "valdr-devnet-1" ||
        "$chain_b" != "$chain_a" ||
        "$chain_c" != "$chain_a" ]]; then
    log "chain ID mismatch: A=$chain_a B=$chain_b C=$chain_c"
    return 1
  fi

  local height_a height_b height_c
  height_a="$(status_field "$NODE_A_RPC" height)"
  height_b="$(status_field "$NODE_B_RPC" height)"
  height_c="$(status_field "$NODE_C_RPC" height)"
  if [[ "$height_a" != "$height_b" || "$height_c" != "$height_a" ]]; then
    log "height mismatch: A=$height_a B=$height_b C=$height_c"
    return 1
  fi

  local tip_a tip_b tip_c
  tip_a="$(status_field "$NODE_A_RPC" tip_hash)"
  tip_b="$(status_field "$NODE_B_RPC" tip_hash)"
  tip_c="$(status_field "$NODE_C_RPC" tip_hash)"
  if [[ -z "$tip_a" || "$tip_b" != "$tip_a" || "$tip_c" != "$tip_a" ]]; then
    log "tip hash mismatch: A=$tip_a B=$tip_b C=$tip_c"
    return 1
  fi

  log "devnet healthy: chain_id=$chain_a height=$height_a tip=$tip_a"
}

start_devnet() {
  build_binaries

  start_node node-a "$NODE_A_P2P" "$NODE_A_RPC"
  if ! wait_rpc node-a "$NODE_A_RPC"; then
    stop_devnet
    return 1
  fi

  start_node node-b "$NODE_B_P2P" "$NODE_B_RPC" \
    --peer "$HOST:$NODE_A_P2P"
  if ! wait_rpc node-b "$NODE_B_RPC"; then
    stop_devnet
    return 1
  fi

  start_node node-c "$NODE_C_P2P" "$NODE_C_RPC" \
    --peer "$HOST:$NODE_B_P2P"
  if ! wait_rpc node-c "$NODE_C_RPC"; then
    stop_devnet
    return 1
  fi

  if ! verify_topology; then
    stop_devnet
    return 1
  fi

  log "VALDR Devnet is running"
  log "Node A RPC: http://$HOST:$NODE_A_RPC"
  log "Node B RPC: http://$HOST:$NODE_B_RPC"
  log "Node C RPC: http://$HOST:$NODE_C_RPC"
  log "logs: $LOG_DIR"
  log "stop: ./scripts/start-devnet.sh stop"
}

stop_node() {
  local node="$1"
  local file
  file="$(pid_file "$node")"

  if [[ ! -f "$file" ]]; then
    return 0
  fi

  local pid
  pid="$(cat "$file")"
  if [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 "$pid" 2>/dev/null; then
    log "stopping $node PID=$pid"
    kill -TERM "$pid" 2>/dev/null || true

    local deadline=$((SECONDS + 10))
    while kill -0 "$pid" 2>/dev/null && (( SECONDS < deadline )); do
      sleep 0.1
    done

    if kill -0 "$pid" 2>/dev/null; then
      log "$node did not stop gracefully; sending KILL"
      kill -KILL "$pid" 2>/dev/null || true
    fi
  fi

  rm -f "$file"
}

stop_devnet() {
  stop_node node-c
  stop_node node-b
  stop_node node-a
  log "VALDR Devnet stopped"
}

show_status() {
  build_binaries

  local failed=0
  for spec in     "node-a:$NODE_A_RPC"     "node-b:$NODE_B_RPC"     "node-c:$NODE_C_RPC"; do
    local node="${spec%%:*}"
    local port="${spec##*:}"

    if node_is_running "$node"; then
      log "$node PID=$(cat "$(pid_file "$node")")"
      status_json "$port" || failed=1
    else
      log "$node is not running"
      failed=1
    fi
  done
  return "$failed"
}

SMOKE_DIR=""

cleanup_smoke() {
  if [[ -z "$SMOKE_DIR" ]]; then
    return 0
  fi

  DEVNET_DIR="$SMOKE_DIR"
  BIN_DIR="$DEVNET_DIR/bin"
  RUN_DIR="$DEVNET_DIR/run"
  LOG_DIR="$DEVNET_DIR/logs"
  DATA_DIR="$DEVNET_DIR/data"
  VALDRD="$BIN_DIR/valdrd"
  VALDR_CLI="$BIN_DIR/valdr-cli"

  stop_devnet || true
  rm -rf "$SMOKE_DIR"
  SMOKE_DIR=""
}

smoke_test() {
  SMOKE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/valdr-devnet-smoke.XXXXXX")"
  trap cleanup_smoke EXIT INT TERM

  DEVNET_DIR="$SMOKE_DIR"
  BIN_DIR="$DEVNET_DIR/bin"
  RUN_DIR="$DEVNET_DIR/run"
  LOG_DIR="$DEVNET_DIR/logs"
  DATA_DIR="$DEVNET_DIR/data"
  VALDRD="$BIN_DIR/valdrd"
  VALDR_CLI="$BIN_DIR/valdr-cli"
  mkdir -p "$BIN_DIR" "$RUN_DIR" "$LOG_DIR" "$DATA_DIR"

  log "running isolated smoke test in $DEVNET_DIR"
  start_devnet
  verify_topology
  log "smoke test passed"

  cleanup_smoke
  trap - EXIT INT TERM
}

case "$MODE" in
  start)
    start_devnet
    ;;
  stop)
    stop_devnet
    ;;
  status)
    show_status
    ;;
  smoke)
    smoke_test
    ;;
  *)
    printf 'usage: %s [start|stop|status|smoke]\n' "$0" >&2
    exit 2
    ;;
esac
