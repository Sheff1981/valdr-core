# VALDR / CONSOLIDATED MASTER TECHNICAL SPECIFICATION v0.2.27

**Two-machine functional Stage 14A baseline + automatic P2P participation + implementation-first engineering**  
**Date:** 1 October 2026  
**Status:** active master implementation baseline  
**Previous baseline:** `docs/VALDR_Master_TZ_v0.2.26.md`  
**Active branch:** `valdr-v0.2`

> v0.2.27 replaces v0.2.26 as the single working master specification. Historical specifications v0.2 through v0.2.26 remain immutable evidence of prior decisions. Frozen legacy v0.1 remains `release/valdr-devnet-v0.1`.

> This revision is intentionally cumulative and implementation-first. It preserves all frozen Testnet2 consensus/network values and the existing architecture. By explicit project-owner instruction on 2026-09-27, Stage 12C human Windows click-through is waived as a release-blocking gate for v0.2; it remains an optional checklist and is not represented as manually verified. Stage 14A independent multi-machine validation is moved ahead of the public Testnet RC freeze/publication so real distributed evidence becomes the next active gate. The project must work from this file first instead of reconstructing the plan from older documents.

## 1. Working rule

VALDR development now follows an implementation-first sequence:

**specification -> implementation -> build -> automated tests/CI -> next technical slice -> complete technical backlog -> consolidated end-to-end validation -> evidence -> release decision**

Rules:

1. Do not knowingly stack broken code: every substantial technical slice must compile and pass its relevant automated tests before the next technical slice.
2. Automated unit/integration/race/security/runtime tests remain mandatory during development. They are not deferred.
3. Manual GUI acceptance, independent-computer Testnet trials, long-running soak observation and public-user acceptance are deferred until the technical implementation backlog for the current milestone is complete.
4. Do not stop engineering work merely because a deferred manual/distributed acceptance gate has not yet been run.
5. Distinguish:
   - **implemented** — code exists;
   - **CI-verified** — automated gate passed on the relevant active network/version;
   - **manually verified** — final human-visible acceptance passed;
   - **distributed-session verified** — final independent-machine sessions passed;
   - **planned** — specified but not implemented.
6. A newer network/profile reset invalidates acceptance evidence that depended on the superseded profile.
7. Consensus, wire, storage, wallet-format or Mainnet-economic changes require a new Master-TZ revision.
8. UX-only changes may be implemented without a new spec only when they do not change mandatory behavior or security boundaries.
9. No public release, Mainnet launch or exchange-listing work may be represented as complete before the final validation phase and its prerequisites are green.
10. Historical stage numbers are requirement identifiers, not permission barriers. Where this v0.2.27 explicitly defers or waives an acceptance gate, later validation work may proceed without falsely claiming that the waived check was performed.
11. Stage 12C is **owner-waived, not passed**. Evidence and status text must preserve that distinction.
12. No public Testnet release is authorized by the Stage 12C waiver alone. Stage 14A must pass before the non-development public Testnet RC is frozen and published.

### 1.1 Current execution priority

Until the technical backlog is exhausted, work in this order:

1. Core consensus/Testnet2 correctness and active-network regression removal.
2. P2P v2 hardening: malformed/oversize/checksum/rate-limit/flood/duplicate/handshake/reconnect/discovery.
3. Chainwork/reorg hardening, including repeated/deeper forks and restart persistence.
4. BadgerDB crash/restart/recovery and atomicity verification.
5. RPC exposure/input/security hardening.
6. Wallet corruption/wrong-password/migration/secret-boundary hardening.
7. Parser/decoder fuzzing, race/static/dependency checks where practical.
8. Desktop lifecycle/runtime hardening that depends on the above.
9. Packaging/release reproducibility/provenance technical completion.
10. Only then run the consolidated end-to-end validation described in Stage 14.

### 1.2 Deferred validation policy

The old continuous `>=24h` Stage 14A soak, mandatory `>=7 consecutive days` Stage 14B window, and fixed-duration `2-3 hour` distributed-session requirement are removed as release-blocking requirements for v0.2.

Final distributed validation is **functional and evidence-based, not time-based**:

- at least two independently launched ordinary nodes/clients must first prove automatic peer discovery and synchronization with no manual IP/port/seed entry;
- at least one real reachable bootstrap route must exist for a cold client;
- additional independent nodes/clients are useful for resilience testing but are not required merely to satisfy an arbitrary session-duration quota;
- the distributed gate is passed by completing the required functional checks: automatic join, peer exchange, height/tip/chainwork convergence, transaction relay, block propagation, restart/reconnect, peer-cache persistence, bootstrap-loss/recovery and installer-upgrade preservation where applicable;
- no minimum number of hours per session is required;
- no continuous 24-hour powered-on computer requirement exists;
- no mandatory seven-day continuous powered-on requirement exists;
- longer observation remains optional evidence and may be used before Mainnet, but it is not a v0.2 completion gate.

The immediate engineering priority is therefore the unresolved P2P bootstrap/discovery path. Existing single-node mining/runtime evidence remains useful, but it does not substitute for proving that independent VALDR nodes can find and communicate with each other automatically.

## 2. Product identity and non-negotiable architecture

VALDR (VDR) is its own cryptocurrency on its own blockchain.

VALDR is **not** an ERC-20, BEP-20, Solana token or asset layered on another blockchain.

Keep:

- one canonical Go consensus implementation;
- one canonical node implementation: `valdrd`;
- Proof of Work;
- UTXO accounting;
- full block/transaction validation by user-run nodes;
- P2P v2;
- headers-first synchronization;
- chainwork fork choice and reorganization;
- bounded mempool and fees;
- encrypted wallet v2 with local signing;
- localhost-only privileged RPC by default;
- separate read-only Explorer;
- CLI/operator tooling;
- Wails v2 Desktop consuming the existing node/wallet/RPC implementation;
- user-run network operation without a mandatory project-owned paid server;
- Bitcoin/Litecoin-style automatic discovery pattern: compiled bootstrap contacts, remembered peers, peer exchange and reconnect; manual `--peer` / `--seed` is operator/debug fallback only and must never be required for an ordinary Desktop user.

Do not add merely for convenience:

- a second consensus engine;
- a cloud custody account;
- server-side private keys;
- mandatory telemetry;
- ads;
- browser-extension wallet;
- mobile app before a separate stage;
- exchange/buy/sell integration before Mainnet policy;
- automatic unsigned self-update;
- smart contracts, staking, bridge, NFT or token platform without a separate future specification.

## 3. Source hierarchy

When requirements conflict, use:

1. this Master-TZ v0.2.27;
2. explicitly frozen network/profile constants in current source code when this spec says they are preserved;
3. current product/security evidence documents;
4. older Master-TZ documents only for historical context;
5. current implementation where the specification intentionally leaves an implementation detail open.

Older specifications are not deleted. This file carries forward every unfinished mandatory item that remains relevant.

## 4. Current repository reality inherited into v0.2.14

### 4.1 Last known green baseline before Testnet2

CI runs through #358 were green on the superseded Testnet1 development baseline.

That evidence remains useful for implementation maturity, but it does **not** certify Testnet2.

### 4.2 Testnet2 migration

The active network was changed to:

- profile: `testnet2`;
- Chain ID: `valdr-testnet-2`;
- P2P port: 17333;
- RPC port: 17332 localhost;
- address prefix: `VDR1`;
- target block interval: 60 seconds;
- Testnet block subsidy: 1 Testnet VDR;
- retarget interval: 10 blocks;
- retarget measurement timespan: 540 seconds;
- retarget clamp: 135..2160 seconds;
- Testnet minimum-difficulty escape: after 10 minutes;
- Testnet PoW limit: 22 leading zero bits.

Frozen Testnet2 Genesis:

- version: 2;
- timestamp: `1790380800`;
- target: `0000031b5d43afe99ee43470e1337c3642e9d9254926038fdf6d1a2e57aaa21f`;
- nonce: `22759786`;
- message: `VALDR genesis block | valdr-testnet-2 | 2026-09-26`;
- hash: `000000a065ed224c03ff3107b2d3a073906415b347600f2e83a8874371f15485`;
- P2P magic: `900d7c51`.

Historical Testnet1 remains readable/preserved but is not the active product network.

### 4.3 Current baseline

The Testnet2 recovery/hardening wave has restored a fully green active CI baseline.

