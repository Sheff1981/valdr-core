#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

go test ./core/blockchain \
  -run '^TestV2RepeatedDeepReorgPreservesBranchesAndUTXO$' \
  -count=1

go test ./p2p \
  -run '^TestTestnet2ChainworkReorgPersistsAfterRestart$' \
  -count=1

printf 'VALDR Testnet2 reorg QA: PASS\n'
