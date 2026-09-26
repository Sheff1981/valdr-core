# VALDR Stage 12 Desktop acceptance matrix

**Date:** 2026-09-26  
**Master baseline:** `docs/VALDR_Master_TZ_v0.2.11.md`  
**Branch:** `valdr-v0.2`

This file records Stage 12 evidence under the cumulative Master-TZ v0.2.11. It does not change consensus or release scope.

## Testnet2 R0 automated baseline

- Active profile: `testnet2`; Chain ID: `valdr-testnet-2`.
- Exact CI-verified source commit: `ed9b1425847f4e391a678cca3cf87d3acf0d866c`.
- VALDR v0.2 CI run `36217224372` (#367): **SUCCESS** on 2026-09-26.
- Windows x64, Linux x64, macOS ARM64 and macOS AMD64 Desktop runtime jobs passed.
- Live Desktop wallet send/receive/history, clean-launch/restart, managed node/miner bundling and Testnet2 identity checks passed.
- This is automated evidence only. The separate Windows owner GUI checklist remains manually unverified.

**Stage 12A automated exit gate: PASS on the Testnet2 CI commit above.** The required cross-platform runtime jobs, first-run, send/receive/history, restart persistence and crash recovery have automated evidence. Stage 12B usability work and Stage 12C human-visible Windows acceptance remain open. The Windows checklist now points to the Testnet2 artifact from this exact run; its former Testnet1 candidate must not be reused.

## Testnet2 P2P v2 relay / reorg runtime gate

- P2P relay defect fixed in commit `5acae7e9f1604050f2559483b74a512914b71295`: transactions received from a peer are now forwarded to other peers using the active P2P v2 frame instead of the legacy transaction frame.
- CI gate commit: `d30acd3bc1bede1c29ff486e9ce96e210da40c6b`.
- [VALDR v0.2 CI run #378](https://github.com/Sheff1981/valdr-core/actions/runs/36227344833): **SUCCESS** on 2026-09-26.
- The mandatory Testnet2 runtime gate covers three-node multi-hop transaction relay, block propagation, greatest-chainwork reorganization, side-branch retention, persistent Badger state and restart recovery.
- Docker Compose Testnet2 smoke, Linux service smoke and three-node runtime smoke also passed in the same full CI run.
- This remains controlled CI/runtime evidence; it is not Stage 14A independent-computer soak evidence.

## Additional Testnet2 Core runtime verification

- Exact source commit: `1b250679e39df2c291dcd54eb8588d5d189e6fa1`.
- [VALDR v0.2 CI run #368](https://github.com/Sheff1981/valdr-core/actions/runs/36222702418): **SUCCESS** on 2026-09-26; all eight jobs passed.
- Docker Testnet smoke now mines on node1, checks all three nodes agree on the same height-1 tip, mines on node2, checks the height-2 tip across all three, verifies node3's stopped Badger database, then restarts node3 and checks convergence again.
- This is local three-container verification. It does not satisfy the later independent-operator or seven-day distributed Testnet gates.


| §22 acceptance requirement | Automated evidence | Status |
| --- | --- | --- |
| Desktop builds on all mandatory target OSes | CI jobs `desktop-linux`, `desktop-windows-amd64`, `desktop-macos-arm64`, `desktop-macos-amd64` | PASS |
| App starts without terminal use | native clean-launch Desktop smoke on Linux, Windows and both macOS architectures; fresh first-run also asserts that valdrd is deferred until wallet setup | PASS |
| First-run creates encrypted wallet | Desktop encrypted-wallet tests and runtime E2E now enforce the §16.3 order: fresh Desktop has no running node, wallet is created/unlocked, then managed valdrd starts; final visual click-through still requires owner QA | AUTOMATED PASS / MANUAL QA PENDING |
| Managed node starts and stops correctly | runtime E2E, managed-stdin shutdown smoke, clean-launch shutdown checks | PASS |
| Outbound-only Testnet sync works | `scripts/desktop-outbound-smoke.sh` now mines on a separate Testnet peer and requires Desktop height/tip convergence while its inbound P2P port remains closed | PASS |
| Restart resumes without DB deletion | clean-launch restart smoke verifies identical persisted height and tip | PASS |
| Balance/send/receive/transaction history work | `TestDesktopRuntimeWalletSendReceiveHistory` on Windows/macOS/Linux runtime jobs | PASS |
| Wallet backup and restore work | runtime E2E encrypted backup, wrong-passphrase rejection, restore preserving address/unlock | PASS |
| Wrong password fails safely | wallet unlock and backup-restore negative tests | PASS |
| Configurable inactivity auto-lock | deterministic wallet-session tests cover activity refresh, timeout expiry, timeout changes and in-memory secret zeroization on replacement/lock | PASS |
| Node crash is surfaced and recoverable | runtime E2E kills managed `valdrd`, requires surfaced error, restarts it, and checks unchanged chain state | PASS |
| UI never displays Mainnet as available | backend state test plus `TestDesktopFrontendDoesNotOfferMainnet`; Settings network is locked to Testnet | PASS |
| No private key/passphrase appears in node RPC/P2P/logs | dedicated secret-boundary tests, redacted log tests, runtime plaintext scan | PASS |

## Remaining Stage 12 closure items

Stage 12 must **not** yet be marked fully complete.

1. **Manual GUI acceptance:** owner click-through of first-run, wallet, Send/Receive, Transactions, Settings and Advanced screens on the Windows build is still pending. The exact procedure is frozen in `docs/product/stage12_windows_manual_qa.md`.
2. **Mining visual confirmation:** the previously reported blank hashrate was traced to the Desktop miner manager parsing pretty-printed JSON one line at a time. The parser now decodes the real JSON stream, and cross-platform runtime E2E requires positive accepted-block/hashrate/hash-count telemetry after a mined block. The GUI value still needs owner visual confirmation on Windows.
3. **Secret presentation hardening:** private-key output is automatically hidden when leaving the Wallet view or when the app is backgrounded, and wallet passphrase inputs are cleared after create/unlock attempts. Frontend source-contract coverage prevents silent regression.
4. If manual QA finds a defect, fix it and rerun the full CI matrix before Stage 12 is frozen.

Stage 13 development packaging/provenance was regenerated on Testnet2 in CI #367, but public release acceptance remains blocked by Stage 12 manual Windows QA and the later release gates. The current project-owner priority is Core/Testnet2 network verification before further UX/site expansion; this does not mark Stage 12 manually complete or authorize a public release.

## Public positioning consistency

The Desktop UI uses the same standalone public positioning as the official website: `INDEPENDENT CHAIN. NATIVE VDR.` Public product copy describes VALDR through its own properties rather than competitor comparisons. This is a presentation-only change and does not alter protocol, consensus, network identity or release gates.