Accepted green baseline:

- commit: `20e725fb52673a25945267a544a09a14ab195033`;
- workflow: `VALDR v0.2 CI`;
- run: `#440`;
- conclusion: `success`;
- active branch: `valdr-v0.2`.

This closes the immediate R0 red-CI blocker recorded in v0.2.12. Later work must preserve a fully green active Testnet2 CI gate; any regression reopens R0.

## 5. Frozen core technical baseline inherited from v0.2-v0.2.10

### 5.1 Storage v2

Backend remains BadgerDB.

Required logical state includes:

- metadata;
- blocks;
- height mapping;
- headers;
- confirmed transaction index;
- UTXO state;
- undo data;
- migration state.

Block acceptance and reorg state changes must remain atomic.

Legacy migration stays explicit, one-way and verification-gated. Frozen v0.1 data is preserved.

### 5.2 Block/header v2

Consensus limits:

- max canonical block size: 1,000,000 bytes;
- max canonical transaction size: 100,000 bytes;
- SHA-256 block hashes;
- exact unsigned 256-bit target;
- target serialized as exactly 32 raw big-endian bytes in the header preimage;
- v2 JSON target is exactly 64 lowercase hex characters;
- target must be greater than zero and not exceed the profile PoW limit;
- v2 `difficulty uint64` remains non-consensus and zero.

Frozen v1 encoding remains unchanged for legacy v0.1.

### 5.3 Timestamp, work and fork choice

Keep:

- Median-Time-Past using previous 11 blocks;
- maximum future timestamp of local time +2 hours;
- exact integer target arithmetic;
- chainwork = floor(2^256 / (target + 1));
- active chain = valid branch with greatest cumulative chainwork;
- equal work keeps current active tip;
- reorg = common ancestor, undo disconnect, normal reconnect, atomic active-tip switch and mempool reconsideration.

### 5.4 Transactions

Legacy v0.1 transaction version 1 remains byte-compatible.

v2 networks use:

- transaction version 2;
- explicit `chain_id`;
- chain ID inside canonical signing/txid serialization;
- cross-network replay rejection;
- network-bound coinbase transaction IDs.

Private keys are used only in the local wallet/Desktop process and never sent through node RPC or P2P.

### 5.5 Fees and mempool

Keep:

- inputs >= outputs;
- fee = inputs - outputs;
- coinbase maximum = subsidy(height) + included fees;
- under-claim valid;
- 64 MiB mempool default;
- 72-hour expiry;
- Testnet minimum relay 1 val/byte;
- no RBF;
- no unconfirmed-parent relay;
- deterministic conflict rejection/eviction;
- deterministic highest-feerate miner selection;
- revalidation after chain changes.

### 5.6 P2P v2

Frame keeps:

- first four SHA-256 bytes of Chain ID as magic;
- protocol uint16;
- message type uint16;
- payload length uint32;
- four-byte SHA-256 checksum prefix;
- strict UTF-8 JSON.

Supported messages remain:

`hello, hello_ack, ping, pong, inv, get_data, block, tx, get_headers, headers, get_blocks, get_peers, peers, reject`.

Keep current protection model:

- handshake timeout;
- idle timeout/ping;
- bounded frame/message sizes;
- bounded inbound peers;
- per-IP inbound limit;
- rate limiting;
- malformed scoring/temp bans;
- bounded duplicate caches;
- routability filtering for public peer gossip.

### 5.7 Synchronization

Keep:

- recent-then-exponential block locator to Genesis;
- bounded locator;
- headers batch <=2,000;
- validate version/linkage/target/timestamp/difficulty/PoW before block-body acceptance;
- bounded body request batches;
- persist side branches;
- resume from persisted state after restart;
- select greatest chainwork.

### 5.8 Automatic P2P bootstrap and growing user network

VALDR must behave for an ordinary user like a Bitcoin/Litecoin-class desktop node: install, launch, automatically enter the same VALDR network, synchronize, and then allow wallet/mining use without typing any IP address, port, seed or shell command.

Mandatory ordinary-user behavior for every distributable Desktop build:

- zero manual P2P configuration on first start;
- the active network profile contains real compiled bootstrap contacts;
- Desktop starts the managed node and automatically attempts compiled bootstrap contacts plus its remembered peer cache;
- after the first connection, `get_peers / peers` discovery learns additional routable VALDR peers;
- learned peers are persisted and retried after restart;
- periodic outbound maintenance replenishes lost connections automatically;
- blocks, transactions and peer information relay across the connected P2P graph;
- a user who mines does not configure networking first: once the node is synchronized, mining uses the same automatically joined network;
- manual `--peer` / `--seed` remains operator/debug tooling only and is never an onboarding requirement;
- no fake seed endpoints are permitted.

Current Testnet2 infrastructure policy:

- the project owner will operate **two initial VALDR nodes** for the current stage;
- at least one of those nodes must be Internet-reachable as a bootstrap/full node before cold-client testing;
- the release profile may contain the two real endpoints once both are actually reachable; never invent or publish a placeholder seed;
- additional friends/users who install VALDR become ordinary independent network participants automatically;
- the network is not limited to the two initial nodes: peer discovery and remembered-peer reconnect must allow the topology to grow as more users join;
- ordinary Desktop nodes listen for Testnet2 P2P automatically where the OS/network permits, while retaining full outbound bootstrap/relay behavior when NAT or firewall makes inbound reachability unavailable;
- public/reachable nodes provide inbound capacity and may be added later without changing consensus.

Bootstrap contacts are discovery helpers only and never consensus authorities. Failure of one bootstrap route must not alter consensus or invalidate an already connected network.

### 5.8.1 Network-law invariant: download, launch, join

This is a project-wide invariant for VALDR-family native chains unless a later Master-TZ explicitly replaces it.

For an ordinary user, the canonical lifecycle is:

**download official client -> launch -> automatic network discovery -> connect to live peers -> verify/synchronize the native chain -> wallet/mining/relay available**

The user must not need to know or enter an IP address, port, seed, node ID, RPC endpoint, bootstrap hostname, or shell command.

Operational rules:

- the first creator-operated node is only the first participant, not a permanent consensus authority;
- no fixed topology such as "3 nodes", "4 nodes", or "10 nodes" exists in protocol design;
- bootstrap/fixed-seed/DNS-seed infrastructure exists only so a cold client can find at least one live peer;
- after first contact, peer exchange and persisted peer cache must let the graph grow from independently operated nodes;
- once an independently connected graph exists, loss of any creator-operated bootstrap node must not stop already-connected peers from validating, relaying or mining;
- zero live nodes means the P2P network is temporarily offline, but the chain state remains persisted on participant disks; when compatible nodes return, they resume the same network identity and reconcile by cumulative chainwork;
- mining is not required for P2P liveness, transaction relay or synchronization; mining is required only to produce new confirmed blocks;
- a fresh client cannot join while every possible peer/bootstrap route is offline; therefore public distribution requires redundant real discovery routes rather than a single machine;
- Testnet/public release infrastructure must use multiple independent real bootstrap routes when available and may additionally use DNS seed discovery; placeholder/fake endpoints are forbidden;
- ordinary Desktop clients listen locally for P2P and automatically attempt safe router/NAT traversal; if no inbound mapping can be established, they fall back to outbound-only participation without user configuration;
- the client must continuously retry remembered peers and bootstrap discovery so a temporarily empty network can recover automatically when any known live participant returns.

This invariant does not change VALDR consensus, Genesis, Chain ID, PoW, UTXO model, address format, subsidy or emission. It strengthens only bootstrap/discovery/liveness behavior.

### 5.8.2 Bitcoin-Core-derived connection manager baseline

VALDR adopts the proven operational networking pattern used by Bitcoin Core, adapted to VALDR's own protocol and Go implementation. This is an architectural requirement, not a requirement to copy Bitcoin Core's C++/Qt codebase.

Connection priority for an ordinary returning client:

1. previously learned/persisted peers;
2. live discovered peers learned from the P2P graph;
3. compiled fixed bootstrap seeds as fallback;
4. DNS seeds as cold-start/recovery fallback when configured.

Additional requirements:

