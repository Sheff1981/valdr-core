# VALDR v0.1 compliance audit

Master specification: `VALDR_Master_TZ_v0.1_14_days.pdf`, dated 23 September 2026.

## Archived experimental line

The previous Bitcoin Core/C++ experiment remains preserved unchanged on:

`archive/bitcoin-core-experiment-0.5.0`

Archive point:

`bd32a30c698056ac601a6553e74169724a56e6ac`

It is not the active VALDR v0.1 architecture.

## Active v0.1 disposition

| Area | Master specification v0.1 | Active VALDR v0.1 |
| --- | --- | --- |
| Implementation | Own VALDR blockchain, Go primary | Go standalone implementation |
| Coin / ticker | VALDR / VDR | VALDR / VDR |
| Atomic unit | 1 VDR = 100,000,000 val | Implemented |
| Chain ID | valdr-devnet-1 | Implemented |
| Block / Genesis | Fixed Genesis + local chain | Implemented and tested |
| Proof of Work | SHA-256 MVP, nonce/target/difficulty | Implemented and tested |
| Wallet | private/public key, VDR address, signatures | Implemented and tested |
| Transactions | UTXO inputs/outputs/signatures/txid | Implemented and tested |
| UTXO | balance/spend/create/double-spend protection | Implemented and tested |
| Coinbase | initial 50 VDR reward | Implemented and tested |
| P2P | peers, blocks, transactions, missing blocks, sync | Implemented basic v0.1 sync |
| Mempool | valid unconfirmed transactions | Implemented and tested |
| Three nodes | same confirmed state | Implemented and tested |
| RPC / CLI | status, block, tx, balance, send, peers, mining | Implemented |
| Miner | start/status/stop | Implemented; node executes PoW through mineBlock RPC |
| Storage | blockchain must survive restart | Implemented and final integration-tested |
| Security checks | required MVP invalid-data checks | Implemented and regression-tested |
| Logging | NODE/P2P/BLOCK/TX/MINER/MEMPOOL/SYNC/ERROR | Implemented category prefixes |
| Day 13 | start-devnet.sh | Implemented and CI smoke-tested |
| Day 14 | mine/send/mine/sync/final balance | Implemented as executable final integration test |

## Storage implementation note

The master specification identifies LevelDB or BadgerDB as preferred embedded storage choices.

VALDR Devnet v0.1 uses a dependency-free atomic on-disk snapshot at:

`<data>/blockchain.json`

The snapshot is written through a temporary `0600` file, synchronized, and atomically renamed. On node startup, persisted blocks are replayed through the normal consensus and UTXO validation pipeline.

This is a deliberate implementation choice relative to the specification's preference, not a change to blockchain architecture or protocol formats.

Benefits:

- no third-party database dependency in the first devnet;
- simple clean-clone build;
- deterministic full-chain validation during restart;
- atomic replacement of the confirmed snapshot.

Risks / limitations:

- O(chain size) rewrite for each confirmed block;
- unsuitable for a large public chain;
- a LevelDB/Badger-style indexed backend should replace it before scale testing.

The master specification does not need a version bump for this v0.1 choice because LevelDB/BadgerDB are listed as preferred options rather than mandatory wire/consensus requirements.

## Parameters intentionally not frozen by the master specification

The exact halving interval is explicitly deferred until block-speed and economic testing. VALDR v0.1 therefore keeps the initial 50 VDR devnet reward without inventing a new halving schedule.

A full fork-choice / chain reorganization policy is also not introduced by the 14-day MVP. Basic synchronization rejects conflicting confirmed history instead of silently replacing it.

These are future protocol-specification items and are not represented as completed mainnet behavior.

## VALDR Devnet v0.1 readiness gate

The repository readiness gate is:

```bash
go build ./...
go test ./...
bash -n ./scripts/start-devnet.sh
bash -n ./scripts/test-devnet-v0.1.sh
./scripts/start-devnet.sh smoke
./scripts/test-devnet-v0.1.sh
```

The final integration script proves:

```text
3 nodes online
wallet A + wallet B
mine 50 VDR to A
A sends 10 VDR to B
transaction propagates
mine confirming block
all nodes synchronize
same balances on all nodes
stop all nodes
restart all nodes
same height / tip / transaction / balances remain
```
