#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
python3 "$ROOT/tools/genesis.py"
python3 "$ROOT/tools/genesis_testnet.py"
python3 "$ROOT/tools/address_vectors.py"
python3 "$ROOT/tools/supply.py"
python3 "$ROOT/tools/asert_reference.py"
python3 "$ROOT/tools/security_sim.py"
python3 "$ROOT/tools/reorg_deposit_model.py"
python3 "$ROOT/tests/test_consensus.py"
python3 "$ROOT/tests/test_reorg_deposit_model.py"
python3 -m py_compile "$ROOT"/tools/*.py "$ROOT"/tests/*.py
echo "VALDR deterministic QA: PASS"