- remembered peers are preferred before hammering fixed/DNS seeds;
- the node maintains a target of multiple outbound peers (current default target: 8) rather than relying on one permanent connection;
- bootstrap endpoints are never consensus authorities;
- fixed and DNS seeds must be real and independently reachable before inclusion;
- candidate selection/retry must avoid a single dead endpoint blocking progress;
- connection failures remain non-fatal and are retried with bounded cadence;
- peer metadata must evolve toward an addrman-like persistent database with recency/success/failure data and candidate diversity;
- DNS discovery must not be queried continuously when a healthy remembered-peer set is available;
- peer exchange remains the normal mechanism by which the live graph grows after initial contact;
- operator-facing public-node configuration is not part of normal Desktop onboarding.

This baseline is the networking reference for subsequent VALDR-family native chains unless a later Master-TZ explicitly replaces it.

### 5.8.3 Testnet2 home-node discovery and NAT traversal baseline

Testnet2 must not depend on the continued existence of a rented VPS or provider-specific server. The network must support ordinary independently operated computers behind common home routers.

Cold-start discovery remains Bitcoin-Core-derived:

1. remembered peer database;
2. live peers learned through peer exchange;
3. verified fixed seeds;
4. verified DNS seeds.

A brand-new client with an empty peer database still requires at least one live discovery route. Two completely new computers behind unrelated NATs cannot discover each other from nothing. Therefore public Testnet distribution still requires at least one real, verified bootstrap/DNS route, but that route may point to an independently operated home/full node rather than a VPS.

Ordinary Desktop networking policy:

- bind the local Testnet2 P2P listener on `0.0.0.0:17333`;
- automatically attempt router mapping in the order PCP -> NAT-PMP where supported;
- never require the ordinary user to enter an IP address, port, seed or router rule;
- when an inbound mapping succeeds, record the externally reachable endpoint and advertise it only after basic validation;
- when automatic mapping fails, continue normally as outbound-only; wallet, validation, synchronization, relay and mining must remain available;
- periodically refresh/renew temporary NAT mappings and remove mappings owned by VALDR on clean shutdown where the protocol permits;
- never advertise private, loopback, link-local or otherwise unroutable addresses;
- never compile placeholder or unverified IPs into Testnet2 `DefaultSeeds`;
- a candidate fixed/DNS seed enters the product only after independent reachability verification on the VALDR P2P port;
- dynamic home IPs should be published through a DNS name or equivalent update mechanism rather than repeatedly hard-coding changing numeric addresses.

Required diagnostics:

- listener state;
- NAT traversal protocol used: PCP, NAT-PMP, or none;
- mapping state: active / unavailable / failed;
- detected advertised endpoint when safe to show;
- outbound peer count and inbound peer count;
- last bootstrap/discovery attempt and last meaningful failure.

This section supersedes the provider-specific bootstrap assumptions in v0.2.22. It changes no consensus rule, Genesis value, Chain ID, PoW, UTXO rule, address format, subsidy or emission.

Bitcoin Core reference note for this implementation baseline:

- current Bitcoin Core uses built-in PCP/NAT-PMP port mapping and has removed UPnP support;
- VALDR follows that current model rather than adding a separate UPnP dependency;
- automatic port mapping is operational networking only and never changes consensus.

Implementation requirements for VALDR Testnet2:

- PCP is attempted first against the IPv4 default gateway on UDP/5351;
- unsupported/unavailable PCP falls back to NAT-PMP on UDP/5351;
- a successful mapping advertises only the externally assigned routable IP:port;
- mapping lifetime is renewed periodically before expiry and removal is attempted on clean shutdown;
- mapping failure is non-fatal and the node remains a valid outbound participant;
- no external port-mapping library is required for the baseline implementation; the protocol implementation is internal and testable with local UDP fixtures.

### 5.9 Wallet v2

Keep:

- no plaintext private key in wallet file;
- scrypt-derived 256-bit key;
- current parameters N=32768, r=8, p=1;
- random 16-byte salt;
- AES-256-GCM;
- fresh nonce;
- authenticated metadata;
- private file/directory permissions where supported;
- local unlock/decrypt/signing;
- no password on command line;
- zero temporary secret buffers where practical;
- no secret logging.

### 5.10 Explorer

Explorer remains a separate read-only service.

Desktop may open a local Explorer link, but must not embed the Explorer server merely to draw ordinary wallet screens.

### 5.11 Desktop process model

Keep:

- Wails v2 stable baseline;
- locally bundled frontend assets;
- TypeScript frontend;
- local encrypted wallet library;
- managed `valdrd` child process;
- optional managed `valdr-miner` in Advanced/Testnet;
- no duplicated consensus logic in Desktop;
- graceful node shutdown;
- no two processes opening the same DB;
- persisted node data across restarts;
- ordinary-user networking is automatic and requires no manual peer/seed entry;
- ordinary Desktop clients bind Testnet2 P2P on `0.0.0.0:17333` and automatically attempt PCP/NAT-PMP mapping;
- if no safe inbound mapping is available, Desktop falls back automatically to outbound-only participation;
- an externally reachable endpoint is advertised only after validation; manual public advertised-address configuration remains operator-only, not ordinary-user onboarding;
- legacy Desktop preferences created before automatic NAT traversal must not silently force a stale manual advertised endpoint; upgrade migration clears deprecated ordinary-user public-node settings and returns the client to automatic discovery/NAT mode. An operator may explicitly re-enable manual advertised-address mode afterwards;
- RPC remains localhost-only.

## 6. Complete execution staircase from the current point

The project now follows the stages below. A stage may have prior implementation evidence, but it is not closed until its current-network acceptance gate is green.

---

## Stage R0 — Restore green Testnet2 baseline

**Status:** complete / CI-verified on commit `20e725fb52673a25945267a544a09a14ab195033`, run #440.

Purpose: repair regressions introduced by the Testnet2 migration before any feature expansion.

Required work:

1. reproduce the failing Go tests from CI #359;
2. reproduce the failing Desktop backend tests;
3. locate every remaining Testnet1 assumption that affects active Testnet2 behavior;
4. fix code/tests without mutating frozen Testnet2 consensus values merely to satisfy tests;
5. verify node, miner, Desktop, Explorer, Docker/Linux scripts and release metadata all resolve to Testnet2 where the active product network is intended;
6. preserve historical Testnet1 compatibility where explicitly required;
7. run complete core tests;
8. run race tests;
9. run the mandatory platform Desktop test matrix;
10. run Docker/Linux/three-node smoke gates;
11. run Stage 13 development packaging/provenance chain.

Exit gate:

- one exact commit has the full active CI green on Testnet2;
- no test is skipped merely because it still assumes Testnet1;
- the green commit is recorded in the Stage 12/13 evidence docs.

R0 is green. Continue from the first incomplete Stage 12/13 evidence or acceptance item while preserving this gate.

---

## Stage 12A — Re-establish Testnet2 Desktop baseline

**Status:** implementation largely present / Testnet2 acceptance incomplete.

Required existing user path:

1. launch without terminal;
2. show Testnet identity;
3. choose safe node-data directory before first wallet creation;
4. explain disk/network requirements;
5. create or restore encrypted wallet;
6. start managed local node after wallet setup;
7. begin outbound-only synchronization;
8. show sync/connectivity state;
9. enter the normal application only after required initialization succeeds.

Required Simple mode:

- Dashboard/Overview;
- Wallet;
- Send;
- Receive;
- Transactions;
- Settings;
- backup/restore;
- lock/unlock/auto-lock;
- synchronization status.

Required Advanced mode:

- Mining;
- Network/Node;
- peers;
- chain/tip/chainwork;
- storage diagnostics;
- logs;
- public-node opt-in;
- diagnostics export;
- local Explorer link when a matching local Explorer is actually running.

Security:

- no telemetry by default;
- no ads;
- no remote scripts;
- no automatic clipboard reading;
- no automatic private-key export;
- irreversible-send confirmation;
- injection-safe metadata rendering;
- secret redaction;
- private-key reveal cleared when leaving Wallet/backgrounding.

Exit gate:

- all automated Stage 12 Testnet2 runtime tests green on Windows, Linux, macOS ARM64 and macOS AMD64;
- clean first-run, send/receive/history, crash recovery and restart persistence work on Testnet2.

---

## Stage 12B — Bitcoin Core-derived usability hardening

**Status:** specified / not yet accepted.

Implement in this order, one slice at a time.

### 12B.1 Synchronization detail

Show only real runtime data:

- lifecycle state;
- local height;
- best-known height when available;
- blocks remaining when calculable;
- sync percentage/state;
- peer count;
- last accepted block time;
- ETA only when reliably calculable, otherwise unknown/calculating.

While sync is incomplete, warn that balance/history/confirmations may be incomplete.

### 12B.2 Balance semantics

Show categories only if VALDR can compute them correctly:

- Spendable;
- Pending;
- Immature mining reward only after an actual maturity rule exists and is exposed correctly;
- Total only without double counting.

Do not imitate Bitcoin terminology with fabricated backend values.

### 12B.3 Transactions usability

Add:

- search by txid/address;
- direction filter;
- pending/confirmed filter;
- time filters;
- custom range;
- mined type when identifiable;
- safe CSV export of visible/filtered history.

Do not export secrets.

### 12B.4 Local address book

Add local-only contacts:

- label;
- canonical VDR address;
- create/edit/delete/copy/search;
- address validation;
- select from Send.

Do not change wallet key/address-generation architecture merely for contacts.

### 12B.5 Privacy masking

Add explicit amount masking:

- balance;
- transaction amounts;
- amount summaries.

It changes presentation only.

### 12B.6 Advanced diagnostics

Expose real fields when available:

- Desktop/Core version;
- source commit;
- profile and Chain ID;
- protocol version;
- uptime;
- local/best height;
- tip hash;
- chainwork;
- last block time;
- mempool count/size;
- peer count;
- data path/size.

Per peer when available:

- address;
- inbound/outbound;
- connection age;
- protocol;
- peer height;
- bytes sent/received;
- ping;
- last block/tx/message timestamps.

Do not invent unavailable metrics.

### 12B.7 About / Build Identity

Show:

- Desktop version;
- Core version if separately identified;
- network/profile;
- Chain ID;
- exact source commit;
- official website;
- source repository;
- license;
- release verification instructions when applicable.

### 12B.8 Deferred from Bitcoin Core

Not required in this stage:

- RBF;
- abandon transaction;
- Coin Control/manual UTXO selection;
- custom change;
- multi-recipient UI;
- payment URI;
- Sign/Verify Message;
- PSBT;
- hardware wallet;
- watch-only/blank wallet;
- pruning;
- Tor/I2P/CJDNS;
- embedded RPC console;
- peer ban UI;
- automatic port mapping;
- tray behavior;
- network traffic graph.

Each requires a separate future decision if needed.

Exit gate:

- every slice has unit/runtime coverage;
- full Stage 12 cross-platform matrix returns green after the final slice.

---

## Stage 12C — Manual Windows acceptance (owner-waived for v0.2)

**Status:** optional checklist / owner-waived as a v0.2 blocking gate on 2026-09-27; **not manually verified**.

Create a **new Testnet2 Windows candidate** from a green CI commit. The old Testnet1 candidate is historical evidence only.

Manual click-through must cover:

1. first run;
2. Dashboard;
3. Wallet;
4. Receive;
5. Send;
6. Transactions;
7. Advanced Network;
8. Advanced Mining;
9. Settings;
10. restart/recovery;
11. new v0.2.11 UX features from Stage 12B.

Mining visual checks:

- mining starts only explicitly;
- reward address is explicit;
- accepted blocks changes;
- average hashrate is numeric after work occurs;
- last solve time/hash count fields populate when supported;
- mining stops and does not silently restart.

Record:

- Windows version/build;
- source commit;
- artifact hash;
- PASS/FAIL per section;
- screenshot/log reference for every failure;
- tester date/time.

Owner decision for v0.2:

- this manual click-through is not required to enter Stage 14A;
- absence of the click-through must never be recorded as a manual PASS;
- if the checklist is later run and finds a defect, the defect still reopens implementation and requires full CI after the fix;
- Stage 12 may be frozen for v0.2 on the basis of the green automated matrix plus this explicit owner waiver.

---

## Stage 13A — Testnet2 packaging and release pipeline regeneration

**Status:** development implementation exists / Testnet2 evidence incomplete.

Required artifacts:

- Windows x64 installer;
- Windows portable ZIP;
- Linux x64 AppImage;
- Linux amd64 deb;
- macOS ARM64 DMG;
- macOS Intel/AMD64 DMG.

Every package must contain the matching Desktop, `valdrd` and `valdr-miner` where the package contract requires them.

Required release metadata:

- exact application version;
- exact git commit;
- network = Testnet2;
- Chain ID = `valdr-testnet-2`;
- protocol versions;
- OS/architecture;
- filename;
- byte size;
- SHA-256;
- vendor-signing state;
- provenance method.

Authenticity:

- SHA-256;
- canonical manifest;
- GitHub/Sigstore keyless provenance;
- clean independent verification constrained to the expected repository/workflow/ref/commit.

Windows Authenticode and Apple Developer ID/notarization remain optional future hardening if legitimately obtainable. Never fake or borrow them.

Exit gate:

- full Testnet2 development package matrix green;
- clean install/launch/uninstall checks green;
- Mainnet unavailable;
- user data survives uninstall where required;
- manifest/checksums/provenance verify from a clean environment.

---

## Stage 13B — Official website technical release integration

**Repository:** `Sheff1981/valdr-site`.

The website stays separate from `valdr-core`.

Required download behavior:

- Testnet prominently identified;
- current release version visible;
- OS suggestion without hiding alternatives;
- architecture visible;
- file size visible;
- SHA-256 visible;
- provenance/verification instructions visible;
- source link;
- system requirements;
- disk/synchronization warning;
- truthful unsigned/unnotarized disclosure;
- fail closed when verified public release metadata is absent.

No executable link may be enabled from placeholder or incomplete metadata.

Website CI must actually execute validation jobs. A zero-step/runner-allocation failure is neither PASS nor FAIL evidence.

Exit gate:

- website CI green on the exact integrated content/release metadata implementation;
- no stale Testnet1 identity appears on active download/verification pages;
- no fake release link exists.

---

## Stage 13C — Official website information architecture and content

**Status:** specified / partial site exists / full content pass incomplete.

Canonical English content is written first.

Required public structure:

- Home;
- Getting Started;
- Individuals / Using VALDR;
- Technology / How It Works;
- Wallet;
- Node;
- Mining;
- Explorer / Network;
- Developers / Docs;
- Security / What You Need to Know;
- Community / Help Build the Network;
- Story;
- FAQ;
- Download;
- Verify;
- Releases;
- Roadmap.

Content rules:

- explain what each concept means before technical detail;
- tell the user why the page exists;
- distinguish wallet, node, miner, blockchain and network;
- explain irreversible transactions and self-custody;
- explain that the current network is Testnet;
- no buy/sell/investment promise;
- no fake partner/exchange claims;
- no fake usage metrics;
- no copying Bitcoin.org text or artwork.

Required original VALDR visuals:

1. Desktop -> encrypted wallet -> local node;
2. P2P user-node mesh and initial bootstrap;
3. transaction -> mempool -> relay -> block confirmation;
4. miner -> PoW -> block -> propagation -> validation;
5. Explorer blocks/transactions;
6. real release-stable Desktop screenshots.

Language policy:

- English = canonical source;
- Russian = full maintained localization;
- French = next priority after English/Russian critical content is stable;
- more languages only with review;
- runtime machine translation is not authoritative copy.

Story:

- may use the founder's real personal philosophy of lineage/continuity, including the concept “My lineage, stand behind me”;
- present it as personal philosophy, not fabricated history;
- etymology/history claims about VALDR must be source-backed;
- avoid novelty “Viking coin” positioning.

Exit gate:

- English and Russian critical pages are complete and consistent with the released software;
- screenshots/diagrams match actual behavior;
- French may begin only after canonical English content is stable.

---

## Stage 13D — Freeze first public Testnet release candidate

**Status:** planned / sequenced after Stage 14A under v0.2.27.

The development string `0.2.0-dev` must not be published as an accepted public Testnet release.

Required:

1. choose one non-development Testnet RC application version;
2. make binary, package names, manifest and website agree;
3. build from one immutable commit;
4. rerun all cross-platform package gates;
5. verify SHA-256;
6. verify GitHub/Sigstore provenance from a clean environment;
7. freeze release notes;
8. update download metadata from the accepted manifest only;
9. publish no asset that does not map to the frozen commit.

Exit gate:

