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

Completed milestone: **Day 6 - UTXO Engine**.

Implemented through Day 6:

- Go project skeleton;
- block model, SHA-256 block hashing, fixed Genesis and local blockchain;
- Proof of Work with nonce search, target validation and simplified devnet difficulty;
- ECDSA P-256 keys, VALDR addresses and digital signatures;
- local CLI wallet create/list/export;
- typed signed UTXO-style transactions and SHA-256 transaction IDs;
- typed transactions committed into block Merkle roots;
- UTXO set reconstruction from previously validated state;
- address balance calculation;
- UTXO ownership validation from the signing public key;
- atomic UTXO spending and output creation;
- insufficient-funds checks;
- global double-spend prevention through spent-output removal;
- atomic transaction-batch application for a block;
- blockchain validation of Merkle root and UTXO state before block acceptance.

## UTXO Engine v0.1

A UTXO is identified by:

```text
transaction_id
output_index
```

and stores:

```text
amount
recipient
```

For a normal signed transaction, the UTXO Engine performs this sequence:

1. validate transaction structure, signature and transaction ID;
2. locate every referenced UTXO;
3. derive the signer address from the transaction public key;
4. require every input UTXO to belong to that address;
5. sum input values safely;
6. sum output values safely;
7. reject insufficient input value;
8. require exact value conservation in v0.1;
9. remove spent UTXOs;
10. create one new UTXO for every transaction output.

Normal v0.1 transactions currently require:

```text
sum(inputs) == sum(outputs)
```

Transaction fees are not introduced in the 14-day MVP and are scheduled for a later protocol stage, so the Day 6 engine does not silently burn the difference as a fee.

A second attempt to spend an already consumed outpoint fails because that UTXO no longer exists.

Block-sized transaction batches are applied to a temporary UTXO state and committed only if every transaction succeeds. A failed later transaction therefore cannot leave earlier transactions partially applied.

`utxo.New(initial)` is only a state-reconstruction constructor for already validated chain state. It is not a minting API. Live VDR creation remains reserved for the coinbase/mining-reward work scheduled for Day 7.

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

## Wallet balance

The UTXO Engine can now calculate an address balance in memory. The standalone `valdr-cli wallet balance` command still needs live node/RPC state access, which is scheduled for the later RPC/CLI integration stage; it does not invent a balance from wallet files alone.

## Wallet and cryptography v0.1

```text
signature: ECDSA P-256 over SHA-256(message)
public key: uncompressed P-256 point, hex encoded
address payload: first 20 bytes of SHA-256(public_key)
checksum: first 4 bytes of SHA-256(chain_id || payload)
address: VDR1 + Base32(payload || checksum)
```

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

Not implemented yet: coinbase/mining reward, P2P synchronization, persistent blockchain storage, RPC behavior, and the final three-node devnet scenario.

## Build and test

```bash
go build ./...
go test ./...
```

## Archived Bitcoin Core experiment

The previous Bitcoin Core 31.1 / C++ experiment is not the active v0.1 architecture. It is preserved unchanged on branch `archive/bitcoin-core-experiment-0.5.0` at commit `bd32a30c698056ac601a6553e74169724a56e6ac`.

See `docs/COMPLIANCE_AUDIT.md` for the recorded divergences.
