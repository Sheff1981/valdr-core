# VALDR / CONSOLIDATED MASTER TECHNICAL SPECIFICATION v0.2.11

**Full inherited backlog + execution staircase + future roadmap**  
**Date:** 26 September 2026  
**Status:** active master implementation baseline  
**Previous baseline:** `docs/VALDR_Master_TZ_v0.2.10.md`  
**Active branch:** `valdr-v0.2`

> v0.2.11 replaces v0.2.10 as the single working master specification. Historical specifications v0.2 through v0.2.10 remain immutable evidence of prior decisions. Frozen legacy v0.1 remains `release/valdr-devnet-v0.1`.

> This revision is intentionally cumulative. It pulls forward unfinished requirements from earlier specifications, preserves the current implemented architecture, adds the Bitcoin Core v31.1 UX decisions already approved in v0.2.10, and defines the future work in a strict gated sequence. The project must work from this file first instead of reconstructing the plan from older documents.

## 1. Working rule

VALDR development follows one sequence:

**specification -> implementation -> build -> tests -> runtime verification -> evidence -> commit -> next gate**

Rules:

1. Do not skip a failed gate.
2. Do not declare a stage complete because code merely compiles.
3. Distinguish:
   - **implemented** — code exists;
   - **CI-verified** — automated gate passed on the relevant active network/version;
   - **manually verified** — required human-visible acceptance passed;
   - **stable in distributed use** — soak criteria passed;
   - **planned** — specified but not implemented.
4. A newer network/profile reset invalidates acceptance evidence that depended on the superseded profile.
5. Consensus, wire, storage, wallet-format or Mainnet-economic changes require a new Master-TZ revision.
6. UX-only changes may be implemented without a new spec only when they do not change mandatory behavior or security boundaries.
7. No public release, Mainnet launch or exchange-listing work may be represented as complete before its prerequisites are green.

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
- Bitcoin/Litecoin-style discovery pattern: remembered peers, peer exchange, optional fixed/DNS seeds and manual peers.

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

1. this Master-TZ v0.2.11;
2. explicitly frozen network/profile constants in current source code when this spec says they are preserved;
3. current product/security evidence documents;
4. older Master-TZ documents only for historical context;
5. current implementation where the specification intentionally leaves an implementation detail open.

Older specifications are not deleted. This file carries forward every unfinished mandatory item that remains relevant.

## 4. Current repository reality at v0.2.11 creation

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

### 4.3 Current immediate blocker

