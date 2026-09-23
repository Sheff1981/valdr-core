#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
D="${VALDR_QA_ROOT:-$ROOT/testnet-data}"
BIN="${VALDR_TESTNET_DAEMON:-$ROOT/release/bin/valdrd}"
CLI="${VALDR_TESTNET_CLI:-$ROOT/release/bin/valdr-cli}"
[[ -x "$BIN" && -x "$CLI" ]] || { echo 'Compiled valdrd/valdr-cli required.' >&2; exit 1; }

rm -rf "$D"
mkdir -p "$D"/{n1,n2,n3}
COMMON=(-testnet4 -server=1 -listen=1 -dnsseed=0 -fixedseeds=0 -fallbackfee=0.00001000 -daemonwait)

cleanup(){
  "$CLI" -testnet4 -datadir="$D/n1" -rpcport=37332 stop >/dev/null 2>&1 || true
  "$CLI" -testnet4 -datadir="$D/n2" -rpcport=37342 stop >/dev/null 2>&1 || true
  "$CLI" -testnet4 -datadir="$D/n3" -rpcport=37352 stop >/dev/null 2>&1 || true
}
trap cleanup EXIT

start_node() {
  local name="$1"; shift
  if ! "$BIN" "$@"; then
    echo "=== $name startup failed ===" >&2
    find "$D/$name" -name debug.log -type f -maxdepth 3 -print -exec tail -n 250 {} \; >&2 || true
    exit 1
  fi
}

start_node n1 -datadir="$D/n1" -port=37333 -rpcport=37332 "${COMMON[@]}"
start_node n2 -datadir="$D/n2" -port=37334 -rpcport=37342 -connect=127.0.0.1:37333 "${COMMON[@]}"
start_node n3 -datadir="$D/n3" -port=37335 -rpcport=37352 -connect=127.0.0.1:37333 "${COMMON[@]}"

C1=("$CLI" -testnet4 -datadir="$D/n1" -rpcport=37332)
C2=("$CLI" -testnet4 -datadir="$D/n2" -rpcport=37342)
C3=("$CLI" -testnet4 -datadir="$D/n3" -rpcport=37352)

"${C1[@]}" createwallet miner >/dev/null
A1="$("${C1[@]}" -rpcwallet=miner getnewaddress miner)"
"${C1[@]}" -rpcwallet=miner generatetoaddress 101 "$A1" >/dev/null

for _ in $(seq 1 90); do
  [[ "$("${C2[@]}" getblockcount)" == 101 && "$("${C3[@]}" getblockcount)" == 101 ]] && break
  sleep 1
done
[[ "$("${C2[@]}" getblockcount)" == 101 && "$("${C3[@]}" getblockcount)" == 101 ]] || { echo '3-node initial sync failed' >&2; exit 1; }

"${C2[@]}" createwallet receiver >/dev/null
A2="$("${C2[@]}" -rpcwallet=receiver getnewaddress receiver)"
TX="$("${C1[@]}" -rpcwallet=miner sendtoaddress "$A2" 1.00000000)"
"${C1[@]}" -rpcwallet=miner generatetoaddress 1 "$A1" >/dev/null

for _ in $(seq 1 90); do
  [[ "$("${C2[@]}" getblockcount)" == 102 && "$("${C3[@]}" getblockcount)" == 102 ]] && break
  sleep 1
done

H1="$("${C1[@]}" getbestblockhash)"
H2="$("${C2[@]}" getbestblockhash)"
H3="$("${C3[@]}" getbestblockhash)"
[[ "$H1" == "$H2" && "$H1" == "$H3" ]] || { echo '3-node tip mismatch' >&2; exit 1; }

BAL="$("${C2[@]}" -rpcwallet=receiver getbalance)"
python3 - "$BAL" <<'PY'
from decimal import Decimal
import sys
assert Decimal(sys.argv[1]) >= Decimal("1.0"), sys.argv[1]
PY

printf 'VALDR TESTNET 3-NODE QA PASS\nheight=%s\ntip=%s\ntxid=%s\nreceiver=%s\nbalance=%s VLD\n'   "$("${C1[@]}" getblockcount)" "$H1" "$TX" "$A2" "$BAL"
