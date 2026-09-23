# VALDR Core

VALDR is a standalone cryptocurrency and blockchain project. The active v0.1 implementation follows `VALDR_Master_TZ_v0.1_14_days.pdf`.

## Protocol identity

- Network / coin: **VALDR**
- Ticker: **VDR**
- Smallest unit: **val**
- `1 VDR = 100,000,000 val`
- Primary implementation language: **Go**
- Devnet chain ID: **valdr-devnet-1**

## Development status

Completed milestone: **Day 2 - blockchain**.

Implemented:

- Day 1 Go project skeleton and package structure;
- VALDR `Block` model with the master-specification fields;
- canonical block-header serialization;
- single SHA-256 block hashing for the MVP;
- deterministic Merkle root calculation for the Day 2 transaction placeholder data;
- fixed VALDR devnet Genesis Block;
- local in-memory blockchain with height, previous-hash, chain-ID and block-hash validation;
- tested local chain: `Genesis -> Block 1 -> Block 2`.

### Fixed devnet Genesis parameters

- chain ID: `valdr-devnet-1`
- timestamp: `1790121600` (2026-09-23 00:00:00 UTC)
- version: `1`
- difficulty placeholder: `1`
- nonce placeholder: `0`
- message: `VALDR genesis block | valdr-devnet-1 | 2026-09-23`
- block hash: `47e3a6c15cab1a41c54a36a65f7133261fa6f75976a2e59825694e001716bfe5`

The difficulty and nonce fields exist in the block format, but Proof-of-Work target calculation, mining and PoW validation are **not implemented yet**. They belong to Day 3.

Transactions are intentionally opaque placeholder strings in Day 2. The typed Transaction Engine is scheduled for Day 5.

Not implemented yet: Proof of Work, wallet cryptography, typed transactions, UTXO engine, mining rewards, P2P synchronization, persistent blockchain storage, RPC behavior, and the final three-node devnet scenario.

## Build and test

```bash
go build ./...
go test ./...
```

## Project layout

```text
cmd/
  valdrd/
  valdr-cli/
  valdr-miner/
core/
  block/
  blockchain/
  transaction/
  utxo/
  consensus/
crypto/
wallet/
p2p/
mining/
storage/
rpc/
config/
tests/
docs/
scripts/
go.mod
```

## Archived Bitcoin Core experiment

The previous Bitcoin Core 31.1 / C++ experiment is not the active v0.1 architecture. It is preserved unchanged on branch `archive/bitcoin-core-experiment-0.5.0` at commit `bd32a30c698056ac601a6553e74169724a56e6ac` for later review. Existing legacy files still present on this branch are historical artifacts and are not evidence that the corresponding Go v0.1 stage has been implemented.

See `docs/COMPLIANCE_AUDIT.md` for the recorded divergences.
