# VALDR Core

VALDR is a standalone cryptocurrency and blockchain project.

The frozen working baseline is **VALDR Devnet v0.1** at `release/valdr-devnet-v0.1`.
Active development on branch `valdr-v0.2` follows `docs/VALDR_Master_TZ_v0.2.md`.

## v0.2 development status

**Stage 1 — Storage v2 + v0.1 migration: implemented and CI-verified.**

Current Stage 1 changes:

- active v0.2 node storage uses **BadgerDB v4.9.4** with schema version `2`;
- block, height, header, transaction, UTXO, undo and metadata indexes are written atomically per accepted block;
- database identity binds schema version, network and Genesis hash before node services start;
- deterministic UTXO-set hash protects active-state consistency;
- `blockchain.json` remains read-only compatibility input for explicit one-way migration;
- `valdrd migrate --from-v0.1 <data> --network valdr-devnet-1` replays and verifies the legacy chain;
- migration verifies height, tip hash, every confirmed txid and UTXO-set hash;
- original v0.1 `blockchain.json` is preserved;
- `valdrd verify-db --data <path>` replays and verifies the indexed database;
- persistence failure rolls the candidate block back from in-memory confirmed state;
- clean build, full tests, race detector, storage/migration gate and three-node runtime smoke pass.

**Stage 2 — Network profiles + P2P v2 framing/versioning: implemented and CI-verified.**

Stage 2 adds the master-spec network profiles for legacy v0.1, `devnet2` and `testnet`, plus the v2 wire envelope (network magic, uint16 protocol/message type, bounded payload length, SHA-256 checksum and strict UTF-8 JSON). The node has an explicit v2 handshake path with `hello/hello_ack`, highest-mutual version selection, wrong-network rejection, ping/pong and peer discovery.

The existing v0.1-compatible runtime remains the default while chainwork/reorg and headers-first synchronization are still pending. v2 block/transaction/header data messages are intentionally not activated yet; they belong to later v0.2 stages and are not silently routed through the legacy `get_block` synchronizer.

**Stage 3 — Chainwork + side branches + reorganization: implemented and CI-verified.**

The active branch is selected by greatest cumulative chainwork, where each block contributes `floor(2^256 / (target + 1))`. Equal chainwork keeps the current active tip. Valid competing branches remain stored and queryable.

A heavier competing branch triggers the master-spec reorganization sequence: find the common ancestor, disconnect the old branch through per-block UTXO undo data, connect the new branch through normal transaction/coinbase validation, then atomically switch Badger active indexes, UTXO state, height mapping, confirmed transaction index and `meta/chainwork`. Disconnected blocks remain retained as side-branch blocks and survive restart.

**Stage 4 — Difficulty v2 + timestamp rules: implemented and CI-verified.**

The v0.2 profiles compile the 60-second target interval, 60-block retarget interval, 3,600-second target timespan, 900..14,400-second clamp, MTP-11 rule and +2-hour future-time limit. Testnet additionally enables the 10-minute min-difficulty escape and deterministic recovery to the last non-special target.

All v2 target arithmetic uses integer/big-int math. The frozen v0.1 runtime keeps its legacy per-block difficulty function; the v2 rules are isolated to the v0.2 network profiles.

**Stage 5 — Fees + consensus block/transaction size limits: implemented and CI-verified.**

Normal transactions now use the v0.2 implicit fee rule `fee = input_total - output_total`; overspend remains invalid. Block validation totals transaction fees before checking the coinbase. Coinbase may claim any positive amount up to `subsidy(height) + block fees`, so under-claim is valid and the unclaimed value is not created. The miner claims subsidy plus the exact fees of its selected transaction set.

Consensus size limits are enforced at 100,000 canonical bytes per transaction and 1,000,000 canonical bytes per block. Oversized blocks are rejected before PoW/UTXO validation, and wallet-side signing refuses to finish an oversized transaction.

**Stage 6 is not started.** Next master-spec milestone: mempool policy, fee-rate ordering and post-connect/disconnect revalidation.

## Stage 5 fees/size gate

Stage 5 implements and tests:

- implicit normal-transaction fees in `val`: `inputs - outputs`;
- overspend rejection when outputs exceed inputs;
- block fee accumulation on the branch UTXO view;
- coinbase maximum claim = subsidy + block fees;
- valid coinbase under-claim;
- miner coinbase construction with exact available fees;
- 100,000-byte transaction consensus limit;
- 1,000,000-byte block consensus limit;
- deterministic golden vectors for transaction and block canonical serialization.

Canonical integer encoding is big-endian. Lengths/counts are unsigned 64-bit values.

