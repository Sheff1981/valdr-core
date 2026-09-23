# VALDR Core

VALDR is a standalone cryptocurrency and blockchain project. The active v0.1 implementation follows `VALDR_Master_TZ_v0.1_14_days.pdf`.

## Protocol identity

- Network / coin: **VALDR**
- Ticker: **VDR**
- Smallest unit: **val**
- `1 VDR = 100,000,000 val`
- Primary implementation language: **Go**
- Devnet chain ID: **valdr-devnet-1**
- Devnet target block time: **60 seconds**
- Address prefix: **VDR1**

## Development status

Completed milestone: **Day 4 - cryptography and CLI wallet**.

Implemented through Day 4:

- Go project skeleton;
- block model, SHA-256 block hashing, fixed Genesis and local blockchain;
- Proof of Work with nonce search, target validation and simplified devnet difficulty;
- ECDSA P-256 private/public key generation using the Go standard library;
- SHA-256 based digital signatures and signature verification;
- VALDR devnet addresses beginning with `VDR1`;
- address checksum bound to `valdr-devnet-1`;
- local wallet persistence;
- wallet integrity verification from the stored private key;
- CLI wallet create/list/export commands.

## Wallet and cryptography v0.1

The master specification fixes the requirements for private/public keys, digital signatures and the `VDR1...` address prefix, but does not prescribe a curve or final address encoding.

For the devnet MVP, VALDR v0.1 uses:

```text
signature: ECDSA P-256 over SHA-256(message)
public key: uncompressed P-256 point, hex encoded
address payload: first 20 bytes of SHA-256(public_key)
checksum: first 4 bytes of SHA-256(chain_id || payload)
address: VDR1 + Base32(payload || checksum)
```

This format is a devnet MVP parameter. Before mainnet, the cryptographic suite and address encoding must be explicitly frozen as protocol constants.

Private keys never leave the wallet code through normal create/list output and are never sent over the network. Wallet files are written with owner-only permissions on supported systems. `wallet export` intentionally reveals the private key and prints a warning.

Default wallet directory:

```text
~/.valdr/wallets
```

It can be overridden with `VALDR_WALLET_DIR` or `--dir`.

### CLI wallet

```bash
valdr-cli wallet create
valdr-cli wallet create --name alice
valdr-cli wallet list
valdr-cli wallet export alice
```

`valdr-cli wallet balance` is intentionally not functional yet. Correct balances require the UTXO Engine scheduled for Day 6; the CLI reports that dependency instead of inventing a balance.

## Proof of Work v0.1

```text
target = devnet_pow_limit / difficulty
valid block: SHA-256(block_header) <= target
```

The devnet PoW limit uses 12 leading zero bits. Difficulty targets the 60-second block interval and each adjustment is clamped to at most 4x.

### Fixed devnet Genesis parameters

- chain ID: `valdr-devnet-1`
- timestamp: `1790121600` (2026-09-23 00:00:00 UTC)
- version: `1`
- difficulty field: `1`
- nonce field: `0`
- message: `VALDR genesis block | valdr-devnet-1 | 2026-09-23`
- block hash: `47e3a6c15cab1a41c54a36a65f7133261fa6f75976a2e59825694e001716bfe5`

Not implemented yet: typed transactions, UTXO balances/spending, coinbase/mining reward, P2P synchronization, persistent blockchain storage, RPC behavior, and the final three-node devnet scenario.

## Build and test

```bash
go build ./...
go test ./...
```

## Archived Bitcoin Core experiment

The previous Bitcoin Core 31.1 / C++ experiment is not the active v0.1 architecture. It is preserved unchanged on branch `archive/bitcoin-core-experiment-0.5.0` at commit `bd32a30c698056ac601a6553e74169724a56e6ac`.

See `docs/COMPLIANCE_AUDIT.md` for the recorded divergences.