The first CI run after the Testnet2 runtime/release migration (#359) failed in:

- the main Go test gate;
- Wails Desktop backend tests on Linux;
- Wails Desktop backend tests on Windows;
- Wails Desktop backend tests on both macOS architectures.

Subsequent documentation-only runs #360-#362 remained red because the same underlying Testnet2 test regression was still present.

**Therefore the next implementation task is not Stage 12 UX expansion. The next task is to restore a green Testnet2 baseline.**

No later stage may be called complete while this gate is red.

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

### 5.8 Peer bootstrap and no mandatory paid server

VALDR does not require a central server or mandatory project-paid seed/full nodes.

Required behavior:

- bounded persistent learned-peer cache;
- reconnect to learned peers after restart;
- optional real fixed seed endpoints;
- optional real DNS seeds;
- manual `--peer` / `--seed` overrides;
- peer exchange;
- periodic outbound connectivity maintenance;
- seeds are discovery helpers only and never consensus authorities;
- no fake seed endpoints.

A cold Internet node still needs at least one reachable initial contact. It may be any independently operated public VALDR node or volunteer seed.

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
- outbound-only default;
- public full-node mode only by explicit Advanced opt-in;
- RPC remains localhost-only.

## 6. Complete execution staircase from the current point

The project now follows the stages below. A stage may have prior implementation evidence, but it is not closed until its current-network acceptance gate is green.

---

## Stage R0 — Restore green Testnet2 baseline

**Status:** mandatory immediate work / not complete.

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

Do not begin Stage 12B before R0 is green.

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

## Stage 12C — Manual Windows acceptance

**Status:** mandatory / not complete for Testnet2.

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

Exit gate:

- manual PASS;
- any discovered defect is fixed;
- full CI is rerun green after the last fix.

Only then may Stage 12 be frozen complete.

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

**Status:** planned / blocked by Stages R0, 12 and 13A-C.

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
- Stage 12 manual PASS exists;
- website CI is green;
- release verification is independently repeatable.

---

## Stage 14A — Independent multi-user Testnet soak

**Status:** planned / not started.

No mandatory paid project server.

Requirements:

- at least three independently launched nodes/clients on the same Testnet2 chain;
- at least one real reachable initial bootstrap route for a cold client;
- peer exchange learns additional peers;
- peer cache survives restart;
- losing one bootstrap route does not break already connected peers or consensus;
- clients may be ordinary user PCs or volunteer public nodes.

Minimum first soak:

- >=24 hours.

Monitor:

- chain height/tip/chainwork;
- block intervals;
- difficulty changes;
- reorg behavior;
- peer count/diversity;
- bootstrap success;
- mempool;
- disk;
- restart;
- node crashes;
- Desktop sync;
- miner behavior;
- installer/download issues.

Exit gate:

- no consensus split;
- no manual DB repair;
- blockers fixed and full CI rerun.

---

## Stage 14B — Public distributed Testnet

**Status:** planned / not started.

After Stage 14A:

1. publish accepted Testnet client;
2. invite users to download and verify;
3. ask users to run nodes;
4. ask volunteers who can accept inbound traffic to run public full nodes;
5. ask users to mine Testnet blocks;
6. send Testnet transactions;
7. report bugs;
8. review source;
9. help with translations/documentation.

Correct public CTA:

**Help build and test the VALDR network.**

Do not ask users to fabricate activity or download counts to influence exchanges.

Minimum acceptance window:

- >=7 consecutive days;
- no consensus split;
- no manual DB repair;
- restart/bootstrap recovery remains functional.

Exit gate:

- documented Testnet report;
- actual observed issues and fixes recorded;
- no critical unresolved security/consensus bug.

---

## Stage 14C — Extended Testnet observation before Mainnet decision

**Status:** future planning gate.

The seven-day gate proves initial distributed stability but is not, by itself, evidence that Mainnet is ready.

Before a Mainnet go/no-go decision, the Stage 15 Mainnet spec must define a longer stability requirement. v0.2.11 recommends a substantially longer distributed observation period (for example, 30 days) but does not freeze the exact Mainnet requirement.

Measure:

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

## Stage 18 — Pre-Mainnet security hardening and review

**Status:** future.

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

**Status:** future / not authorized by v0.2.11.

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

The next work must follow this order:

1. **R0:** repair Testnet2 CI failures and obtain one fully green Testnet2 commit.
2. Update Stage 12/13 evidence docs to that green Testnet2 commit.
3. Complete Stage 12A Testnet2 automated baseline verification.
4. Implement Stage 12B UX hardening one slice at a time.
5. Run full cross-platform CI.
6. Build a fresh Testnet2 Windows QA candidate.
7. Complete Stage 12C manual Windows acceptance.
8. Freeze Stage 12.
9. Regenerate Stage 13A Testnet2 package/provenance evidence.
10. Make official website CI execute and pass.
11. Complete Stage 13B download integration.
12. Complete Stage 13C page/content structure and critical EN/RU content.
13. Freeze a non-development Testnet release candidate.
14. Complete Stage 13D clean verification and publish Testnet.
15. Run Stage 14A private/multi-user soak.
16. Run Stage 14B distributed public Testnet >=7 days.
17. Collect Stage 14C longer-term evidence.
18. Draft Stage 15 Mainnet specification.
19. Only after approval, proceed into Stage 16+ future work.

No “next stage” command may jump over the first incomplete item in this list unless the project owner explicitly instructs otherwise.

# 12. Definition of Done for the current v0.2 productization line

v0.2 productization is complete only when all of the following are true:

- Testnet2 full CI green;
- Storage/P2P/reorg/difficulty/fees/mempool/sync/wallet/Explorer gates green;
- Desktop Stage 12 automated + manual acceptance green;
- v0.2.11 UX requirements green;
- Windows/macOS/Linux packages clean-tested;
- canonical manifest/checksums/provenance verify;
- official website CI green;
- official website download path fail-closed and accurate;
- critical English/Russian Testnet onboarding/security/download content is accurate;
- ordinary user can install/use Testnet without terminal;
- at least three independently operated Testnet clients participate;
- cold client can bootstrap through at least one real route, learn peers, persist and reconnect;
- public distributed Testnet remains stable for at least seven days without consensus split/manual DB repair;
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
