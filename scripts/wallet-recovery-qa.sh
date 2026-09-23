#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DAEMON="${VALDR_TESTNET_DAEMON:-$ROOT/release/bin/valdrd}"
CLI="${VALDR_TESTNET_CLI:-$ROOT/release/bin/valdr-cli}"
[[ -x "$DAEMON" && -x "$CLI" ]] || { echo 'Compiled valdrd/valdr-cli required.' >&2; exit 1; }

D="${VALDR_QA_ROOT:-$ROOT/work/wallet-recovery}"
rm -rf "$D"
mkdir -p "$D/backups"
PORT=38441
RPC=38451

cleanup(){ "$CLI" -testnet4 -datadir="$D/node" -rpcport=$RPC stop >/dev/null 2>&1 || true; }
trap cleanup EXIT

"$DAEMON" -testnet4 -datadir="$D/node" -port=$PORT -rpcport=$RPC -server=1 -listen=0 -dnsseed=0 -fixedseeds=0 -fallbackfee=0.00001 -daemonwait
C(){ "$CLI" -testnet4 -datadir="$D/node" -rpcport=$RPC "$@"; }

C createwallet original >/dev/null
ADDR=$(C -rpcwallet=original getnewaddress)
C -rpcwallet=original generatetoaddress 101 "$ADDR" >/dev/null
BAL=$(C -rpcwallet=original getbalance)
C -rpcwallet=original backupwallet "$D/backups/original.dat" >/dev/null
sha256sum "$D/backups/original.dat" > "$D/backups/original.dat.sha256"

C unloadwallet original >/dev/null
C restorewallet restored "$D/backups/original.dat" false >/dev/null
RBAL=$(C -rpcwallet=restored getbalance)

python3 - "$BAL" "$RBAL" <<'PY'
from decimal import Decimal
import sys
assert Decimal(sys.argv[1]) == Decimal(sys.argv[2]), (sys.argv[1], sys.argv[2])
PY

DEST=$(C -rpcwallet=restored getnewaddress)
TX=$(C -rpcwallet=restored sendtoaddress "$DEST" 1.0)
C -rpcwallet=restored generatetoaddress 1 "$DEST" >/dev/null
CONF=$(C -rpcwallet=restored gettransaction "$TX" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("confirmations",0))')
(( CONF >= 1 )) || { echo 'restored wallet spend did not confirm' >&2; exit 1; }
sha256sum -c "$D/backups/original.dat.sha256" >/dev/null
echo "VALDR wallet recovery QA: PASS"
