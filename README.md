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

Completed milestone: **Day 11 - local RPC API + CLI integration**.

Implemented through Day 9:

- fixed Genesis, blocks, Proof of Work and difficulty;
- wallet keys, addresses, signatures and signed payments;
- UTXO balances, spending and double-spend protection;
- coinbase and 50 VDR mining reward;
- TCP P2P handshake and peer tracking;
- block broadcast;
- transaction broadcast;
- peer discovery;
- missing-block requests by height;
- sequential basic blockchain synchronization;
- in-memory mempool for valid unconfirmed transactions;
- mempool duplicate and unconfirmed-input conflict protection;
- confirmed transaction removal and stale mempool pruning;
- thread-safe blockchain access for concurrent P2P readers;
- three-node A ↔ B ↔ C integration with identical confirmed block hashes and balances;
- local HTTP RPC API on the devnet RPC port;
- RPC queries for status, blocks, transactions, balances, mempool, peers and mining info;
- RPC transaction submission;
- RPC-backed valdr-cli status/block/tx/balance/send/peers/mempool/mining commands;
- valdrd init/start/status/version command surface.

## Day 9 P2P data propagation

The v0.1 wire transport remains length-prefixed JSON:

```text
4-byte big-endian payload length
JSON payload
```

Maximum P2P frame payload is currently 4 MiB.

Supported message types:

```text
hello
transaction
block
get_block
get_peers
peers
```

### Basic synchronization

A peer advertises its latest block height during the handshake.

When a node sees that a peer is ahead, it requests exactly the next missing height:

```text
Node B height 0
Node A height 2

B -> A: get_block(1)
A -> B: block(1)
B validates and appends block(1)
B -> A: get_block(2)
A -> B: block(2)
B validates and appends block(2)
```

Every received block passes the existing blockchain validation path: height/linkage, difficulty, Merkle root, block hash, PoW, coinbase and UTXO validation.

The basic synchronizer does not silently accept alternate history. If a block at an already-confirmed height has a different hash, v0.1 reports an unsupported fork/reorganization instead of replacing the local chain. Full fork-choice and reorganization rules are not defined in the current master specification.

### Block broadcast

A node may broadcast only a block already present in its local validated chain. Receiving peers validate and append the block before relaying it further.

Transactions confirmed by an accepted block are removed from mempool. Remaining mempool transactions are revalidated against the new confirmed UTXO state and stale/conflicting transactions are removed.

### Transaction broadcast and mempool

Before a local or remote transaction enters mempool:

1. the transaction structure, signature and transaction ID must validate;
2. the confirmed blockchain UTXO state must allow the spend;
3. the same transaction ID must not already exist in mempool;
4. no existing unconfirmed transaction may already use the same input.

Accepted transactions are relayed to other connected peers. Duplicate relays are ignored after mempool deduplication.

Current v0.1 mempool does not support spending outputs of another still-unconfirmed mempool transaction; validation is against confirmed UTXO state.

### Peer discovery

Connected peers exchange known `node_id + listen_address` pairs through `get_peers` / `peers`.

Discovered peers are stored as connection candidates. Day 9 does not automatically build a three-node synchronized topology; the full Node A + Node B + Node C same-chain scenario is reserved for Day 10.

## Day 9 automated network scenario

The test suite verifies with real loopback TCP sockets:

```text
Node A: Genesis -> Block 1 -> Block 2
Node B: Genesis

Node B connects to Node A
        ↓
B requests Block 1
        ↓
B validates Block 1
        ↓
B requests Block 2
        ↓
A and B have the same height and hashes
        ↓
A broadcasts a signed payment
        ↓
B stores it in mempool
        ↓
A mines Block 3 containing the payment
        ↓
A broadcasts Block 3
        ↓
B validates Block 3
        ↓
payment removed from both mempools
        ↓
A and B have the same confirmed tip and balances
```

A separate discovery test connects B to C, then A to B, and verifies that A learns C's address through B. It does **not** claim the Day 10 three-node blockchain convergence milestone.

## Day 10 three-node devnet integration

Day 10 validates the master-spec minimum test network with three independent in-memory blockchains and three real loopback TCP P2P nodes.

Topology:

```text
Node A <-> Node B <-> Node C
```

