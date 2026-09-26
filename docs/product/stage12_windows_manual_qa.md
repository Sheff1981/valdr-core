# VALDR Stage 12 Windows manual QA

**Baseline:** Master-TZ `docs/VALDR_Master_TZ_v0.2.11.md`
**Branch:** `valdr-v0.2`  
**Purpose:** final human-visible acceptance before Stage 12 is frozen.

## QA candidate

Use the Testnet2 Windows development package from the fully green [VALDR v0.2 CI run #367](https://github.com/Sheff1981/valdr-core/actions/runs/36217224372):

- source commit: `ed9b1425847f4e391a678cca3cf87d3acf0d866c`;
- GitHub Actions artifact: `valdr-stage13-windows-development` from **that exact run**;
- active network: `testnet2`, Chain ID `valdr-testnet-2`;
- application version: `0.2.0-dev`;
- installer: `VALDR-Desktop-0.2.0-dev-windows-x64-setup.exe`;
- portable ZIP: `VALDR-Desktop-0.2.0-dev-windows-x64-portable.zip`.

Before launch, verify the installer in PowerShell:

```powershell
Get-FileHash .\VALDR-Desktop-0.2.0-dev-windows-x64-setup.exe -Algorithm SHA256
```

Compare the value with the installer entry in the **same run's** canonical `SHA256SUMS` / `release-manifest.json` from `valdr-stage13-release-assembly-development`. Confirm the manifest identifies the exact source commit and `valdr-testnet-2`. Do not use checksums or artifacts from run #356: those belong to Testnet1. This is a development acceptance candidate, not a public Testnet release. It is intentionally not Authenticode-signed; Master-TZ v0.2.11 uses SHA-256 + GitHub/Sigstore provenance as the mandatory Testnet authenticity model.

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
