# VALDR Core

VALDR is a standalone cryptocurrency and blockchain project. `main` preserves the tested VALDR Devnet v0.1 baseline from `VALDR_Master_TZ_v0.1_14_days.pdf`. Active v0.2 development is isolated on `valdr-v0.2-dev`.

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

Stable baseline: **Day 14 - VALDR Devnet v0.1 final integration**.\n\nActive milestone: **VALDR v0.2 - Block Explorer, slice 1**.

Implemented through Day 14:

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
- valdrd init/start/status/version command surface;
- `scripts/start-devnet.sh` for build/start/status/stop/smoke of a local three-node devnet;
- CI smoke test that starts three real `valdrd` processes and verifies RPC/P2P health;
- persistent on-disk blockchain state with validated replay and UTXO reconstruction;
- real `valdr-miner start/status/stop` command surface;
- structured NODE/P2P/BLOCK/TX/MINER/MEMPOOL/SYNC/ERROR logs;
- final executable three-node mine/send/mine/sync/restart/persistence test.

## VALDR v0.2 - Block Explorer slice 1

The first v0.2 change follows section 30 of the master specification and adds a read-only web explorer without changing consensus, block/transaction formats, P2P, mining, or wallet rules.

New binary:

```bash
go run ./cmd/valdr-explorer \
  --node http://127.0.0.1:7332 \
  --listen 127.0.0.1:7331 \
  --index-file ~/.valdr/explorer-index.json
```

Current explorer routes:

- `/` - network status and the latest blocks;
- `/block/<height-or-hash>` - block details and transaction links;
- `/tx/<txid>` - confirmed/mempool transaction details;
- `/address/<VDR-address>` - confirmed address balance plus received/spent transaction history;
- `/search?q=...` - height/hash/txid/address lookup;
- `/healthz` - node-backed health response.

The explorer uses the existing node RPC only and keeps private keys outside the process. It adds basic browser security headers and server timeouts.

The explorer maintains a separate confirmed-chain index. It resolves transaction inputs against previously indexed outputs so address pages can show received and spent amounts. The default index is `~/.valdr/explorer-index.json` (override with `--index-file` or `VALDR_EXPLORER_INDEX_FILE`). Writes use a temporary `0600` file plus `fsync` and atomic rename. The index verifies its last indexed block against the node and rebuilds if confirmed history changes.

Explorer acceptance also has a live integration gate:

```bash
bash ./scripts/test-explorer-v0.2.sh
```

That scenario starts the real three-node Devnet, mines a block, runs `valdr-explorer` against `valdrd`, verifies network/block/transaction/address pages, confirms the persistent index is created, restarts the Explorer, and verifies indexed history again.

Pagination and richer aggregate statistics are UI hardening items; they do not change the VALDR consensus or node protocol.

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

`valdrd start` starts the blockchain, mempool, P2P transport and local RPC server in one process. Repeated `--peer host:port` flags may be used for outbound P2P connections. The confirmed blockchain is loaded from and persisted to `<data>/blockchain.json`; startup replays persisted blocks through normal consensus/UTXO validation before serving RPC/P2P.

### Explorer-ready data

The existing block and transaction JSON returned by RPC exposes the fields required by the master specification for a later explorer: block height/hash/previous hash, transactions, transaction IDs, addresses, amounts, difficulty and timestamp.

## Day 12 security checks

The Day 12 pass maps the master-spec MVP security requirements to explicit rejection paths and regression tests.

| Required check | Current VALDR v0.1 behavior |
| --- | --- |
| double spend | spent UTXO disappears; a second spend is rejected; two spends of one UTXO in the same block fail atomically |
| bad signature | malformed, tampered-message and wrong-key ECDSA signatures are rejected |
| negative amount | transaction output amount is `uint64`; negative JSON amounts cannot decode into the transaction type |
| VDR creation outside coinbase | normal transactions require value conservation and cannot produce outputs above their confirmed inputs |
| bad block hash | block hash must exactly equal SHA-256 of the current block header |
| bad previous hash | candidate must reference the current confirmed tip |
| bad nonce | changing nonce without the matching block hash is rejected |
| bad PoW | a correctly encoded hash above the current target is rejected |
| duplicate block | a previously confirmed valid block hash is explicitly rejected with `ErrDuplicateBlock` |
| duplicate transaction | duplicate txids inside a block or replay of a confirmed txid are explicitly rejected with `ErrDuplicateTransaction` |
| bad coinbase reward | coinbase output must equal the consensus reward for the block height |

### Atomic rejection

Security tests verify that invalid block processing does not partially mutate confirmed state.

For example:

```text
confirmed 50 VDR UTXO
        ↓
forged normal tx outputs 50 VDR + 1 val
        ↓
block is mined but fails UTXO validation
        ↓
block height unchanged
miner balance unchanged
recipient balance remains 0
```

The same rollback property is tested for a two-transaction double spend inside one candidate block.

### Existing checks retained

Day 12 does not replace the earlier consensus/UTXO validation. Existing tests continue to cover:

- wrong previous hash;
- tampered Merkle root;
- wrong difficulty;
- PoW above target;
- missing coinbase;
- wrong coinbase reward;
- missing UTXO;
- wrong UTXO owner;
- insufficient input value;
- transaction ID tampering;
- mempool duplicate transaction;
- mempool unconfirmed-input conflict.

No new cryptographic algorithm or consensus architecture is introduced by Day 12.