The automated scenario verifies:

```text
Node A mines Block 1 and Block 2
        ↓
Node B connects to A and synchronizes
        ↓
Node C connects to B and synchronizes through B
        ↓
A, B and C have identical block hashes at every confirmed height
        ↓
Node C broadcasts a signed 25 VDR payment
        ↓
C -> B -> A transaction relay
        ↓
all three mempools contain the same transaction
        ↓
Node A mines Block 3 containing the payment
        ↓
A -> B -> C block relay
        ↓
all three chains converge on the same Block 3
        ↓
confirmed transaction is removed from all three mempools
        ↓
all three nodes report the same balances
```

Expected final confirmed state in the test:

```text
height             = 3
recipient balance  = 25 VDR
miner balance      = 125 VDR
mempool size       = 0 on A, B and C
```

The test compares every confirmed block hash from Genesis through the tip, not only the final height.

Day 10 does not add RPC/CLI behavior. That is the next master-plan stage.

## Day 11 RPC API + CLI

The node exposes a local HTTP endpoint:

```text
POST /rpc
default: http://127.0.0.1:7332/rpc
```

The request envelope is:

```json
{
  "method": "getStatus",
  "params": {}
}
```

Mandatory v0.1 RPC methods from the master specification are implemented:

```text
getStatus
getBlock
getBlockByHash
getTransaction
getBalance
getMempool
sendTransaction
getPeers
getMiningInfo
```

Day 11 also adds one read-only helper:

```text
getUTXOs
```

`getUTXOs` exists so `valdr-cli send` can select spendable outputs and sign the transaction locally. The node never receives the wallet private key.

### CLI

Examples:

```bash
valdr-cli status
valdr-cli block get 2
valdr-cli tx get <txid>
valdr-cli balance VDR1...
valdr-cli peers
valdr-cli mempool
valdr-cli mining info

valdr-cli send \
  --from alice \
  --to VDR1... \
  --amount 10
```

A non-default RPC endpoint can be selected with `--node`.

Amounts passed to `send --amount` are parsed as decimal VDR with at most 8 decimal places and converted to atomic `val` without floating-point arithmetic.

### valdrd

The node command surface now includes:

```bash
valdrd init --data ./data/node1

valdrd start \
  --data ./data/node1 \
  --node-id node1 \
  --p2p-port 7333 \
  --rpc-port 7332

valdrd status
valdrd version
```

`valdrd start` starts the existing in-memory blockchain, mempool, P2P transport and local RPC server in one process. Repeated `--peer host:port` flags may be used for outbound P2P connections.

Important current limitation: the `--data` directory stores node metadata only. Blockchain persistence to an embedded database is not yet implemented, so a restarted Day 11 node still starts from Genesis. This limitation is intentionally not hidden.

### Explorer-ready data

The existing block and transaction JSON returned by RPC exposes the fields required by the master specification for a later explorer: block height/hash/previous hash, transactions, transaction IDs, addresses, amounts, difficulty and timestamp.

## Coinbase and mining reward v0.1

Only a validated block coinbase may create new VDR.

```text
50 VDR = 5,000,000,000 val
```

## UTXO Engine v0.1

Normal transactions preserve value:

```text
sum(inputs) == sum(outputs)
```

Fees are not part of the 14-day MVP.

## Wallet and cryptography v0.1

```text
signature: ECDSA P-256 over SHA-256(message)
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
- block hash: `47e3a6c15cab1a41c54a36a65f7133261fa6f75976a2e59825694e001716bfe5`

Not implemented yet: persistent blockchain storage, final halving interval, full fork/reorganization policy, and the Day 12 security-hardening pass.

## Build and test

```bash
go build ./...
go test ./...
```

## Archived Bitcoin Core experiment

The previous Bitcoin Core / C++ experiment is not the active v0.1 architecture. It is preserved unchanged on branch `archive/bitcoin-core-experiment-0.5.0` at commit `bd32a30c698056ac601a6553e74169724a56e6ac`.

Legacy files such as `consensus/valdr-consensus.json` remain historical artifacts and are not the active Go v0.1 protocol configuration.

See `docs/COMPLIANCE_AUDIT.md` for the recorded divergences.
