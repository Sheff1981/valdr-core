# VALDR Core

VALDR is a standalone cryptocurrency and blockchain project.

The frozen working baseline is **VALDR Devnet v0.1** at `release/valdr-devnet-v0.1`.
Active development on branch `valdr-v0.2` follows the current baseline `docs/VALDR_Master_TZ_v0.2.4.md`. Earlier v0.2/v0.2.1/v0.2.2/v0.2.3 specifications remain preserved. v0.2.4 keeps the Desktop/cross-platform plan and adds explicit network binding for v2 transactions to prevent cross-network replay.

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

The frozen v0.1-compatible runtime remains available on the legacy profile. Devnet2 uses the separate v2 path; later stages have since activated exact-target blocks, chainwork/reorg and headers-first synchronization without changing the frozen v0.1 wire/consensus format.

**Stage 3 — Chainwork + side branches + reorganization: implemented and CI-verified.**

The active branch is selected by greatest cumulative chainwork, where each block contributes `floor(2^256 / (target + 1))`. Equal chainwork keeps the current active tip. Valid competing branches remain stored and queryable.

A heavier competing branch triggers the master-spec reorganization sequence: find the common ancestor, disconnect the old branch through per-block UTXO undo data, connect the new branch through normal transaction/coinbase validation, then atomically switch Badger active indexes, UTXO state, height mapping, confirmed transaction index and `meta/chainwork`. Disconnected blocks remain retained as side-branch blocks and survive restart.

**Stage 4 — Difficulty v2 + timestamp rules: implemented and CI-verified.**

The v0.2 profiles compile the 60-second target interval, 60-block retarget interval, 3,600-second target timespan, 900..14,400-second clamp, MTP-11 rule and +2-hour future-time limit. Testnet additionally enables the 10-minute min-difficulty escape and deterministic recovery to the last non-special target.

All v2 target arithmetic uses integer/big-int math. The frozen v0.1 runtime keeps its legacy per-block difficulty function; the v2 rules are isolated to the v0.2 network profiles.

**Stage 5 — Fees + consensus block/transaction size limits: implemented and CI-verified.**

Normal transactions now use the v0.2 implicit fee rule `fee = input_total - output_total`; overspend remains invalid. Block validation totals transaction fees before checking the coinbase. Coinbase may claim any positive amount up to `subsidy(height) + block fees`, so under-claim is valid and the unclaimed value is not created. The miner claims subsidy plus the exact fees of its selected transaction set.

Consensus size limits are enforced at 100,000 canonical bytes per transaction and 1,000,000 canonical bytes per block. Oversized blocks are rejected before PoW/UTXO validation, and wallet-side signing refuses to finish an oversized transaction.

**Stage 6 — Mempool policy + miner template selection: implemented and CI-verified.**

The mempool is bounded to 64 MiB of canonical transaction bytes with 72-hour expiry. Testnet profile compiles a 1 val/byte minimum relay fee. RBF remains disabled: a second unconfirmed spend of the same outpoint is rejected, and spending a mempool parent is not relayed in v0.2. Eviction is exact fee-rate ascending then oldest; mining order is exact fee-rate descending then txid.

Node admission calculates the real implicit fee against the active UTXO set. After active-chain connect/reorg the pool is revalidated, confirmed transactions are removed, and valid non-coinbase transactions from disconnected blocks can be reconsidered. Miner template selection uses fee-rate order and skips transactions that would exceed the 1,000,000-byte block limit.

**Stage 7 — Full headers-first synchronization: implemented and CI-verified.**

Devnet2 P2P v2 now performs block-locator based synchronization. A peer may return at most 2,000 headers per batch; the receiver verifies network/version, linkage, exact target, MTP/future timestamp rules, difficulty transition and PoW before requesting any block body. Block bodies are requested in bounded windows of at most 32 outstanding items and must exactly match a previously validated/requested header.

Sync uses cumulative chainwork rather than height as the fork-choice trigger, while height remains only a compatibility fallback when chainwork metadata is unavailable. Valid side branches flow through the Stage 3 persistence/reorg path. Session state is ephemeral; persisted blocks and branches are the resume source after restart.

CI covers fresh Genesis-to-tip sync, restart/offline resume using BadgerDB, greater-chainwork reorg sync, live `inv → get_headers → headers → get_data → block` catch-up, strict locator/header/inventory payload rules and chainwork-based sync decisions.

**Stage 8 — Seed bootstrap + P2P protection: implemented and CI-verified (software gate).**