## Day 13 automated devnet startup

The master-plan Day 13 startup entrypoint is:

```bash
./scripts/start-devnet.sh
```

The script builds `valdrd` and `valdr-cli`, initializes three local data directories, starts three real node processes and connects them in this topology:

```text
Node A <-> Node B <-> Node C
```

Default local ports:

```text
Node A: P2P 7333 / RPC 7332
Node B: P2P 7433 / RPC 7432
Node C: P2P 7533 / RPC 7532
```

The generated runtime directory is `.valdr-devnet/` and is ignored by Git.

Supported operations:

```bash
./scripts/start-devnet.sh start
./scripts/start-devnet.sh status
./scripts/start-devnet.sh stop
./scripts/start-devnet.sh smoke
```

A normal `start` does not return success until all three RPC endpoints answer and the expected peer topology is visible:

```text
Node A peers = 1
Node B peers = 2
Node C peers = 1
```

The script also verifies that all three nodes report:

- chain ID `valdr-devnet-1`;
- the same confirmed height;
- the same confirmed tip hash.

If startup fails partway through, already-started nodes are stopped instead of leaving a partial devnet running.

### CI automation

The GitHub Actions pipeline now runs:

```bash
go build ./...
go test ./...
bash -n ./scripts/start-devnet.sh
./scripts/start-devnet.sh smoke
```

The smoke mode uses an isolated temporary runtime directory, starts the three node processes, verifies the network, then terminates the processes and removes the temporary directory.

### Day 14 readiness

The Day 13 startup gate is retained. Day 14 adds the final user-facing integration and persistence checks described below.

## Day 14 VALDR Devnet v0.1

The final executable integration entrypoint is:

```bash
./scripts/test-devnet-v0.1.sh
```

It uses the built binaries, not in-process mocks:

```text
3 valdrd processes online
        ↓
wallet A created
wallet B created
        ↓
valdr-miner mines 50 VDR to A
        ↓
all nodes synchronize Block 1
        ↓
A sends 10 VDR to B through valdr-cli
        ↓
transaction broadcasts into all three mempools
        ↓
valdr-miner mines the confirming block
        ↓
all nodes synchronize Block 2
        ↓
B = 10 VDR on A, B and C
A = 90 VDR on A, B and C
        ↓
all three nodes stop
        ↓
all three nodes restart from the same data directories
        ↓
height, tip hash, confirmed transaction and balances are preserved
```

### Persistent blockchain

Each node stores confirmed chain state at:

```text
<data>/blockchain.json
```

v0.1 persistence uses a dependency-free atomic snapshot:

1. encode the complete validated chain to a temporary file;
2. file mode `0600`;
3. `fsync` the temporary file;
4. atomically rename it over the previous snapshot.

On startup the node loads the file, checks the canonical Genesis and replays every stored non-Genesis block through the existing block/PoW/coinbase/UTXO validation path. UTXO state is reconstructed from the validated history.

If persistence fails before the atomic rename, the candidate block is rolled back from RAM so confirmed in-memory and on-disk state do not diverge.

The master specification names LevelDB or BadgerDB as **preferred** MVP choices. v0.1 deliberately uses the atomic file backend to keep the first devnet dependency-free. This does not change block, transaction, consensus or network formats. The tradeoff is O(chain size) rewrite cost per confirmed block, so a database backend should replace it before a larger public testnet.

### Miner

`valdr-miner` is now functional:

```bash
valdr-miner start \
  --node http://127.0.0.1:7332 \
  --reward-address VDR1... \
  --blocks 1

valdr-miner status --node http://127.0.0.1:7332
valdr-miner stop
```

The miner controller calls the node's v0.1 `mineBlock` RPC extension. The node selects its current mempool, builds coinbase + candidate block, executes the existing Proof of Work, commits the block to persistent storage, removes confirmed mempool transactions and broadcasts the block to peers.

`--blocks 0` means continuous mining until the miner receives SIGINT/SIGTERM. A protected PID file backs the `status/stop` process controls.

### Structured logging

Runtime events use the master-spec category prefixes:

```text
[NODE]
[P2P]
[BLOCK]
[TX]
[MINER]
[MEMPOOL]
[SYNC]
[ERROR]
```

### CI readiness gate

Every candidate now executes:

```bash
go build ./...
go test ./...
bash -n ./scripts/start-devnet.sh
bash -n ./scripts/test-devnet-v0.1.sh
./scripts/start-devnet.sh smoke
./scripts/test-devnet-v0.1.sh
```

A commit is not considered VALDR Devnet v0.1-ready unless all of these pass.

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

Intentionally not frozen in v0.1: the exact halving interval (the master specification defers it until block-speed/economic testing) and a full fork/reorganization policy. Those protocol refinements must be specified before a later public testnet/mainnet.

## Build and test

```bash
go build ./...
go test ./...
./scripts/start-devnet.sh smoke
./scripts/test-devnet-v0.1.sh
```

## Archived Bitcoin Core experiment

The previous Bitcoin Core / C++ experiment is not the active v0.1 architecture. It is preserved unchanged on branch `archive/bitcoin-core-experiment-0.5.0` at commit `bd32a30c698056ac601a6553e74169724a56e6ac`.

Legacy files such as `consensus/valdr-consensus.json` remain historical artifacts and are not the active Go v0.1 protocol configuration.

See `docs/COMPLIANCE_AUDIT.md` for the recorded divergences.
