# VALDR Stage 14A independent Testnet runbook

**Master baseline:** `docs/VALDR_Master_TZ_v0.2.14.md`  
**Network:** `testnet2` / `valdr-testnet-2`  
**Status:** ACTIVE NEXT GATE. Automated/local evidence does not complete Stage 14A.

## Required topology

Every distributed validation session uses at least three genuinely independent computers/clients:

- **A — bootstrap/public node:** reachable on TCP/17333.
- **B — independent node/miner:** initially connects to A.
- **C — independent node/client:** initially connects to A and learns B through peer exchange.

Three containers on one CI host do not satisfy this gate. RPC stays localhost-only on TCP/17332.

## Validation schedule

Run at least **three separate distributed session groups**, each approximately **2-3 hours**. Each group contains evidence from A, B and C. The normal topology therefore produces at least nine local evidence directories.

Use the exact same accepted source/build commit on all systems.

## Evidence collection

On every machine run the collector with the same `--session-group` and a unique `--machine-id`:

```bash
bash scripts/stage14a-session-validate.sh \
  --session-group s1 \
  --machine-id node-a \
  --operator operator-a \
  --source-commit EXACT_40_CHAR_COMMIT \
  --bootstrap-route PUBLIC_A:17333 \
  --node http://127.0.0.1:17332 \
  --data ./valdr-testnet2-a \
  --duration-seconds 10800 \
  --interval-seconds 60 \
  --output-dir ./stage14a-evidence
```

Repeat for B and C, then repeat the complete A/B/C session as `s2` and `s3`.

Each evidence directory contains `manifest.json`, `snapshots.jsonl`, `summary.json`, `events.jsonl` and `SHA256SUMS`. The local summary deliberately keeps `stage14a_pass: false`.

## Record operator events

Use another terminal to record important actions/results:

```bash
python3 scripts/stage14a-event.py ./stage14a-evidence/SESSION_ID peer_exchange --result pass
python3 scripts/stage14a-event.py ./stage14a-evidence/SESSION_ID mining --result pass
python3 scripts/stage14a-event.py ./stage14a-evidence/SESSION_ID transaction --result pass
python3 scripts/stage14a-event.py ./stage14a-evidence/SESSION_ID restart --result pass
python3 scripts/stage14a-event.py ./stage14a-evidence/SESSION_ID db_verify --result pass
python3 scripts/stage14a-event.py ./stage14a-evidence/SESSION_ID bootstrap_loss --result pass
```

Other supported event types: `mempool`, `desktop_sync`, `package_start`, `wallet_backup_restore`, `reorg_observation`, `crash_error`, `note`.

These entries are operator-recorded observations, not automatic proof.

## Bootstrap-loss/restart exercise

After peer exchange is confirmed:

1. Confirm C learned B.
2. Stop A.
3. Keep B running.
4. Restart C.
5. C must reconnect from learned-peer cache without A.
6. Mine a block on B.
7. C must converge to B's height/tip/chainwork.
8. Restart A.
9. A must catch up without deleting or repairing its database.

Record the actions in the evidence event log.

## Mining, transactions and product checks

Across the three session groups exercise, as applicable:

- mining from at least two independent miner instances/operators;
- block propagation and retarget observation;
- transaction creation/signing/relay/confirmation;
- mempool behavior;
- peer discovery and learned-peer cache;
- restart persistence and DB verification;
- Desktop synchronization;
- installer/package startup;
- wallet backup/restore;
- crash/error reporting.

Never share private keys merely to run validation.

## Consolidated verification

After copying all evidence directories into one review directory:

```bash
python3 scripts/stage14a-evidence-verify.py ./stage14a-evidence
```

The verifier checks SHA-256 integrity, Testnet2 identity, one exact source commit, at least three distributed session groups, at least three machine labels in every group, a recorded bootstrap route in every group, and at least one common tip hash observed by the participating machines in every group. It also reports operator-recorded scenario coverage.

A successful automated report sets `automated_evidence_ready: true` but still sets `stage14a_pass: false` and `human_review_required: true`.

## Final human exit gate

Stage 14A can be accepted only after reviewing real evidence and confirming independent machines, a genuinely reachable cold-client bootstrap route, approximately 2-3 hour sessions, peer exchange/cache recovery, mining/transactions/restart/recovery exercises, no unresolved consensus split, no manual DB repair, and no unresolved critical blocker.

Any blocker must be fixed and the full active CI rerun green before acceptance. CI/Docker/local multi-node runs are preflight evidence only.
