#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export VALDR_NETWORK="${VALDR_NETWORK:-testnet2}"
export VALDR_EXPECTED_CHAIN_ID="${VALDR_EXPECTED_CHAIN_ID:-valdr-testnet-2}"

exec "$ROOT/scripts/start-devnet.sh" smoke