- exact RC commit and all accepted artifacts are recorded;
- the Stage 12C owner waiver is recorded truthfully (no false manual-PASS claim);
- Stage 14A distributed validation has passed on the accepted candidate line;
- website CI is green;
- release verification is independently repeatable.

---

## Stage 14A — Final independent multi-machine Testnet validation

**Status:** blocked by unresolved automatic P2P bootstrap/discovery.

Purpose: prove that independent ordinary VALDR clients can automatically find each other through the intended Bitcoin-Core-derived discovery path and then complete the required distributed functional checks. Public Testnet publication remains blocked until this functional gate passes and Stage 13D freezes a non-development RC.

Requirements:

- begin with at least two independently launched ordinary nodes/clients on the same Testnet2 chain;
- at least one real reachable initial bootstrap route exists for a cold client;
- a fresh ordinary Desktop client joins without manual IP/port/seed entry;
- peer exchange learns additional reachable peers when available;
- peer cache survives restart and remembered peers are preferred before bootstrap infrastructure;
- losing one bootstrap route does not break already connected peers or consensus;
- exact tested source/release commit is recorded.

Functional checks, with **no minimum session duration**:

- automatic peer connection with no manual IP/port/seed entry;
- chain height/tip/chainwork convergence;
- transaction creation/signing/relay and matching txid observation;
- mining on one node and block propagation/confirmation on the other;
- restart and automatic reconnect;
- learned-peer cache persistence;
- bootstrap loss and recovery;
- conflict/reorg recovery when exercised by the current test harness;
- installer upgrade preserves data.

Exit gate:

- the two-machine functional sequence in section 20.7 passes;
- no unresolved consensus split;
- no manual DB deletion/repair needed for normal recovery;
- no unresolved critical blocker;
- defects found are fixed and the full active CI is rerun green;
- exact evidence of the completed functional checks is recorded.

There is no requirement to keep the machines running for 2-3 hours, 24 hours or seven days merely to satisfy a time quota.

---

## Stage 14B — Final public/volunteer Testnet validation

**Status:** optional extension after Stage 14A for v0.2; not a seven-day blocking gate.

After Stage 14A, additional users/volunteers may:

1. download and verify the accepted Testnet client;
2. run nodes;
3. run public full nodes where inbound connectivity is available;
4. mine Testnet blocks;
5. send Testnet transactions;
6. report bugs;
7. review source;
8. help with translations/documentation.

Correct public CTA:

**Help build and test the VALDR network.**

Do not ask users to fabricate activity or download counts to influence exchanges.

Acceptance for v0.2:

- no mandatory seven consecutive day window;
- additional volunteer sessions may be accumulated for evidence, with no fixed minimum duration;
- any consensus/security blocker found reopens technical implementation and requires a full CI rerun after the fix.

---

## Stage 14C — Optional extended Testnet observation

**Status:** optional evidence / not a v0.2 blocking gate.

Longer observation remains useful before a future Mainnet decision, especially for:

- real block-time distribution;
- retarget behavior under varying hashrate;
- orphan/reorg frequency;
- peer topology;
- bootstrap resilience;
- wallet/backup incidents;
- storage growth;
- upgrade behavior;
- mining distribution;
- release/install failures.

v0.2 does not require a fixed 2-3 hour session, a continuously powered-on 24-hour computer, or a seven-day user computer. Any longer Mainnet-specific stability requirement must be decided later in the Mainnet specification using evidence available at that time.

---

## Stage 15 — Mainnet specification draft only

**Status:** planned / no launch authorization.

This stage creates a draft and evidence-based decisions. Mainnet remains disabled.

The Mainnet spec must explicitly resolve:

### Network identity

- Chain ID;
- profile name;
- Genesis timestamp/message/nonce/hash;
- P2P magic;
- P2P port;
- RPC port;
- address/network prefixes;
- bootstrap policy.

### Consensus/economics

- target block interval;
- PoW limit;
- initial difficulty/bootstrap target;
- retarget interval/formula/clamp;
- timestamp rules;
- Testnet-only min-difficulty behavior must not accidentally carry into Mainnet;
- initial block subsidy;
- emission curve;
- halving/reduction schedule;
- maximum supply or explicitly chosen alternative;
- tail emission if any;
- coinbase maturity;
- Genesis/premine/allocation policy, if any, with explicit disclosure;
- fee/min-relay defaults;
- block/transaction limits.

### Fork/reorg policy

- cumulative chainwork remains the base fork rule unless deliberately changed;
- checkpoint policy, if any;
- deep-reorg operational response;
- database recovery/verification procedures.

### Wallet/product

- Desktop Mainnet enablement;
- Testnet/Mainnet visual separation;
- backup/migration policy;
- release/upgrade policy.

### Operations/security

- public-node diversity expectations;
- incident response;
- vulnerability disclosure/security contact;
- reproducible/provenance release expectations;
- Mainnet launch/rollback policy.

Stage 15 does not authorize Mainnet.

---

# 7. Future v0.3+ staircase

The following stages are specified now so future work has an order. They remain blocked until the prior stage gates are met.

## Stage 16 — Mainnet mining architecture decision

**Status:** future / planned.

The current Testnet miner is a separate local process using node RPC. Before Mainnet, decide whether that is sufficient for the intended network.

Required decisions/tests:

- solo mining workflow;
- external miner template interface;
- pool compatibility requirement;
- whether a Stratum-like protocol is needed;
- CPU/GPU/ASIC expectations must be measured, not assumed;
- do not claim Bitcoin ASIC compatibility merely because SHA-256 is used;
- reward-address handling;
- work-template freshness;
- stale-share/block handling if pools are introduced;
- authentication/security boundary;
- DoS limits;
- miner/pool test harness.

If an external/pool protocol is added, specify it before implementation and test it independently of wallet private keys.

Exit gate:

- Mainnet mining model is frozen in the Mainnet specification;
- at least two independent miner instances can mine valid blocks against test infrastructure using the chosen interface.

## Stage 17 — Mainnet consensus/economic freeze

**Status:** future / blocked by Stage 15/16.

Turn the Stage 15 draft into an approved Mainnet consensus specification.

Rules:

- every consensus constant has a test vector;
- Genesis is not yet published as live until release candidate approval;
- no hidden premine/allocation;
- any allocation is explicit and documented;
- supply/emission math has independent tests;
- cross-network replay protection is proven;
- Mainnet addresses cannot be mistaken for Testnet by network validation.

Exit gate:

- approved spec revision;
- golden vectors;
- full unit/fuzz/integration suite green.

## Stage 18 — Security hardening and review

**Status:** technical hardening items applicable to Testnet2 are pulled forward into the current implementation wave; Mainnet-specific review remains future.

Under v0.2.12, the following implementation/test work must be completed before Stage 14 final validation. This does not authorize Mainnet.

Required technical work:

- parser/decoder fuzzing;
- DB crash/restart/corruption recovery;
- deep and repeated reorg testing;
- P2P flood/oversize/checksum/malformed testing;
- RPC exposure tests;
- wallet tamper/wrong-password/migration tests;
- Desktop lifecycle/duplicate-instance tests;
- release upgrade/rollback tests;
- dependency audit;
- static analysis;
- race tests;
- deterministic/reproducible build analysis where practical;
- external review if competent independent reviewers become available.

No claim of “audited” may be made without an actual review and report.

Exit gate:

- no known critical/high unresolved issue;
- security report/checklist stored in repository;
- full CI green.

## Stage 19 — Mainnet launch rehearsal

**Status:** future.

Use Testnet2 or an explicitly approved launch-candidate network to rehearse:

- clean node bootstrap;
- clean wallet creation;
- mining;
- transaction propagation;
- reorg;
- upgrades;
- package installation;
- website Mainnet/Testnet separation;
- disaster recovery;
- incident communication.

Do not reuse an accidental rehearsal chain as Mainnet without an explicit Genesis freeze.

Exit gate:

- documented full rehearsal passes;
- rollback/recovery procedure demonstrated.

## Stage 20 — Mainnet Genesis and release-candidate freeze

**Status:** future.

Only after approved Mainnet spec and rehearsal.

Freeze:

- exact Mainnet Genesis;
- exact Mainnet profile;
- exact release version;
- exact source commit;
- release notes;
- installers/packages;
- checksums;
- provenance;
- website Mainnet pages;
- operator/miner documentation.

Mainnet/Testnet data directories must remain distinct.

Exit gate:

- full cross-platform clean verification;
- exact artifact/commit binding;
- independent verification by at least one clean environment.

## Stage 21 — Mainnet launch

**Status:** future / not authorized by v0.2.12.

Launch only under the separately approved Mainnet specification.

Requirements:

- multiple independently operated reachable nodes;
- no mandatory central VALDR server;
- bootstrap diversity;
- publicly verifiable source/releases;
- Mainnet Explorer if operated, read-only and separable;
- miners can join using the frozen mining interface;
- incident channel/security contact active.

Do not enable exchange trading as a substitute for network stability.

## Stage 22 — Post-launch stabilization

**Status:** future.

Before exchange-integration priority:

- observe chain stability;
- verify peer bootstrap;
- verify difficulty behavior;
- monitor reorgs/orphans;
- validate wallet backups/recovery;
- publish upgrade procedure;
- fix launch blockers;
- keep release provenance verifiable.

Define and satisfy a stability window in the Mainnet launch spec.

## Stage 23 — Exchange/infrastructure integration readiness

**Status:** future / after stable Mainnet.

Prepare factual technical integration material, not marketing.

Required integration dossier:

- Mainnet Chain ID;
- Genesis;
- supported node release/version;
- P2P/operator RPC ports;
- address format;
- UTXO model;
- transaction construction/broadcast;
- deposit address handling;
- withdrawal flow;
- fee handling;
- coinbase maturity;
- reorg behavior;
- confirmation guidance based on Mainnet evidence;
- node installation;
- upgrade/rollback;
- Explorer/API references;
- source repository;
- release verification;
- security contact;
- Testnet integration procedure.

Do not publish a listing claim until an exchange confirms it.

## Stage 24 — Exchange listing applications/support

**Status:** future / non-code gate.

Only after Stage 23 and stable Mainnet.

Possible work:

- submit official listing/application forms directly where available;
- answer engineering due diligence;
- provide Testnet/Mainnet node integration support;
- provide legal/compliance information honestly where requested;
- provide public source, release provenance and network evidence;
- disclose actual adoption/network metrics without inflation.

Rules:

- no fake download/node/wallet counts;
- no fake market volume;
- no fake partners;
- no guaranteed-listing intermediaries represented as official;
- no claim that a fee guarantees listing;
- no paid service is mandatory merely because someone says so.

The project may remain operational without any centralized exchange listing.

## Stage 25 — Optional later product features

**Status:** optional / separate specs required.

Possible later work, each independently justified:

- payment URI;
- address labels/requests beyond the local contact book;
- Sign/Verify Message;
- Coin Control;
- hardware wallets/external signer;
- PSBT-like offline signing;
- pruning;
- compact blocks;
- package relay/RBF if policy changes;
- Tor/I2P;
- light client;
- mobile wallet;
- additional mining protocols;
- advanced privacy;
- smart contracts;
- token standards;
- bridges.

None of these are inherited requirements for the current VALDR release.

# 8. Website public narrative and network growth

Before Mainnet, the website must focus on verifiable product reality:

- own blockchain;
- own VDR coin;
- PoW;
- UTXO;
- user-run full nodes;
- local self-custody wallet;
- open source;
- verifiable releases;
- Testnet status.

Do not present:

- price;
- investment return;
- guaranteed scarcity before Mainnet economics are frozen;
- exchange availability before confirmed;
- fabricated history.

Community growth should come from useful participation:

- download;
- verify;
- run a node;
- mine Testnet;
- send transactions;
- report defects;
- translate;
- review docs/source.

# 9. Release/version discipline

Specification version and application version are separate.

Rules:

- Master-TZ filename never determines the application version;
- one application version source of truth must feed binary/package/manifest names;
- production Testnet/Mainnet release generation fails closed on a development version;
- every public artifact maps to one exact git commit;
- checksums and provenance are mandatory;
- OS-vendor signing is optional hardening when legitimately obtainable;
- no unsigned silent updater.

# 10. Security requirements carried forward

Mandatory through all stages:

- secrets local;
- RPC localhost by default;
- no P2P secret leakage;
- no telemetry/analytics by default;
- no ads;
- no remote runtime code for core UI;
- input validation;
- transaction confirmation;
- secret redaction;
- secure backup semantics;
- no plaintext private-key storage;
- no password in process arguments;
- bounded network parsers;
- bounded caches/queues;
- atomic state transitions;
- crash/restart tests;
- provenance verification.

# 11. Current implementation queue — exact order

The owner-waived Stage 12C rule and the functional, evidence-based Stage 14A rule in v0.2.27 are authoritative.

1. **R0 complete:** green Testnet2 baseline restored.
2. Stage 12A automated Testnet2 baseline: complete.
3. Stage 12B UX hardening + cross-platform automated matrix: complete.
4. Stage 12C manual Windows click-through: **owner-waived for v0.2; not manually verified**.
5. Stage 13A development packaging/provenance: implemented and CI-verified on the accepted development line.
6. Stage 13B website download integration: implemented and CI-verified.
7. Stage 13C critical site/content structure: implemented to the current release-candidate preparation level; website CI green.
8. **Current technical priority:** finish the real P2P bootstrap/discovery path required for ordinary zero-configuration clients.
9. Use `valdrd probe-peer` only as an operator preflight to prove that computer B can reach computer A with the real VALDR P2P v2 handshake on TCP/17333. This does not by itself satisfy automatic-join acceptance.
10. Once a genuinely reachable bootstrap route exists, add only that real route to the active Testnet2 bootstrap configuration; never compile a placeholder.
11. Run the Stage 14A two-machine functional sequence: fresh install, automatic join without manual peer/seed/IP/port entry, convergence, transaction relay while mining is off, mining on one machine, block confirmation on both, restart/reconnect, peer-cache persistence, DB verification, and installer-upgrade preservation.
12. Fix any blocker found in Stage 14A and rerun the full active CI.
13. After Stage 14A PASS, freeze a non-development Testnet RC.
14. Complete Stage 13D clean verification, update verified website release metadata, and publish the Testnet RC.
15. Stage 14B public/volunteer Testnet validation remains optional for v0.2.
16. Stage 14C longer-term observation remains optional for v0.2.
17. Draft Stage 15 Mainnet specification.
18. Only after separate approval, proceed into Stage 16+ future work.

No command may silently skip Stage 14A. A probe, local CI topology, Docker topology or one-machine run is preflight evidence only and must not be represented as distributed-session verification.

# 12. Definition of Done for the current v0.2 productization line

v0.2 productization is complete only when all of the following are true:

- Testnet2 full CI green on the exact accepted candidate;
- Storage/P2P/reorg/difficulty/fees/mempool/sync/wallet/Explorer gates green;
- Desktop Stage 12 automated acceptance green; Stage 12C manual acceptance is explicitly owner-waived and must not be described as manually passed;
- Windows/macOS/Linux packages clean-tested;
- canonical manifest/checksums/provenance verify;
- official website CI green and download path remains fail-closed until Stage 14A passes and a non-development RC is frozen;
- critical English/Russian Testnet onboarding/security/download content is accurate;
- at least two genuinely independent ordinary Testnet2 computers/clients participate in Stage 14A;
- at least one real reachable bootstrap route exists for a cold client;
- a fresh ordinary Desktop client joins without terminal work and without entering any peer, seed, IP or port;
- both clients converge on height, tip hash and cumulative chainwork;
- a transaction is created and relayed while mining is off, with the same txid observable on the receiving node;
- mining on one machine propagates the confirming block to the other machine;
- restart/reconnect and learned-peer cache persistence work without manual topology repair;
- normal restart/recovery requires no manual database deletion or repair and DB verification passes;
- installer upgrade preserves wallet and node data;
- conflict/reorg recovery gates remain green;
- exact distributed evidence is recorded and reviewed;
- there is **no minimum Stage 14A session duration or mandatory session count**; acceptance is functional and evidence-based;
- Mainnet remains disabled.

# 13. Explicit unresolved decisions

These must not be silently guessed:

- Mainnet Genesis;
- Mainnet Chain ID/ports/magic;
- Mainnet subsidy;
- emission/halving/max supply;
- premine/allocation policy;
- coinbase maturity;
- Mainnet difficulty bootstrap/retarget values;
- Mainnet min relay/fee defaults;
- Mainnet minimum-difficulty policy;
- checkpoint policy;
- final mining/pool interface;
- hardware-wallet model;
- payment URI;
- RBF/package relay;
- pruning;
- mobile/light client;
- exchange confirmation count;
- any investment/economic promise.

