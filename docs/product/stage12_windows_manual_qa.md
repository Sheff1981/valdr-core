# VALDR Stage 12 Windows manual QA

> **Testnet2 reset:** the CI #356 / `valdr-testnet-1` package below is retained as historical Windows evidence only. Master-TZ v0.2.9 changes consensus to `testnet2` / `valdr-testnet-2`; final Stage 12 acceptance must be repeated against a new CI-green Testnet2 Windows package. Do not mark Stage 12 complete from the old candidate.

**Baseline:** Master-TZ `docs/VALDR_Master_TZ_v0.2.8.md`  
**Branch:** `valdr-v0.2`  
**Purpose:** final human-visible acceptance before Stage 12 is frozen.

## QA candidate

Use the CI-green Windows development package built from:

- source commit: `ea74320f9950dad733e4f95c0d24e9b008e2d221`;
- VALDR v0.2 CI run: `36183171614` (#356) — SUCCESS;
- GitHub Actions artifact: `valdr-stage13-windows-development` (artifact ID `10885735468`);
- application version: `0.2.0-dev`;
- installer: `VALDR-Desktop-0.2.0-dev-windows-x64-setup.exe`;
- installer SHA-256: `f20d90211ab5e4e19633ae7563652bcdbfb9f04851f53a18031e705a5a68c365`;
- portable ZIP: `VALDR-Desktop-0.2.0-dev-windows-x64-portable.zip`;
- portable SHA-256: `6cd8b93c14e4ecb3233161f20e581b1cf195ed8c7b3c5b9024d19b8c257cfdba`;
- release manifest SHA-256: `c1e3b213eebbd4b5e8f4d4c9e5b51e1c9447dc74b929b8a622eca518b3bf9ef2`.

Before launch, verify the installer in PowerShell:

```powershell
Get-FileHash .\VALDR-Desktop-0.2.0-dev-windows-x64-setup.exe -Algorithm SHA256
```

The value must equal the installer SHA-256 above. The values were independently recomputed from the downloaded CI artifact and match its embedded `release-manifest.json`. This package is a development acceptance candidate, not a public Testnet release. It is intentionally not Authenticode-signed; Master-TZ v0.2.8 uses SHA-256 + GitHub/Sigstore provenance as the mandatory Testnet authenticity model.

Use a fresh Windows profile/data directory for first-run checks. Mainnet must remain unavailable.

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

Stage 12 may be frozen only after this checklist passes on the Windows build. Record any failed item before freezing the Testnet release candidate.

Record the manual result with:

- Windows version/build;
- PASS/FAIL for sections 1–8;
- failed item text and screenshot/log reference if any;
- tester date/time.

A manual PASS closes only the Stage 12 human-visible gate. Stage 13 final RC, website CI and final release verification remain separate gates.
