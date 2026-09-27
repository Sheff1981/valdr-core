# VALDR Stage 12 Windows manual QA

**Baseline:** Master-TZ `docs/VALDR_Master_TZ_v0.2.14.md`
**Branch:** `valdr-v0.2`  
**Purpose:** optional retained human-visible checklist. By owner instruction on 2026-09-27, this is no longer a blocking v0.2 gate and has not been manually passed.

## QA candidate

Use the **current Testnet2 Windows development package from fully green VALDR v0.2 CI #531**, run `36336520832`.

- source commit: `b2704ad26659e6e6de8377013dcc33c8aa8d8ee0`;
- GitHub Actions artifact: `valdr-stage13-windows-development`, artifact ID `10937387339`;
- canonical release assembly: artifact ID `10937527319`;
- active network: `testnet2`, Chain ID `valdr-testnet-2`;
- application version: `0.2.0-dev`;
- installer: `VALDR-Desktop-0.2.0-dev-windows-x64-setup.exe`;
- installer size: `18,853,305` bytes;
- installer SHA-256: `8080562cd1025a6d8151ce6b2935477e42074861d3070731c1723e7134fc23be`;
- portable ZIP: `VALDR-Desktop-0.2.0-dev-windows-x64-portable.zip`;
- portable ZIP size: `21,980,284` bytes;
- portable ZIP SHA-256: `959dd314a7b35be6ce11448b3d34d83a3144fa91792fe993d3d9d6ff5923b114`.

These values come from the exact CI #531 Windows development artifact and its generated `release-manifest.json`. The same run's canonical release assembly passed checksum verification, repository/workflow/commit-bound GitHub/Sigstore provenance verification and the non-public release handoff. The manifest binds exact commit `b2704ad26659e6e6de8377013dcc33c8aa8d8ee0`, version `0.2.0-dev`, network `testnet2` and Chain ID `valdr-testnet-2`.

Before launch, verify the installer in PowerShell:

```powershell
Get-FileHash .\VALDR-Desktop-0.2.0-dev-windows-x64-setup.exe -Algorithm SHA256
```

Expected value:

```text
8080562cd1025a6d8151ce6b2935477e42074861d3070731c1723e7134fc23be
```

For portable ZIP:

```powershell
Get-FileHash .\VALDR-Desktop-0.2.0-dev-windows-x64-portable.zip -Algorithm SHA256
```

Expected value:

```text
959dd314a7b35be6ce11448b3d34d83a3144fa91792fe993d3d9d6ff5923b114
```

This is a **development manual-acceptance candidate**, not a public Testnet release. It is intentionally not Authenticode-signed. Mandatory authenticity for this development gate remains SHA-256 plus exact GitHub/Sigstore provenance. Use a fresh Windows profile/data directory for first-run checks. Mainnet must remain unavailable.

## 1. First run

- Start `VALDR.exe` without a terminal.
- Confirm the screen says Desktop/Testnet and Mainnet is disabled.
- Confirm the node-data directory is visible and can be changed before wallet creation.
- Confirm disk/network requirements are explained.
- Create an encrypted wallet with a passphrase.
- Confirm the local node starts only after wallet setup.
- Confirm first-run remains until local node initialization succeeds.

## 2. Dashboard

Confirm:

- spendable VDR balance;
- selected wallet name and Locked/Unlocked state;
- Testnet network identity;
- local/best height;
- sync state;
- peer count;
- latest transaction summary.

## 3. Wallet

- Lock and unlock the wallet.
- Confirm a wrong passphrase fails without damaging the wallet.
- Create an encrypted backup.
- Restore the encrypted backup.
- Open private-key export and verify the high-risk confirmation is required.
- Hide the key, leave Wallet, and verify the key is no longer visible.
- Minimize/background the app while a key is visible and verify it is cleared when returning.

## 4. Receive

- Confirm the active wallet address is displayed.
- Copy it.
- Confirm a QR code appears and is generated locally.

## 5. Send / Transactions

- Use Testnet funds only.
- Preview a transaction and verify amount, fee and total spend.
- Confirm the irreversible-transaction warning.
- Broadcast only after the final confirmation dialog.
- Verify the transaction first appears pending and later confirmed.
- Open transaction detail and check txid, timestamp, direction, amount, fee, confirmations and block data.

## 6. Advanced mode

Enable Advanced in Settings.

### Network

Confirm peers, tip, chainwork, node-data state, logs and storage diagnostics are visible. Public-node mode must remain opt-in and RPC must remain localhost-only.

### Mining

- Start mining explicitly to the selected Testnet reward address.
- Confirm Accepted blocks increases.
- Confirm Average effective hashrate is a numeric value, not `—`.
- Confirm Last block hashrate, solve time and hashes tried populate.
- Stop mining and confirm it does not restart automatically.

## 7. Settings

Confirm language, theme, managed-node startup preference, Testnet-only network field, data paths, Advanced mode and diagnostics export.

## 8. Restart/recovery

- Close and reopen VALDR Desktop.
- Confirm chain state resumes without deleting node data.
- Stop/start the managed node from the GUI.
- If a node error is shown, confirm Restart node recovers it.

## Acceptance result

Under Master-TZ v0.2.14 this checklist is optional for v0.2 and Stage 12 is frozen using automated evidence plus the explicit owner waiver. If this checklist is later executed, any failure must still be recorded and fixed before any affected release is accepted.

Record the manual result with:

- Windows version/build;
- PASS/FAIL for sections 1–8;
- failed item text and screenshot/log reference if any;
- tester date/time.

A manual PASS closes only the Stage 12 human-visible gate. Stage 13 final RC, website CI and final release verification remain separate gates.