Network profiles now carry a compiled seed-list field and operators can extend bootstrap with repeated `--seed host:port` values. Seed failures are best-effort and do not stop a running node; bootstrap continues through remaining seeds and then uses discovered peers in bounded rounds toward the outbound target of 8. With no seeds configured, automatic maintenance remains disabled so legacy/manual topologies are unchanged.

P2P protection defaults now implement the master-spec requirements: 5-second handshake timeout, 5-minute idle detection with ping-before-close, max 64 inbound peers, max 4 inbound peers per IP, 4 MiB global frame cap plus per-message caps, token-bucket message/byte limits, malformed-IP scoring with a 1-hour temporary ban, bounded duplicate caches, and public-discovery rejection of unsafe numeric addresses.

Operational defaults not numerically fixed by the master spec are currently: 30-second ping grace, malformed threshold 10, duplicate cache 4,096 entries, 64 messages/s with burst 128, and 2 MiB/s with 4 MiB burst. These are local anti-abuse policy, not consensus parameters.

**Public Testnet seed deployment note:** the repository does not invent public seed addresses before infrastructure exists. The Testnet compiled seed list remains unpopulated until Stage 12 provisions at least 3 stable public full nodes across at least 2 independent regions/providers. Those real endpoints must then be frozen into the Testnet profile and the seed gate rerun before public launch.

**Stage 9 — Wallet encryption v2: implemented and CI-verified.**

New wallet files never persist the private key in plaintext. Wallet v2 derives a 256-bit encryption key with scrypt and encrypts the private-key payload with AES-256-GCM using a random salt and nonce. Public metadata remains readable without unlocking so `wallet list` does not need a passphrase.

CLI wallet operations accept passphrases only through hidden interactive terminal input or `--password-fd N`. There is no `--password` string argument. Regular files supplied through `--password-fd` must not expose group/other permissions. `send` unlocks locally before signing; private keys never enter RPC/P2P payloads. `wallet export` remains an explicit high-risk operation and prints a warning.

`wallet migrate` validates a legacy v0.1 plaintext wallet, preserves its name/address/public/private key identity, then atomically rewrites the same wallet path as encrypted v2 with directory mode 0700 and wallet mode 0600. Legacy plaintext wallets cannot be used for signing until migrated.

**Stage 10 — Explorer + reorg-safe indexer: implemented and CI-verified.**

`cmd/valdr-explorer` is a standalone read-only HTTP service. Its default listen address is `127.0.0.1:8080`; it talks only to the node RPC endpoint supplied with `--node` and exposes no signing, mining or privileged RPC controls.

The Explorer provides search by block height/hash, txid and VDR address; HTML views for overview/block/transaction/address; and the required REST surface: `/api/v1/status`, `/api/v1/blocks`, `/api/v1/block/{height-or-hash}`, `/api/v1/tx/{txid}`, `/api/v1/address/{address}`, and `/api/v1/mempool`. Status includes peers, network identity, chainwork, current target/difficulty and recent block intervals.

The Explorer keeps its own persistent index with active block hashes, confirmed address activity and UTXOs. On an active-chain change it locates the common ancestor, rolls the index back by rebuilding through that ancestor, then indexes the new active branch. The index records Chain ID + Genesis identity and refuses cross-network reuse.

**Stage 11 — Testnet + Docker + Linux deployment: implemented and CI-verified.**

The Testnet runtime is active under profile `testnet` / Chain ID `valdr-testnet-1`, protocol v2, P2P port 17333 and RPC port 17332. `valdrd init/start/verify-db --network testnet` bind storage identity to the frozen Testnet Genesis and reject databases from another network.

The frozen Testnet Genesis is:
- timestamp: `1790208000` (2026-09-24 00:00:00 UTC);
- target: `000fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff`;
- nonce: `12480`;
- message: `VALDR genesis block | valdr-testnet-1 | 2026-09-24`;
- hash: `0009d956448a8caefcd798af1a7957840d0aa7b72f8350362909f241ea100122`.

The root multi-stage Docker image builds `valdrd`, `valdr-cli`, `valdr-miner` and `valdr-explorer`, then runs as the unprivileged `valdr` user with persistent state under `/var/lib/valdr` and operator configuration under `/etc/valdr`. The Testnet Compose topology starts three v2 nodes plus Explorer, publishes P2P only for the first node, does not publish node RPC to the host, and binds the Explorer host port to localhost.