# 14. v0.2.11 change log

Changes relative to v0.2.10:

1. Convert the Master-TZ into a cumulative working specification so unfinished historical requirements no longer need to be reconstructed from older files.
2. Record the actual current blocker: Testnet2 migration CI is red after the formerly green Testnet1 baseline.
3. Add mandatory recovery Stage R0 before any further Desktop/site feature expansion.
4. Split current work into Stage 12A baseline, 12B UX hardening and 12C manual Windows acceptance.
5. Split release work into Stage 13A packaging/provenance, 13B technical site integration, 13C public content architecture and 13D first public Testnet RC freeze.
6. Preserve all v0.2.10 Bitcoin Core-derived UX requirements.
7. Carry forward all relevant deferred v0.2 requirements and place them into explicit future stages instead of leaving them as an unstructured appendix.
8. Define Stage 14A/14B/14C distributed Testnet progression.
9. Keep Stage 15 as Mainnet specification only.
10. Add future Stages 16-25 for mining architecture, Mainnet freeze, security review, launch rehearsal, Mainnet launch, stabilization, exchange integration/listing support and optional later features.
11. Keep paid/public project servers optional; user-run P2P remains the target network model.
12. Keep exchange listing outside the current v0.2 release and forbid fabricated adoption/market metrics.
13. Preserve current Testnet2 consensus constants unchanged.
14. Preserve old Master-TZ files as immutable history.

Benefits:

- one file now contains the working order from the current broken gate through long-term Mainnet/exchange readiness;
- unfinished historical tasks cannot be accidentally forgotten;
- future work is visible without authorizing premature implementation;
- the project can resume from “continue” by selecting the first incomplete gate.

Risks:

- the consolidated document is longer;
- future stages may require later revisions when real Testnet evidence changes assumptions;
- keeping a long roadmap accurate requires updating status after every accepted stage.

Mitigation:

- treat the “Current implementation queue” as the operational index;
- update stage status/evidence after every accepted gate;
- create a new Master-TZ version for any consensus/security-boundary change.

**No Mainnet launch, VDR sale, exchange listing or investment claim is authorized by v0.2.11.**


# 15. v0.2.13 change log

Changes relative to v0.2.12:

1. Remove the internal contradiction between the session-based Stage 14 policy and stale lower-section requirements for a mandatory continuous 24-hour or seven-day validation window.
2. Make Stage 14A the mandatory v0.2 distributed-validation gate: at least three independent sessions, approximately 2-3 hours each.
3. Make Stage 14B and Stage 14C optional evidence extensions for v0.2 rather than duration-based release blockers.
4. Update the Current implementation queue and Definition of Done to use the same Stage 14 acceptance rule.
5. Record the restored green Testnet2 baseline: commit `20e725fb52673a25945267a544a09a14ab195033`, VALDR v0.2 CI run #440, conclusion `success`.
6. Mark Stage R0 complete and move the operational queue to the first remaining Stage 12/13 evidence or acceptance item.
7. No consensus, wire, storage, wallet-format, Testnet2 network constants or Mainnet economics are changed.

Benefits:

- one unambiguous v0.2 validation gate;
- no requirement for a continuously powered-on computer for 24 hours or seven days;
- Master-TZ now matches the accepted green Testnet2 repository state.

Risks:

- shorter session-based validation provides less continuous uptime evidence than a mandatory seven-day window.

Mitigation:

- keep Stage 14C extended observation available as optional evidence;
- any consensus/security blocker reopens implementation and requires a full green CI rerun.

**No Mainnet launch, VDR sale, exchange listing or investment claim is authorized by v0.2.13.**


# 16. v0.2.14 change log

Changes relative to v0.2.13:

1. Record the project owner's explicit 2026-09-27 decision to waive Stage 12C manual Windows click-through as a v0.2 release-blocking gate.
2. Preserve truthfulness: Stage 12C is **waived, not passed**, and no manual verification may be claimed.
3. Move Stage 14A independent multi-machine Testnet validation to the next active gate.
4. Move non-development public Testnet RC freeze/publication after Stage 14A, rather than using the manual Windows click-through as the prerequisite.
5. Pin the current Stage 14A preflight candidate line to the fully green technical baseline commit `b2704ad26659e6e6de8377013dcc33c8aa8d8ee0`, CI #531.
6. Preserve all Testnet2 consensus, P2P wire, storage, wallet-format and network constants unchanged.
7. Preserve Stage 14A requirements: at least three independently launched nodes/clients, at least one real bootstrap route, three separate approximately 2-3 hour sessions, restart/bootstrap-loss/recovery/mining/transaction/peer-cache evidence, and no unresolved consensus or DB-repair blocker.
8. Keep Stage 14B/14C optional for v0.2.
9. Keep Mainnet disabled and Stage 15+ separately gated.

Benefits:

- removes the human GUI click-through as the immediate blocker, per owner instruction;
- puts the next effort into the higher-value real distributed-network validation;
- keeps public Testnet publication fail-closed until distributed evidence exists.

Risks:

- Windows-specific visual/interaction defects may remain undiscovered because the manual GUI pass was not performed;
- Stage 14A may expose installer/Desktop defects later and force another build;
- the project must avoid accidentally describing the waived check as a PASS.

No consensus or architecture change is introduced by v0.2.14.


# 17. v0.2.15 change log

Changes relative to v0.2.14:

1. Make automatic Bitcoin/Litecoin-style P2P participation a mandatory product behavior for every distributable VALDR Desktop build.
2. Remove any interpretation that an ordinary user must type a peer IP, seed, port, shell command or enable public-node mode before joining the network.
3. Define the normal flow as: install -> launch -> automatic bootstrap -> peer discovery -> synchronization -> wallet/mining use.
4. Keep manual `--peer` / `--seed` only as operator/debug overrides.
5. Record the owner's current infrastructure decision: two owner-operated initial nodes are sufficient for the current Testnet2 stage; a third project-owned server is not required.
6. Preserve Stage 14A multi-client validation: the third and later clients may be ordinary friend/volunteer installations and must join automatically.
7. Require compiled bootstrap contacts to be real. A second endpoint is added only after it exists and is reachable; placeholders are forbidden.
8. Preserve learned-peer cache, peer exchange, automatic reconnect, periodic outbound maintenance, block/transaction relay, P2P v2 and all frozen Testnet2 consensus values.
9. Preserve outbound-only operation as a valid client fallback where inbound reachability is unavailable; public full-node operation remains an operator function.
10. Mainnet remains disabled.

Benefits:

- ordinary users receive a normal cryptocurrency-client experience with no networking expertise required;
- each additional installation joins the same VALDR network and contributes validation/relay/mining through its P2P connections;
- the network can grow beyond the initial owner-operated nodes through discovery and persisted peer knowledge;
- the two-node owner infrastructure decision no longer conflicts with the working Master-TZ.

Risks:

- with only one currently reachable bootstrap endpoint, a brand-new cold client has a temporary single bootstrap dependency until the second real endpoint is available;
- outbound-only users do not add inbound connection capacity, so network resilience improves as more reachable public nodes appear;
- automatic bootstrap must never weaken RPC localhost-only or wallet-secret boundaries.

Mitigation:

- keep at least one owner bootstrap endpoint continuously reachable during the current distributed test;
- add the second real endpoint to the compiled profile as soon as it is routable;
- test cold start, peer exchange, cache persistence, bootstrap loss, reconnect, block relay and mining propagation in Stage 14A;
- never substitute fake seeds or require ordinary users to repair topology manually.

No Testnet2 consensus, Genesis, wire-format, PoW, UTXO or wallet-format constant is changed by v0.2.15.


# 18. v0.2.16 change log

Clarification of the network model after direct review of current Bitcoin Core peer discovery:

