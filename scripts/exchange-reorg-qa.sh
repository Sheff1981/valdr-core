#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DAEMON="${VALDR_TESTNET_DAEMON:-$ROOT/release/bin/valdrd}"
CLI="${VALDR_TESTNET_CLI:-$ROOT/release/bin/valdr-cli}"
[[ -x "$DAEMON" && -x "$CLI" ]] || { echo 'Compiled valdrd/valdr-cli required.' >&2; exit 1; }

Q="${VALDR_QA_ROOT:-$ROOT/work/reorg-qa}"
A="$Q/a"
B="$Q/b"
rm -rf "$Q"
mkdir -p "$A" "$B"
PORTA=38341
PORTB=38343
RPCA=38351
RPCB=38352

cleanup(){
  "$CLI" -testnet4 -datadir="$A" -rpcport=$RPCA stop >/dev/null 2>&1 || true
  "$CLI" -testnet4 -datadir="$B" -rpcport=$RPCB stop >/dev/null 2>&1 || true
}
trap cleanup EXIT

"$DAEMON" -testnet4 -datadir="$A" -port=$PORTA -rpcport=$RPCA -server=1 -listen=1 -listenonion=0 -natpmp=0 -dnsseed=0 -fixedseeds=0 -fallbackfee=0.00001 -daemonwait
"$DAEMON" -testnet4 -datadir="$B" -port=$PORTB -rpcport=$RPCB -server=1 -listen=1 -dnsseed=0 -fixedseeds=0 -fallbackfee=0.00001 -daemonwait

Acli(){ "$CLI" -testnet4 -datadir="$A" -rpcport=$RPCA "$@"; }
Bcli(){ "$CLI" -testnet4 -datadir="$B" -rpcport=$RPCB "$@"; }

Acli createwallet miner >/dev/null
Bcli createwallet exchange >/dev/null
AA=$(Acli -rpcwallet=miner getnewaddress)
BA=$(Bcli -rpcwallet=exchange getnewaddress)

Acli addnode "127.0.0.1:$PORTB" onetry >/dev/null
Acli -rpcwallet=miner generatetoaddress 101 "$AA" >/dev/null

for _ in $(seq 1 90); do
  [[ $(Bcli getblockcount) == 101 ]] && break
  sleep 1
done
[[ $(Bcli getblockcount) == 101 ]] || { echo 'initial sync failed' >&2; exit 1; }

TXID=$(Acli -rpcwallet=miner sendtoaddress "$BA" 1.0)
Acli -rpcwallet=miner generatetoaddress 6 "$AA" >/dev/null

for _ in $(seq 1 90); do
  [[ $(Bcli getblockcount) == 107 ]] && break
  sleep 1
done
[[ $(Bcli getblockcount) == 107 ]] || { echo 'deposit confirmation sync failed' >&2; exit 1; }

PRE_JSON=$(Bcli -rpcwallet=exchange gettransaction "$TXID")
PRE=$(python3 -c 'import sys,json; print(json.load(sys.stdin).get("confirmations",0))' <<<"$PRE_JSON")
TXBLOCK=$(python3 -c 'import sys,json; print(json.load(sys.stdin).get("blockhash",""))' <<<"$PRE_JSON")
(( PRE >= 6 )) || { echo "deposit never reached six confirmations: $PRE" >&2; exit 1; }

Acli disconnectnode "127.0.0.1:$PORTB" >/dev/null 2>&1 || true
Bcli invalidateblock "$TXBLOCK" >/dev/null
[[ $(Bcli getblockcount) == 101 ]] || { echo "unexpected rollback height: $(Bcli getblockcount)" >&2; exit 1; }

for _ in $(seq 1 8); do
  Bcli generateblock "$BA" '[]' >/dev/null
done
[[ $(Bcli getblockcount) == 109 ]] || { echo 'competing branch build failed' >&2; exit 1; }

Acli addnode "127.0.0.1:$PORTB" onetry >/dev/null
for _ in $(seq 1 90); do
  [[ $(Acli getbestblockhash) == $(Bcli getbestblockhash) ]] && break
  sleep 1
done
[[ $(Acli getbestblockhash) == $(Bcli getbestblockhash) ]] || { echo 'reorg convergence failed' >&2; exit 1; }

POST=$(Bcli -rpcwallet=exchange gettransaction "$TXID" 2>/dev/null | python3 -c 'import sys,json; print(json.load(sys.stdin).get("confirmations",0))' || echo 0)
(( POST <= 0 )) || { echo 'FAIL: orphaned deposit retained positive confirmations' >&2; exit 1; }

echo "VALDR exchange deep-reorg QA: PASS"