Canonical transaction bytes are:

```text
chain_id length + chain_id
version uint32
timestamp uint64
input_count uint64
  repeated: previous_txid length + bytes, output_index uint32
output_count uint64
  repeated: amount uint64, recipient length + bytes
public_key length + bytes
signature length + bytes
```

`transaction_id` is derived as SHA-256 of those bytes and is not included in its own serialization.

Canonical block-size bytes are:

```text
existing HeaderBytes:
  version uint32
  height uint64
  previous_block_hash length + bytes
  merkle_root length + bytes
  timestamp uint64
  difficulty uint64
  nonce uint64
  chain_id length + bytes
  extra_data length + bytes
transaction_count uint64
  repeated: transaction_length uint64 + canonical transaction bytes
```

`block_hash` is derived from the existing header bytes and is not included in block-size serialization.

**Master-TZ clarification:** v0.2 requires canonical serialization to be documented and covered by golden vectors, but does not itself spell out the exact byte layout for size accounting. The implementation above freezes that layout without changing the existing transaction-ID or block-hash algorithms. This byte layout should be copied into the next master-TZ revision before Public Testnet so independent implementations use identical size accounting.

## Stage 4 difficulty/timestamp gate

The Stage 4 consensus module implements:

- exact 256-bit PoW targets from the compiled network PoW limit;
- a 60-block retarget boundary and 3,600-second target timespan;
- deterministic integer `new_target = old_target * actual_timespan / target_timespan`;
- 900-second lower and 14,400-second upper timespan clamps;
- network PoW-limit bounding;
- Median-Time-Past over the previous 11 available branch headers;
- strict `candidate timestamp > MTP`;
- rejection above local system time + 2 hours;
- Testnet min-difficulty after more than 10 minutes without a block;
- the next normal Testnet block returning to the last non-special target;
- target-based PoW validation/mining helpers for later devnet2/testnet activation.

Golden tests pin exact target hex values, clamps, MTP rejects, future-time rejects, Testnet escape/recovery and retarget behavior.

**Implementation clarification:** the master spec fixes the 60-block window and formula but does not explicitly name the two timestamp endpoints used for `actual_timespan`. The implementation freezes the boundary as: for candidate height divisible by 60, use the first and last timestamps in the preceding 60 accepted headers. This keeps the retarget a pure function of already accepted history. This endpoint convention should be added explicitly to the next master-TZ revision before Public Testnet so independent implementations cannot interpret the window differently.

## Stage 3 chainwork/reorg gate

Stage 3 adds:

- deterministic block-work and cumulative-chainwork calculation;
- side-branch retention instead of rejecting every non-tip parent;
- greatest-chainwork active-tip selection;
- equal-work stability (the current tip wins ties);
- atomic UTXO undo for block disconnect;
- normal block validation when reconnecting the winning branch;
- Badger persistence for side blocks, per-header target/chainwork and undo records;
- atomic reorg updates of `height/*`, `tx/*`, `utxo/*`, active tip/height and `meta/chainwork`;
- restart reconstruction of both the active chain and retained side branches.

The Stage 3 competing-chain test builds two valid branches from the same Genesis, verifies that equal cumulative work does not switch the tip, extends the side branch until it becomes heavier, verifies balances after undo/reconnect, then reopens Badger and verifies the same winning tip and retained disconnected branch.

## Stage 2 protocol gate

Network profiles:

| Profile | Chain ID | P2P | P2P port | RPC port |
| --- | --- | ---: | ---: | ---: |
| `legacy-v0.1` | `valdr-devnet-1` | v1 | 7333 | 7332 |
| `devnet2` | `valdr-devnet-2` | v2 | 7333 | 7332 |
| `testnet` | `valdr-testnet-1` | v2 | 17333 | 17332 |

P2P v2 frame:

```text
4 bytes  network magic = first 4 bytes SHA-256(chain_id)
2 bytes  protocol version
2 bytes  message type
4 bytes  payload length
4 bytes  checksum = first 4 bytes SHA-256(payload)
N bytes  strict UTF-8 JSON payload
```

Automated Stage 2 coverage includes deterministic frame/network-magic vectors, checksum rejection, strict JSON rejection, wrong-network rejection, version negotiation, live v2 hello/hello_ack, and v2 peer discovery. The v0.2 CI keeps the existing storage and legacy runtime gates in place as regression protection.

## Current runtime protocol identity (Stage 1 compatibility baseline)

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

## v0.1 baseline status

Completed frozen baseline: **Day 14 - VALDR Devnet v0.1 final integration**.

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