1. The compiled seed/bootstrap list is **not** a registry of all VALDR users and must never be maintained as one.
2. Only a small set of stable initial-contact mechanisms belongs in the shipped network profile (fixed bootstrap endpoints and, later if deployed, DNS seeds).
3. Nodes learned after bootstrap are discovered dynamically through peer exchange, kept in the local learned-peer database/cache, and retried automatically.
4. New ordinary installations are never added manually to the program or Master-TZ. They join the network through bootstrap and discovery.
5. Reachable public nodes may advertise routable listening addresses and become candidates that other nodes can learn and connect to.
6. Nodes behind NAT/firewalls can still be full validating/mining participants using outbound connections even when they are not usable as inbound bootstrap targets.
7. Peer discovery responses must include suitable learned routable peers, not only the responder's currently connected sockets, so knowledge can propagate beyond one hop.
8. Learned peer state must survive normal restart. Periodic persistence is preferred in addition to clean-shutdown persistence.
9. The user experience remains: install -> launch -> automatic network join -> sync -> mine/use wallet. No peer IP/port/seed entry is part of normal onboarding.
10. Current owner infrastructure remains two machines for this stage; only actually reachable bootstrap endpoints are compiled. The user population is unlimited and discovered dynamically.

This revision changes discovery/operational behavior only. It does not change Testnet2 Genesis, Chain ID, P2P magic, PoW, UTXO, consensus validation, wallet format or emission parameters.


# 19. v0.2.17 change log

1. Ordinary VALDR Desktop nodes now listen on the active P2P port automatically instead of forcing `--outbound-only`.
2. This does not require users to enter an advertised public address. Explicit advertised-address mode remains Advanced/operator functionality.
3. Windows installer owns an inbound firewall rule for the bundled `valdrd.exe` on Testnet2 TCP port 17333 so normal installation does not require PowerShell.
4. NAT/firewall-unreachable clients remain valid participants through outbound connections; receiving and relaying transactions does not require local mining.
5. Transaction lifecycle is explicitly Bitcoin-style: broadcast -> peer mempools / pending -> inclusion by any miner -> confirmed.
6. No consensus, Genesis, PoW, UTXO, wallet format, emission or chain-ID value changes.


# 20. Bitcoin Core-class baseline completion mandate (v0.2.18)

The project owner requires the **complete practical Bitcoin-Core-class base**, adapted to VALDR rather than a cosmetic subset. This is an operational/client baseline, not a byte-for-byte Bitcoin consensus clone.

## 20.1 Mandatory full-node behavior

VALDR must provide, before the current Testnet milestone is considered complete:

- independent full validation of every accepted block and transaction;
- UTXO validation and persistent UTXO state;
- cumulative-chainwork fork choice and safe reorganization with undo data;
- headers-first synchronization and restart-resume;
- automatic bootstrap, remembered peers, peer exchange, reconnect and outbound replenishment;
- inbound P2P listening where locally possible without ordinary-user manual configuration;
- transaction relay, block relay and peer inventory propagation;
- bounded mempool with admission policy, fee policy, expiry and deterministic mining selection;
- mempool persistence across clean restart, with revalidation against the active chain on reload;
- transaction conflict/double-spend state surfaced to wallet/history instead of silently disappearing;
- peer diagnostics, bans/protection, malformed-frame limits and connection health reporting;
- no dependence on a project-owned central server for validation, wallet operation or mining.

## 20.2 Mandatory wallet behavior

Desktop wallet must provide:

- local encrypted private keys and local signing;
- create, restore, lock, unlock and auto-lock;
- multiple independent local wallets;
- receive address display/copy/QR;
- send with pre-send validation and fee preview;
- deterministic change handling;
- correct spendable/pending/total semantics without double counting;
- transaction history for sent, received, self-transfer and mined rewards;
- explicit transaction states: pending, confirmed and conflicted/reorged where applicable;
- confirmation count, txid, fee, timestamp, block height and block hash;
- transaction search/filter/export without secrets;
- local address book;
- wallet rescan/rebuild of wallet history from the local blockchain;
- encrypted backup/restore with verification before activation;
- history depth sufficient that mining rewards cannot hide ordinary transfers.

## 20.3 Mandatory Bitcoin-style transaction lifecycle

The user experience must be:

1. wallet signs locally;
2. local node validates and admits the transaction to mempool;
3. transaction is relayed to connected peers immediately;
4. receiving wallet can show the transaction as pending / 0 confirmations without running a miner;
5. any miner in the VALDR network may include it in a valid block;
6. block relay changes the transaction to confirmed / 1+ confirmations on all synchronized nodes;
7. reorg/conflict changes the wallet state correctly and predictably.

Local mining is never required to *receive or see* a valid relayed transaction.

## 20.4 Mandatory fee and mempool usability

Before release-candidate freeze:

- expose the effective relay fee and the exact fee selected for a send;
- reject absurd/excessive fee choices client-side where possible;
- maintain fee-rate-based transaction selection for miners;
- add a safe fee estimation path based on observed VALDR block/mempool conditions once enough data exists;
- fall back explicitly when there is insufficient estimation data; never fabricate an estimate;
- surface mempool count/size and transaction pending state in Desktop diagnostics.

## 20.5 Mandatory recovery and persistence

- peer cache persists and is retried after restart;
- mempool persists across clean restart and is fully revalidated;
- blockchain/UTXO/reorg state survives crash/restart;
- wallet history can be rebuilt from chain data;
- interrupted synchronization resumes from persisted state;
- installer upgrade must preserve wallet and node data.

## 20.6 Deliberate VALDR differences from Bitcoin

The phrase “Bitcoin Core-class baseline” does **not** silently change VALDR consensus economics or identity.

Preserve unless a later explicit consensus revision changes them:

- VALDR Chain ID and address format;
- Testnet2 Genesis;
- VALDR block interval and PoW parameters;
- VALDR subsidy/emission rules;
- VALDR transaction format;
- current no-RBF policy;
- current no-unconfirmed-parent-relay policy;
- no Bitcoin Script compatibility requirement;
- no Lightning, smart contracts, tokens or exchange integration.

Adding Bitcoin's exact coinbase-maturity rule, RBF policy, script language, monetary schedule or other consensus/policy semantics would be a separate explicit Master-TZ change and may require a Testnet reset.

## 20.7 Release gate

Do not call the Windows EXE a completed Bitcoin-Core-class VALDR client until the following real two-machine sequence passes:

- fresh install on independent machines;
- automatic peer connection with no manual IP entry;
- synchronized height/tip/chainwork;
- send while mining is off;
- receiver sees the same txid as pending;
- enable mining on only one machine;
- both machines observe the same confirmed transaction and block;
- restart both;
- peers reconnect, chain remains consistent, wallet history remains intact;
- conflict/reorg and recovery tests pass;
- installer upgrade preserves data.

Status terms remain strict: implemented != CI-verified != distributed-session verified.


# 21. v0.2.27 change log

Changes relative to v0.2.26:

1. Remove stale lower-section text that still required three clients, three sessions and approximately 2-3 hours despite the newer functional Stage 14A policy already stated in v0.2.26.
2. Make the blocking distributed gate consistently require two genuinely independent computers/clients and no arbitrary minimum duration or session count.
3. Preserve automatic zero-configuration ordinary-user joining as mandatory: manual `--peer`, `--seed`, IP and port entry remain operator/debug fallbacks only.
4. Define `valdrd probe-peer` as a real-protocol reachability preflight for TCP/17333, not as proof of automatic bootstrap.
5. Keep the immediate technical blocker explicit: Testnet2 currently needs at least one genuinely reachable real bootstrap route before a fresh ordinary client can automatically join.
6. Align the Current implementation queue and Definition of Done with the two-machine sequence already specified in Stage 14A and section 20.7.
7. Restore the rule that historical Master-TZ revisions are immutable; corrections to working requirements go into this new revision rather than rewriting v0.2.15.
8. No Testnet2 consensus, Genesis, wire-format, PoW, UTXO, wallet-format or emission parameter changes are introduced.

Benefits:

- one unambiguous Stage 14A rule;
- the user's two-computer test setup is sufficient for the blocking distributed gate;
- no wasted hours are required merely to satisfy a timer;
- reachability preflight and automatic-user bootstrap are clearly separated.

Risks:

- with no real compiled bootstrap endpoint, a cold ordinary client still cannot satisfy the zero-configuration join requirement;
- a two-computer test does not prove alternate-third-peer resilience.

Mitigation:

- first prove the real TCP/17333 VALDR handshake between the two computers;
- compile only a real reachable bootstrap route after it exists;
- keep third-node/volunteer resilience as optional Stage 14B/14C evidence.

No Mainnet launch, VDR sale, exchange listing or investment claim is authorized by v0.2.27.
