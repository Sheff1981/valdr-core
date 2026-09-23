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

Completed milestone: **Day 5 - Transaction Engine**.

Implemented through Day 5:

- Go project skeleton;
- block model, SHA-256 block hashing, fixed Genesis and local blockchain;
- Proof of Work with nonce search, target validation and simplified devnet difficulty;
- ECDSA P-256 private/public key generation and digital signatures;
- VALDR devnet addresses beginning with `VDR1`;
- local CLI wallet create/list/export;
- typed UTXO-style transaction inputs and outputs;
- canonical transaction signing bytes;
- transaction-level digital signature verification;
- SHA-256 transaction IDs;
- typed transactions committed into block Merkle roots;
- Day 5 transaction structure validation.

## Transaction Engine v0.1

The master specification defines the transaction fields as `version`, `inputs[]`, `outputs[]`, `timestamp`, `signature` and `transaction_id`.

VALDR v0.1 represents an input as:

```text
transaction_id
output_index
```

and an output as:

```text
amount       # uint64 atomic units (val)
recipient    # VDR1... address
```

A normal Day 5 transaction has one transaction-level public key and signature. The signature covers the chain ID, version, timestamp, every input, every output and the public key. The transaction ID is:

```text
SHA-256(canonical signed transaction bytes)
```

The chain ID is included in signed bytes to prevent the same signature from being replayed unchanged across a future network with a different chain ID.

Day 5 validation checks:

- transaction version;
- non-empty inputs and outputs;
- syntactically valid previous transaction IDs;
- no duplicate input reference inside the same transaction;
- output amount greater than zero;
- amount overflow;
- valid `VDR1...` recipient;
- valid public key;
- valid digital signature;
- correct transaction ID.

The following checks are **not implemented on Day 5** because the master plan assigns them to the UTXO Engine on Day 6:

- whether the referenced UTXO actually exists;
- whether that UTXO belongs to the signing public key;
- sufficient input value / balance;
- creating and spending UTXOs;
- global double-spend protection.

## Wallet and cryptography v0.1

For the devnet MVP:

```text
signature: ECDSA P-256 over SHA-256(message)
public key: uncompressed P-256 point, hex encoded
address payload: first 20 bytes of SHA-256(public_key)
checksum: first 4 bytes of SHA-256(chain_id || payload)
address: VDR1 + Base32(payload || checksum)
```

This format remains a devnet MVP parameter. Before mainnet, the cryptographic suite and address encoding must be explicitly frozen as protocol constants.

Private keys never leave the wallet code through normal create/list output and are never sent over the network. `wallet export` intentionally reveals the private key and prints a warning.

`valdr-cli wallet balance` remains intentionally unavailable until the UTXO Engine exists.

## Proof of Work v0.1

```text
target = devnet_pow_limit / difficulty
valid block: SHA-256(block_header) <= target
```

The devnet PoW limit uses 12 leading zero bits. Difficulty targets the 60-second block interval and each adjustment is clamped to at most 4x.

### Fixed devnet Genesis

- chain ID: `valdr-devnet-1`
- timestamp: `1790121600` (2026-09-23 00:00:00 UTC)
- version: `1`
- difficulty field: `1`
- nonce field: `0`
- message: `VALDR genesis block | valdr-devnet-1 | 2026-09-23`
- block hash: `47e3a6c15cab1a41c54a36a65f7133261fa6f75976a2e59825694e001716bfe5`

Not implemented yet: UTXO balances/spending, coinbase/mining reward, P2P synchronization, persistent blockchain storage, RPC behavior, and the final three-node devnet scenario.

## Build and test

```bash
go build ./...
go test ./...
```

## Archived Bitcoin Core experiment

The previous Bitcoin Core 31.1 / C++ experiment is not the active v0.1 architecture. It is preserved unchanged on branch `archive/bitcoin-core-experiment-0.5.0` at commit `bd32a30c698056ac601a6553e74169724a56e6ac`.

See `docs/COMPLIANCE_AUDIT.md` for the recorded divergences.
