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
- P2P protocol version: **1**
- Default P2P port: **7333**
- Default RPC port: **7332**

## Development status

Completed milestone: **Day 8 - P2P Node A ↔ Node B**.

Implemented through Day 8:

- block model, fixed Genesis and local blockchain;
- Proof of Work with nonce search, target validation and simplified difficulty;
- ECDSA P-256 keys, canonical VALDR addresses and digital signatures;
- local wallet creation and signed payments;
- typed UTXO transactions and transaction IDs;
- UTXO balances, spending and double-spend protection;
- coinbase and 50 VDR mining reward;
- local mine -> receive -> send -> mine flow;
- TCP P2P listener and outbound connection;
- bounded length-prefixed P2P framing;
- Node A ↔ Node B handshake;
- chain ID validation;
- P2P protocol-version validation;
- self-connection rejection;
- duplicate-peer rejection;
- exchange and recording of each peer's latest block height;
- inbound/outbound peer tracking.

## P2P v0.1 - Day 8

Day 8 intentionally implements only peer transport and connection establishment.

A node is configured with:

```text
node_id
listen_address
chain_id
protocol_version
height_provider
```

A connection begins with a framed JSON `hello` message carrying:

```text
type
protocol_version
chain_id
node_id
listen_address
height
```

The frame format is:

```text
4-byte big-endian payload length
JSON payload
```

Handshake payloads are size-limited. A peer is accepted only when:

- the message is a valid VALDR hello;
- `chain_id` matches the local network;
- P2P protocol versions match;
- the remote node ID is valid and is not the local node ID;
- the peer is not already connected;
- the remote listen address is syntactically valid.

After a successful handshake both nodes retain peer metadata including the remote latest block height.

The Day 8 test uses real loopback TCP sockets and verifies:

```text
Node A height=7
      |
      | TCP + VALDR hello
      v
Node B height=3
```

After connection:

```text
Node A sees Node B at height 3
Node B sees Node A at height 7
```

### Deliberately deferred to Day 9

The following are not implemented by Day 8:

- block broadcast;
- transaction broadcast;
- peer discovery;
- missing-block requests;
- blockchain synchronization.

Those are the next milestone in the master specification.

## Coinbase and mining reward v0.1

VALDR v0.1 allows new VDR only through the block coinbase transaction.

Current reward:

```text
50 VDR = 5,000,000,000 val
```

Coinbase is required at transaction index zero of each non-genesis block and the reward is validated by consensus.

## UTXO Engine v0.1

Normal signed transactions preserve value:

```text
sum(inputs) == sum(outputs)
```

Only a validated block coinbase may create new value. Spent outpoints are removed, preventing a second spend. Block transaction application is atomic.

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

Not implemented yet: P2P data propagation/synchronization, persistent blockchain storage, RPC behavior, final halving interval, and the final three-node devnet scenario.

## Build and test

```bash
go build ./...
go test ./...
```

## Archived Bitcoin Core experiment

The previous Bitcoin Core 31.1 / C++ experiment is not the active v0.1 architecture. It is preserved unchanged on branch `archive/bitcoin-core-experiment-0.5.0` at commit `bd32a30c698056ac601a6553e74169724a56e6ac`.

Legacy files such as `consensus/valdr-consensus.json` remain historical artifacts and are not the active Go v0.1 protocol configuration.

See `docs/COMPLIANCE_AUDIT.md` for the recorded divergences.
