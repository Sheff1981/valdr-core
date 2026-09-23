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

## Development status

Completed milestone: **Day 3 - Proof of Work**.

Implemented:

- Day 1 Go project skeleton and package structure;
- Day 2 `Block`, single SHA-256 hashing, fixed Genesis Block and local blockchain;
- nonce search over the block header;
- 256-bit PoW target calculation;
- integer difficulty multiplier;
- PoW verification before a block is accepted;
- simplified v0.1 difficulty adjustment around the 60-second block target;
- local mined chain: `Genesis -> Block 1 -> Block 2`.

## Proof of Work v0.1

The master specification allows a simplified difficulty algorithm for the first devnet. The current v0.1 rule is:

```text
target = devnet_pow_limit / difficulty
valid block: SHA-256(block_header) <= target
```

The devnet PoW limit uses 12 leading zero bits. Difficulty `1` therefore still performs a real nonce search while keeping automated tests fast.

For the next block:

```text
next_difficulty ~= previous_difficulty * 60 / observed_block_interval
```

The observed interval is clamped so one block can change difficulty by at most 4x. Difficulty never falls below `1`. This is intentionally an MVP rule; the master specification already schedules an improved difficulty adjustment for VALDR v0.2.

The fixed Genesis Block is the hard-coded trust anchor and is not re-mined during startup. PoW validation applies to subsequently appended blocks.

### Fixed devnet Genesis parameters

- chain ID: `valdr-devnet-1`
- timestamp: `1790121600` (2026-09-23 00:00:00 UTC)
- version: `1`
- difficulty field: `1`
- nonce field: `0`
- message: `VALDR genesis block | valdr-devnet-1 | 2026-09-23`
- block hash: `47e3a6c15cab1a41c54a36a65f7133261fa6f75976a2e59825694e001716bfe5`

Transactions are still opaque placeholder strings. The typed Transaction Engine is scheduled for Day 5.

Not implemented yet: wallet cryptography, typed transactions, UTXO engine, coinbase/mining reward, P2P synchronization, persistent blockchain storage, RPC behavior, and the final three-node devnet scenario.

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
