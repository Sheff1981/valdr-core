# VALDR Stage 14A independent Testnet runbook

**Master baseline:** `docs/VALDR_Master_TZ_v0.2.15.md`  
**Network:** `testnet2` / `valdr-testnet-2`  
**Status:** ACTIVE NEXT GATE. Automated/local evidence does not complete Stage 14A.

## Required topology

Every distributed validation session uses two genuinely independent computers/clients:

- **A — bootstrap/listening node:** reachable by B on TCP/17333.
- **B — independent node/miner/client:** starts cold and connects to A using the real VALDR P2P v2 protocol.

Two containers on one CI host do not satisfy this gate. RPC stays localhost-only on TCP/17332. A third independent node is useful optional evidence, but it is not a v0.2 Stage 14A blocker.

## Validation schedule

Run at least **three separate distributed session groups**, each approximately **2-3 hours**. Each group contains evidence from A and B. The normal topology therefore produces at least six local evidence directories.

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

Repeat for B, then repeat the complete A/B session as `s2` and `s3`.

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

## Disconnect/restart/reconnect exercise

After the VALDR protocol handshake and chain convergence are confirmed:

1. Confirm A and B show each other as connected VALDR peers.
2. Stop A while B remains running.
3. Optionally mine a block on B while A is offline.
4. Restart A.
5. B must reconnect to A through the configured/remembered reachable route.
6. A must converge to B's height/tip/chainwork without deleting or repairing its database.
7. Stop and restart B.
8. B must reconnect to A and converge again.
9. Record any peer-cache behavior that is actually observable; do not claim alternate-third-peer recovery from a two-computer test.

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

The verifier checks SHA-256 integrity, Testnet2 identity, one exact source commit, at least three distributed session groups, at least two machine labels in every group, a recorded bootstrap route in every group, and at least one common tip hash observed by the participating machines in every group. It also reports operator-recorded scenario coverage.

A successful automated report sets `automated_evidence_ready: true` but still sets `stage14a_pass: false` and `human_review_required: true`.

## Final human exit gate

Stage 14A can be accepted only after reviewing real evidence and confirming two independent computers, a genuinely reachable cold-client route on TCP/17333, approximately 2-3 hour sessions, protocol handshake/reconnect, mining/transactions/restart/recovery exercises, no unresolved consensus split, no manual DB repair, and no unresolved critical blocker.

Any blocker must be fixed and the full active CI rerun green before acceptance. CI/Docker/local multi-node runs are preflight evidence only.
