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
- Initial devnet mining reward: **50 VDR**
- Target maximum supply: **21,000,000 VDR**
- Address prefix: **VDR1**

## Development status

Completed milestone: **Day 7 - Coinbase and mining reward**.

Implemented through Day 7:

- Go project skeleton;
- block model, SHA-256 block hashing, fixed Genesis and local blockchain;
- Proof of Work with nonce search, target validation and simplified devnet difficulty;
- ECDSA P-256 keys, canonical VALDR addresses and digital signatures;
- local CLI wallet create/list/export;
- typed signed UTXO-style transactions and SHA-256 transaction IDs;
- typed transactions committed into block Merkle roots;
- UTXO balances, ownership validation, spending and double-spend protection;
- atomic block transaction application;
- coinbase transaction validation;
- 50 VDR devnet mining reward;
- rejection of missing, duplicate or wrong-reward coinbase transactions;
- miner block assembly with coinbase at transaction index zero;
- wallet UTXO selection, change output and signed zero-fee payment creation;
- local single-chain flow: mine -> receive VDR -> send VDR -> mine next block.

## Coinbase and mining reward v0.1

The master specification requires coinbase to be the only mechanism that creates new VDR.

VALDR v0.1 keeps the existing transaction fields and encodes coinbase using a special input:

```text
transaction_id = 0000000000000000000000000000000000000000000000000000000000000000
output_index   = block height
```

Coinbase rules:

- exactly one coinbase is required per non-genesis block;
- coinbase must be transaction index zero;
- coinbase has exactly one output;
- the output recipient must be a valid `VDR1...` address;
- the output amount must equal the consensus block reward;
- coinbase has no public key and no signature;
- its transaction ID includes the height marker, making each block subsidy transaction unique;
- coinbase is accepted only with block context, never as a normal transaction;
- the newly mined reward cannot be spent inside the same block.

Current reward:

```text
50 VDR = 5,000,000,000 val
```

The master specification defines a 21,000,000 VDR limited-supply model and future halving, but does not yet freeze the halving interval. Therefore v0.1 exposes the maximum-supply parameter and keeps the initial devnet reward fixed at 50 VDR; final halving/supply enforcement must be frozen before mainnet.

## Day 7 local flow

The automated local-node core scenario now verifies:

```text
create miner wallet
    ↓
mine block #1
    ↓
coinbase credits 50 VDR
    ↓
create recipient wallet
    ↓
send 10 VDR
    ↓
mine block #2 containing payment
    ↓
recipient balance = 10 VDR
miner balance     = 90 VDR
```

The 90 VDR miner balance consists of 40 VDR change from the first reward plus the second 50 VDR coinbase reward.

This is the in-memory single-node core path required by Day 7. Persistent node state, P2P propagation and RPC/CLI network integration remain later stages in the master plan.

## UTXO Engine v0.1

A UTXO is identified by:

```text
transaction_id
output_index
```

Normal signed transactions must preserve value:

```text
sum(inputs) == sum(outputs)
```

Fees are not introduced in the 14-day MVP. Only a validated block coinbase is allowed to create new value.

A second attempt to spend an already consumed outpoint fails because the UTXO no longer exists. Block transaction batches are applied atomically.

## Transaction Engine v0.1

A normal transaction contains:

```text
version
inputs[]
outputs[]
timestamp
public_key
signature
transaction_id
```

The signature covers chain ID, version, timestamp, inputs, outputs and public key. The transaction ID is:

```text
SHA-256(canonical signed transaction bytes)
```

## Wallet and cryptography v0.1

```text
signature: ECDSA P-256 over SHA-256(message)
public key: uncompressed P-256 point, hex encoded
address payload: first 20 bytes of SHA-256(public_key)
checksum: first 4 bytes of SHA-256(chain_id || payload)
address: VDR1 + canonical Base32(payload || checksum)
```

The wallet can now construct and sign a zero-fee payment from its available UTXOs, including a change output. The standalone CLI balance/send path still needs live node/RPC state integration and is intentionally not simulated from wallet files.

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

Not implemented yet: P2P synchronization, persistent blockchain storage, RPC behavior, final halving interval, and the final three-node devnet scenario.

## Build and test

```bash
go build ./...
go test ./...
```

## Archived Bitcoin Core experiment

The previous Bitcoin Core 31.1 / C++ experiment is not the active v0.1 architecture. It is preserved unchanged on branch `archive/bitcoin-core-experiment-0.5.0` at commit `bd32a30c698056ac601a6553e74169724a56e6ac`.

Legacy files such as `consensus/valdr-consensus.json` remain historical artifacts and are not the active Go v0.1 protocol configuration.

See `docs/COMPLIANCE_AUDIT.md` for the recorded divergences.
