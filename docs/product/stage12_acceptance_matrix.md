# VALDR Stage 12 Desktop acceptance matrix

**Date:** 2026-09-25  
**Master baseline:** `docs/VALDR_Master_TZ_v0.2.5.md`  
**Branch:** `valdr-v0.2`

This file records evidence for Master-TZ §22. It does not change consensus or release scope.

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
| Node crash is surfaced and recoverable | runtime E2E kills managed `valdrd`, requires surfaced error, restarts it, and checks unchanged chain state | PASS |
| UI never displays Mainnet as available | backend state test plus `TestDesktopFrontendDoesNotOfferMainnet`; Settings network is locked to Testnet | PASS |
| No private key/passphrase appears in node RPC/P2P/logs | dedicated secret-boundary tests, redacted log tests, runtime plaintext scan | PASS |

## Remaining Stage 12 closure items

Stage 12 must **not** yet be marked fully complete.

1. **Manual GUI acceptance:** owner click-through of first-run, wallet, Send/Receive, Transactions, Settings and Advanced screens on the Windows build is still pending.
2. **Mining visual confirmation:** the previously reported blank hashrate was traced to the Desktop miner manager parsing pretty-printed JSON one line at a time. The parser now decodes the real JSON stream, and cross-platform runtime E2E requires positive accepted-block/hashrate/hash-count telemetry after a mined block. The GUI value still needs owner visual confirmation on Windows.
3. **Secret presentation hardening:** private-key output is automatically hidden when leaving the Wallet view or when the app is backgrounded, and wallet passphrase inputs are cleared after create/unlock attempts. Frontend source-contract coverage prevents silent regression.
4. If manual QA finds a defect, fix it and rerun the full CI matrix before Stage 12 is frozen.

When those items are closed, Stage 12 can be frozen and work can move to Stage 13 installers/release pipeline. Stage 13 must not begin merely because the automated matrix is green.
