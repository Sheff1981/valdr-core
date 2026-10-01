# VALDR Stage 14A independent Testnet runbook

**Master baseline:** `docs/VALDR_Master_TZ_v0.2.27.md`  
**Network:** `testnet2` / `valdr-testnet-2`  
**Status:** ACTIVE NEXT GATE. Automated/local evidence does not complete Stage 14A.

## Required topology

Use two genuinely independent computers/clients:

- **A — reachable/bootstrap candidate:** VALDR node listening on TCP/17333.
- **B — cold ordinary client:** separate machine and separate VALDR data directory.

Two containers or two processes on one computer do not satisfy this gate. RPC remains localhost-only on TCP/17332.

A third independent node is useful optional Stage 14B/14C evidence, but is not a v0.2 Stage 14A blocker.

## Gate model

Stage 14A is **functional and evidence-based, not time-based**.

There is:

- no mandatory 2-3 hour session;
- no mandatory three-session quota;
- no 24-hour powered-on requirement;
- no seven-day powered-on requirement.

One complete distributed evidence group is sufficient when it contains both independent computers and all mandatory functional checks.

Use the exact same accepted source/build commit on both systems.

## Step 1 — reachability preflight

Before claiming automatic bootstrap, prove that B can reach A with the real VALDR P2P v2 protocol:

```text
valdrd.exe probe-peer --network testnet2 --address REAL_A_HOST_OR_IP:17333
```

A successful probe proves:

- TCP/17333 is reachable from B;
- the remote endpoint speaks VALDR P2P v2;
- Testnet2 network identity and protocol handshake succeed.

It does **not** by itself prove ordinary-user automatic bootstrap. Manual address entry here is operator preflight only.

## Step 2 — real bootstrap configuration

Only after A has a genuinely reachable stable route may that real endpoint be added to the active Testnet2 bootstrap configuration.

Rules:

- no placeholder or invented seed;
- no private/loopback address in the public compiled list;
- ordinary Desktop onboarding must not require the user to type the route;
- remembered peers and peer exchange remain normal reconnect/discovery mechanisms after the first contact.

## Step 3 — distributed functional sequence

Run the following on A and B:

1. Fresh install/launch on both independent computers.
2. Start B cold with no manual peer/seed/IP/port entry.
3. Confirm B automatically joins Testnet2 and A/B expose each other as VALDR peers.
4. Confirm height, tip hash and cumulative chainwork converge.
5. With mining disabled, create/send a transaction.
6. Confirm the receiving node observes the same txid as pending/0-confirmation.
7. Enable mining on **one** machine only.
8. Confirm the block propagates and both machines show the transaction confirmed.
9. Restart A and B.
10. Confirm automatic reconnect and reconvergence without deleting or repairing either database.
11. Confirm learned-peer cache survives restart.
12. Run DB verification.
13. Exercise bootstrap loss/recovery where the available topology permits it.
14. Verify installer upgrade preserves wallet and node data.
15. Keep the existing conflict/reorg recovery gates green.

## Evidence collection

On each machine run the collector with the same `--session-group` and unique `--machine-id`.

A short observation window is acceptable; choose only enough time to capture the functional sequence.

Example:

```bash
bash scripts/stage14a-session-validate.sh \
  --session-group final-ab \
  --machine-id node-a \
  --operator operator-a \
  --source-commit EXACT_40_CHAR_COMMIT \
  --bootstrap-route REAL_A_HOST_OR_IP:17333 \
  --node http://127.0.0.1:17332 \
  --data ./valdr-testnet2-a \
  --duration-seconds 300 \
  --interval-seconds 15 \
  --output-dir ./stage14a-evidence
```

Repeat on B using the same `--session-group final-ab`.

Each evidence directory contains `manifest.json`, `snapshots.jsonl`, `summary.json`, `events.jsonl` and `SHA256SUMS`. Local summaries deliberately keep `stage14a_pass: false`.

## Record operator events

Record the observed checks in the corresponding evidence directory:

```bash
python3 scripts/stage14a-event.py ./stage14a-evidence/SESSION_ID automatic_join --result pass
python3 scripts/stage14a-event.py ./stage14a-evidence/SESSION_ID peer_handshake --result pass
python3 scripts/stage14a-event.py ./stage14a-evidence/SESSION_ID peer_exchange --result pass
python3 scripts/stage14a-event.py ./stage14a-evidence/SESSION_ID transaction --result pass
python3 scripts/stage14a-event.py ./stage14a-evidence/SESSION_ID pending_relay --result pass
python3 scripts/stage14a-event.py ./stage14a-evidence/SESSION_ID mining --result pass
python3 scripts/stage14a-event.py ./stage14a-evidence/SESSION_ID block_propagation --result pass
python3 scripts/stage14a-event.py ./stage14a-evidence/SESSION_ID restart --result pass
python3 scripts/stage14a-event.py ./stage14a-evidence/SESSION_ID peer_cache_reconnect --result pass
python3 scripts/stage14a-event.py ./stage14a-evidence/SESSION_ID db_verify --result pass
python3 scripts/stage14a-event.py ./stage14a-evidence/SESSION_ID installer_upgrade --result pass
```

Use `bootstrap_loss` when that scenario is actually exercised. Other supported evidence types include `mempool`, `desktop_sync`, `package_start`, `wallet_backup_restore`, `reorg_observation`, `crash_error` and `note`.

These are operator-recorded observations and remain subject to human review.

## Consolidated verification

After copying both machine evidence directories into one review directory:

```bash
python3 scripts/stage14a-evidence-verify.py ./stage14a-evidence
```

The verifier checks:

- SHA-256 integrity;
- Testnet2 identity;
- one exact source commit;
- at least one distributed evidence group;
- at least two independent machine labels in every group;
- a recorded real bootstrap route;
- common observed tip and final height/tip/chainwork convergence;
- no operator-recorded failure;
- mandatory functional scenario coverage;
- mining evidence from at least one machine.

A successful automated report sets `automated_evidence_ready: true` but still keeps `stage14a_pass: false` and `human_review_required: true`.

## Final human exit gate

Stage 14A can be accepted only after reviewing the real evidence and confirming:

- the machine labels represent two genuinely independent computers;
- the bootstrap route was genuinely reachable;
- cold B joined without manual peer/seed/IP/port entry;
- transaction relay, one-machine mining and block confirmation worked;
- restart/reconnect, peer-cache persistence, DB verification and installer-upgrade preservation worked;
- no unresolved consensus split, manual DB repair or critical blocker remains.

Any blocker is fixed first, then the full active CI must be green before acceptance.