Public-profile P2P now separates bind address from advertised address. A node may listen on `0.0.0.0:17333` while advertising a routable DNS/IP endpoint through `--advertise-address`; public gossip still rejects unspecified, loopback, private, multicast and link-local numeric addresses. Linux deployment reads `VALDR_ADVERTISE_ADDRESS` from `/etc/valdr/valdr.env`.

The Linux reference deployment targets Ubuntu Server 24.04 LTS x86_64 with a dedicated non-login `valdr` account, hardened systemd units, journald/stdout logging, localhost RPC/Explorer, P2P-only firewall exposure, HTTPS reverse-proxy guidance, and documented backup/restore/verify/upgrade/rollback procedures.

CI verifies Testnet Genesis/runtime identity, Docker image construction, a three-node container Testnet that maintains P2P connectivity and synchronizes a newly mined block, Explorer visibility of that chain, Linux restart with the same BadgerDB plus `verify-db`, and the frozen legacy three-node smoke.

**Stage 12 — VALDR Desktop: in progress.** The current slice includes the Wails v2 shell, Testnet-only first-run encrypted-wallet setup, managed outbound-only local node lifecycle, sync progress, balance/send/receive, reorg-safe transaction history, encrypted backup/restore, explicit surfacing/restart recovery for an unexpectedly exited managed node, local wallet lock/unlock with configurable inactivity auto-lock, locally generated Receive QR codes with no web/API dependency, explicit high-risk private-key export requiring an unlocked wallet plus typed confirmation, the required latest-transaction summary on Overview, a dedicated irreversible-transaction confirmation dialog before signing/broadcast, and an explicit transaction detail view for wallet history. Stage 12 is not complete until the full Desktop acceptance gate in Master-TZ v0.2.3 is green across the mandatory OS matrix. Private/Public Testnet remains behind completed Stages 12 and 13.

## Stage 11 Testnet/Docker/Linux gate

Stage 11 implements and tests:

- frozen `valdr-testnet-1` Genesis and golden PoW verification;
- `valdrd --network testnet` runtime selection with network-specific storage identity;
- Testnet P2P v2 on 17333 and RPC on 17332;
- explicit routable P2P advertise-address distinct from the bind address;
- multi-stage Docker build containing node, CLI, miner and Explorer binaries;
- unprivileged container runtime user;
- persistent `/var/lib/valdr` and `/etc/valdr` layout;
- three-node Testnet Docker Compose with health checks;
- node RPC not host-published by the Compose deployment;
- localhost-bound host Explorer on 8080;
- real container mining followed by headers-first synchronization to all three nodes;
- Explorer observation/indexing of the synchronized Testnet block;
- Ubuntu 24.04 LTS x86_64 deployment documentation;
- hardened `valdrd` and Explorer systemd units;
- Linux node/Explorer process smoke, persistent restart and `verify-db`;
- backup, restore, upgrade and rollback procedures;
- legacy v0.1 three-node regression smoke after all Stage 11 gates.

The explicit `--advertise-address` option is operational configuration, not a consensus parameter, and does not change the master-TZ architecture. Freezing the Testnet Genesis fulfills the Stage 11 requirement in Master-TZ v0.2.2; Mainnet remains disabled.

## Stage 10 Explorer/reorg gate

Stage 10 implements and tests:

- standalone `cmd/valdr-explorer`;
- localhost default `127.0.0.1:8080`;
- read-only node RPC client;
- block height/hash, txid and VDR-address search;
- latest blocks, block detail, transaction detail and address views;
- address confirmed history and current UTXO display;
- mempool, peers, network identity, chainwork, target/difficulty and recent block intervals;
- required `/api/v1` REST endpoints;
- persistent Explorer-owned index;
- Chain ID + Genesis identity binding;
- restart-safe index reload;
- common-ancestor detection on reorg;
- stale history/UTXO rollback followed by active-branch reindex;
- security headers and GET/HEAD-only HTTP surface;
- live Devnet2 Explorer test against the real RPC server;
- dedicated Explorer API/reorg CI gate.

The Explorer index is deliberately separate from consensus storage. It derives only public chain data from read-only RPC and can be rebuilt without changing node consensus state.

## Stage 9 wallet-v2 encryption gate

Stage 9 implements and tests:

- wallet file version `2`;
- scrypt-derived 256-bit key;
- scrypt parameters `N=32768, r=8, p=1`;
- independent random 16-byte salt per wallet;
- AES-256-GCM with a fresh random nonce per encrypted file;
- authenticated metadata binding for version, name, address, public key and creation time;
- no plaintext private key or `private_key` field in v2 wallet files;
- metadata-only wallet listing without unlock;
- wrong-passphrase rejection;
- ciphertext and metadata tamper rejection;
- secure `--password-fd` / hidden terminal passphrase input and no `--password` CLI flag;
- protected regular password-file descriptors;
- explicit high-risk private-key export warning;
- v0.1 plaintext-wallet migration to encrypted v2;
- legacy wallet signing blocked until migration;
- CLI send/signing from an unlocked encrypted wallet;
- complete build, tests, race detector, Wallet v2 CI gate and legacy three-node smoke.

The master specification fixes scrypt + AES-256-GCM but does not specify the exact scrypt work factors or JSON wallet-v2 envelope. The values above are therefore the current implementation file format, not consensus parameters. They should be copied into a future master-TZ revision if cross-implementation wallet-file compatibility becomes a requirement.

## Stage 8 seed/P2P protection gate

Stage 8 software implements and tests:

- compiled network-profile seed-list support plus operator seed overrides;
- best-effort seed bootstrap where one offline seed does not abort startup;
- bounded post-seed discovery toward outbound target 8;
- no-seed mode preserving manual/legacy topology behavior;
- handshake timeout 5 seconds;
- idle watchdog at 5 minutes with ping before disconnect;
- inbound cap 64 and per-IP inbound cap 4;
- 4 MiB global frame payload limit plus smaller per-message limits;
- token-bucket inbound message and byte rate limiting;
- malformed IP score and 1-hour temporary bans;
- bounded transaction/inventory duplicate caches;
- public peer-gossip filtering for unspecified, loopback, private, multicast and link-local numeric addresses;
- regression coverage for seed failure, discovery beyond seeds, bans, rate limits, duplicate eviction, idle keepalive and old three-node topology.

The actual Public Testnet seed endpoints are deployment data, not fabricated placeholders. They remain a Stage 12 launch prerequisite: minimum 3 stable public nodes in at least 2 independent regions/providers, followed by updating `NetworkTestnetV02.DefaultSeeds`.

## Stage 7 headers-first sync gate

Stage 7 implements and tests:

- active-chain block locator with recent hashes then exponential backoff to Genesis;
- strict `get_headers/headers/get_data/block/get_blocks/inv/tx` v2 payload contracts;
- locator limit 128 hashes;
- header batches up to 2,000;
- exact v2 header validation before body download;
- maximum 32 outstanding/requested block bodies;
- body/header identity enforcement;
- fresh Devnet2 Genesis-to-tip synchronization;
- persisted Badger restart/offline resume without deleting the DB;
- competing-branch synchronization and reorg to greater cumulative chainwork;
- live v2 block announcements through `inv` followed by headers-first catch-up;
- v2 transaction relay through the Stage 6 mempool policy;
- cumulative-chainwork based sync trigger instead of height-only selection.

Master-TZ v0.2.1 froze exact 32-byte target encoding in block header v2. Master-TZ v0.2.2 then froze the Stage 7 JSON sync-wire contract and restart semantics. Frozen v0.1 header/P2P formats remain unchanged.

## Stage 6 mempool/miner-policy gate

Stage 6 implements:

- 64 MiB mempool serialized-byte cap;
- 72-hour arrival-time expiry;
- Testnet 1 val/byte minimum relay policy;
- RBF disabled via second-unconfirmed-spend rejection;
- no unconfirmed-parent relay;
- deterministic eviction: lowest exact fee-rate, then oldest, then txid;
- deterministic miner order: highest exact fee-rate, then txid;
- fee-rate comparison with integer 128-bit cross-products, never float;
- active-UTXO fee calculation at local/P2P admission;
- revalidation after active-chain changes;
- reconsideration of valid non-coinbase disconnected transactions;
- miner block-template byte-limit enforcement before PoW.

RPC mempool listing remains txid-sorted for a stable query surface; mining uses the separate fee-rate ordered snapshot.

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

**Implementation clarification:** the master spec fixes the 60-block window and formula but does not explicitly name the two timestamp endpoints used for `actual_timespan`. The implementation freezes the boundary as: for candidate height divisible by 60, use the first and last timestamps in the preceding 60 accepted headers. This keeps the retarget a pure function of already accepted history. This endpoint convention is now frozen explicitly in Master-TZ v0.2.1/v0.2.2 so independent implementations use the same retarget window.

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
