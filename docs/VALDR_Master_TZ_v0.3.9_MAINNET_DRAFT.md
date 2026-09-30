# VALDR / MASTER TECHNICAL SPECIFICATION v0.3.9-DRAFT

**Mainnet engineering, launch and stabilization baseline**  
**Date:** 29 September 2026  
**Status:** DRAFT — current Mainnet engineering baseline; Mainnet disabled  
**Previous Mainnet engineering baseline:** `docs/VALDR_Master_TZ_v0.3.8_MAINNET_DRAFT.md`  
**Inherited Testnet2 baseline:** `docs/VALDR_Master_TZ_v0.2.14.md`  
**Active branch:** `valdr-v0.2`

> v0.2.14 replaces v0.2.13 as the single working master specification. Historical specifications v0.2 through v0.2.13 remain immutable evidence of prior decisions. Frozen legacy v0.1 remains `release/valdr-devnet-v0.1`.

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
10. Historical stage numbers are requirement identifiers, not permission barriers. Where this v0.2.14 explicitly defers or waives an acceptance gate, later validation work may proceed without falsely claiming that the waived check was performed.
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

The old continuous `>=24h` Stage 14A soak and mandatory `>=7 consecutive days` Stage 14B window are removed as release-blocking requirements for v0.2.

Final distributed validation is session-based:

- at least three independently launched nodes/clients;
- at least one real reachable bootstrap route;
- at least three separate validation sessions;
- each session approximately 2-3 hours;
- sessions should include restart, bootstrap-loss/recovery, mining, transactions, peer exchange, reorg observation and installer/Desktop use where applicable;
- no continuous 24-hour powered-on computer requirement;
- no mandatory seven-day continuous powered-on requirement;
- any optional longer observation remains useful evidence but is not a v0.2 completion gate.

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

1. this Master-TZ v0.2.14;
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

**Status:** planned / sequenced after Stage 14A under v0.2.14.

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

**Status:** **ACTIVE NEXT GATE** under v0.2.14.

Purpose: validate the completed technical milestone on real independent computers/clients after Core, P2P, storage, RPC, wallet, Desktop and packaging hardening are complete. Stage 14A may run on the exact green development candidate pinned to CI #531 (`b2704ad26659e6e6de8377013dcc33c8aa8d8ee0`); public Testnet publication remains blocked until Stage 14A passes and Stage 13D freezes a non-development RC.

Requirements:

- at least three independently launched nodes/clients on the same Testnet2 chain;
- at least one real reachable initial bootstrap route for a cold client;
- peer exchange learns additional peers;
- peer cache survives restart;
- losing one bootstrap route does not break already connected peers or consensus;
- clients may be ordinary user PCs or volunteer public nodes;
- exact tested source/release commit is recorded.

Validation schedule:

- at least three separate sessions;
- each session approximately 2-3 hours;
- no continuous 24-hour requirement;
- computers may be shut down between sessions.

Each session should exercise as applicable:

- chain height/tip/chainwork convergence;
- mining from at least two independent miner instances/operators;
- block propagation and retarget observation;
- transaction creation/signing/relay/confirmation;
- peer discovery and learned-peer cache;
- bootstrap loss and recovery;
- restart persistence and DB verification;
- mempool behavior;
- Desktop synchronization;
- installer/package startup;
- wallet backup/restore;
- crash/error reporting.

Exit gate:

- no unresolved consensus split;
- no manual DB deletion/repair needed for normal recovery;
- no unresolved critical blocker;
- defects found are fixed and the full active CI is rerun green;
- session evidence is recorded.

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
- evidence from additional 2-3 hour sessions may be accumulated;
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

v0.2 does not require a continuously powered-on 24-hour or seven-day user computer. Any longer Mainnet-specific stability requirement must be decided later in the Mainnet specification using evidence available at that time.

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

The owner-waived Stage 12C rule and Stage 14-first validation order are authoritative for v0.2.14.

1. **R0 complete:** green Testnet2 baseline restored.
2. Stage 12A automated Testnet2 baseline: complete.
3. Stage 12B UX hardening + cross-platform automated matrix: complete.
4. Current post-hardening technical baseline: commit `b2704ad26659e6e6de8377013dcc33c8aa8d8ee0`, VALDR v0.2 CI #531, SUCCESS.
5. Stage 12C manual Windows click-through: **owner-waived for v0.2; not manually verified**.
6. Freeze Stage 12 for v0.2 using automated evidence + explicit owner waiver.
7. Stage 13A development packaging/provenance: complete on CI #531.
8. Stage 13B website download integration: implemented and CI-verified.
9. Stage 13C critical site/content structure: implemented to the current release-candidate preparation level; website CI green.
10. **Run Stage 14A now** on at least three independently launched clients/nodes: at least three separate sessions, approximately 2-3 hours each, using the exact accepted candidate line.
11. Fix any blocker found in Stage 14A and rerun full active CI.
12. After Stage 14A PASS, freeze a non-development Testnet RC.
13. Complete Stage 13D clean verification, update verified website release metadata, and publish the Testnet RC.
14. Stage 14B public/volunteer Testnet validation remains optional for v0.2.
15. Stage 14C longer-term evidence remains optional for v0.2.
16. Draft Stage 15 Mainnet specification.
17. Only after separate approval, proceed into Stage 16+ future work.

No command may silently skip Stage 14A. The Stage 12C exception is explicit owner direction recorded in this revision.

# 12. Definition of Done for the current v0.2 productization line

v0.2 productization is complete only when all of the following are true:

- Testnet2 full CI green;
- Storage/P2P/reorg/difficulty/fees/mempool/sync/wallet/Explorer gates green;
- Desktop Stage 12 automated acceptance green; Stage 12C manual acceptance is explicitly owner-waived and must not be described as manually passed;
- v0.2.11 UX requirements green;
- Windows/macOS/Linux packages clean-tested;
- canonical manifest/checksums/provenance verify;
- official website CI green;
- official website download path fail-closed and accurate until Stage 14A passes and a non-development RC is frozen;
- critical English/Russian Testnet onboarding/security/download content is accurate;
- ordinary user can install/use Testnet without terminal;
- at least three independently operated Testnet clients participate;
- cold client can bootstrap through at least one real route, learn peers, persist and reconnect;
- Stage 14A completes at least three separate validation sessions of approximately 2-3 hours each, with no unresolved consensus split, no normal-recovery requirement for manual DB deletion/repair, and session evidence recorded;
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


---

# 9. Mainnet v0.3.7 controlling appendix

This appendix is authoritative for Mainnet work on branch `experiment/mainnet-randomx-prototype`.
Where it conflicts with inherited v0.2.14 wording, this appendix controls Mainnet planning.
Testnet2 frozen values remain unchanged unless a later approved Master-TZ explicitly changes them.

## 9.1 Current Mainnet decisions and candidates

Product identity:
- VALDR is its own cryptocurrency and own blockchain.
- Mainnet is not enabled.
- Genesis is not frozen.

Economic candidate:
- target block interval: 600 seconds;
- initial block subsidy: 50 VDR;
- halving interval candidate: 262,800 blocks, approximately five years;
- reward path: 50 -> 25 -> 12.5 -> 6.25 -> ... using exact integer consensus arithmetic;
- premine candidate: zero;
- tail emission candidate: zero;
- coinbase maturity candidate: 100 blocks;
- exact terminal issuance must be derived and golden-tested before freeze.

PoW candidate:
- RandomX, based on upstream RandomX v1.2.3 prototype evidence;
- SHA-256 remains the inexpensive canonical block identifier;
- RandomX is the expensive Mainnet PoW hash;
- validity: unsigned 256-bit pow_hash <= target;
- Testnet2 PoW is not changed by this candidate.

RandomX seed candidate:
- epoch length: 2,048 blocks;
- lag: 64 blocks;
- heights below 64 use fixed epoch-0 seed;
- otherwise seed_height = floor((height - 64) / 2048) * 2048;
- seed_key = block_id(seed_height);
- epoch-0 seed:
  `0fca8879c33460a767246a02b156794ec9958113261e04db0b3a9e66fe6febb9`.

Mining-blob candidate:
- do not introduce a second header serialization;
- `block_id = SHA256(existing canonical v2 header)`;
- `pow_blob = "VALDR/RANDOMX/POW/V1\\x00" || existing canonical v2 header`;
- `pow_hash = RandomX(seed_key, pow_blob)`;
- prototype sample canonical header = 235 bytes;
- prototype sample mining blob = 256 bytes;
- sample block ID =
  `9d7838b61a5e10c252d690149eec1a2fe80e32617fd29ebacaaf6d9903144a8a`;
- sample mining-blob SHA-256 test oracle =
  `474a5ca610d1573c3f2b0cd0d634448ee0a9a909af40ea0773b9b780ccbf29bf`;
- the SHA-256 mining-blob digest is only a serialization oracle, not Mainnet PoW.

DAA candidate:
- per-block LWMA-style adjustment;
- target T = 600 seconds;
- window N = 30;
- MTP-11;
- Mainnet future-time candidate: +30 minutes;
- solvetime clamp inside DAA: 6*T = 3600 seconds;
- out-of-order effective timestamp: previous effective timestamp + 1 second;
- no permanent Mainnet minimum-difficulty escape;
- exact integer/big-int arithmetic only;
- PowLimit and InitialTarget remain unresolved pending calibration and simulation.

DAA candidate formula:
- solve_i = effective_timestamp_i - effective_timestamp_(i-1), capped at 6T;
- L = sum(i * solve_i);
- lower bound L = max(L, N*N*T/20);
- work(target) = floor(2^256 / (target + 1));
- avgWork = floor(sum(work_i) / N);
- nextWork = floor(avgWork * N * (N+1) * T * 99 / (200 * L));
- derive target from `nextWork` using the exact consensus candidate inverse:
  `nextTarget = floor(2^256 / nextWork) - 1`, with `nextWork > 0`;
- then cap by Mainnet PowLimit: `nextTarget = min(nextTarget, PowLimit)`;
- all divisions are integer floor divisions using exact big-integer arithmetic; floating-point arithmetic is forbidden;
- startup missing oldest slots use solve=T and InitialMainnetWork.

RandomX implementation evidence:
- official upstream RandomX vector is verified;
- deterministic prototype hash is identical across Windows, Linux and macOS;
- full-memory benchmark path passed on all three CI platforms;
- historical CI 1-thread measurements were approximately:
  - macOS ARM64: 68.2954 H/s;
  - Ubuntu: 541.106 H/s;
  - Windows: 387.989 H/s;
- these CI numbers are engineering evidence only and are not sufficient to freeze launch difficulty;
- real-CPU benchmark pack is implemented and CI-verified.

Difficulty calibration evidence rule:
- project owner is not required to possess five physical computers;
- at least one real user-owned computer must run the benchmark;
- evidence must cover at least three CPU performance classes:
  ordinary/low-power laptop, mainstream desktop, higher-performance desktop;
- classes may be covered by a combination of real user hardware, reproducible independent benchmarks and controlled CI/cloud hardware with recorded CPU identity;
- final InitialDifficulty must be conservative and pass DAA simulations;
- M5 calibration evidence is now CI-verified, including one real user-owned i5-8265U measurement and controlled reproducible CPU-class evidence;
- exact candidate InitialWork = 251233;
- exact candidate InitialTarget =
  `000042c78dcd2f09f6673e99417b7b7d0dbb0ddcf732f964dc73fce1c6b50768`;
- selected PowLimit policy candidate = 4x easier than InitialTarget in work terms;
- exact candidate PowLimitWork = 62808;
- exact candidate PowLimitTarget =
  `00010b1e7ce2d1034b183a2be3b26504779a86ab0ed6197fb90be6d3c07b200c`;
- DAA comparison and independent exact-arithmetic reproduction are CI-verified;
- these values remain **pre-freeze candidates**, not frozen consensus, because the non-skippable staircase still requires earlier gates M0-M2 to be closed first.

## 9.2 Non-skippable execution law

VALDR development follows:

**specification -> implementation -> build -> automated tests -> cross-platform validation -> distributed validation -> evidence -> freeze -> next stage**

A later stage may be researched in parallel only when it does not bypass the acceptance gate of the current stage.

No stage may be marked complete merely because it is slow, inconvenient, expensive or manually difficult.

If the project owner asks to skip a mandatory gate, the engineering response must identify the blocker and keep the stage open. A gate may be simplified only by a new Master-TZ revision that records:
- exact changed requirement;
- reason;
- benefits;
- risks;
- replacement evidence;
- historical predecessor.

Status vocabulary:
- planned;
- implemented;
- CI-verified;
- manually verified;
- distributed verified;
- rehearsal passed;
- frozen;
- launched;
- stabilized.

Do not use "ready", "complete" or "released" where a stronger required state is still missing.

## 9.3 Complete Mainnet-to-exchange staircase

### M0 — Testnet evidence prerequisite
- keep Testnet2 consensus green;
- preserve the exact frozen v0.2.14 Testnet2 consensus/network baseline;
- for the Mainnet engineering staircase, the earlier requirement for three independently operated physical Testnet clients is superseded by this revision;
- replacement evidence is:
  1. full automated Testnet2 core regression on the exact Mainnet candidate branch;
  2. cross-platform Windows/Linux/macOS build/test coverage;
  3. automated multi-node Testnet2 smoke/recovery evidence exercising at least three node processes;
  4. Stage14A evidence-tooling self-tests remain green;
  5. at least one real user-owned Windows Testnet2 installation/session must demonstrate node startup, mining progress, persisted wallet/balance state and restart recovery;
  6. no unresolved consensus split or consensus-critical Testnet2 defect.
- the replacement evidence is sufficient only for M0 of the Mainnet engineering staircase; it does not retroactively claim the historical v0.2.14 Stage14A three-machine gate was passed.
- M0 is not frozen until all replacement evidence above is recorded.

### M1 — Mainnet network identity
Freeze Chain ID, profile, P2P magic, ports, address prefixes, protocol version, bootstrap/discovery, data-directory separation and Mainnet/Testnet visual separation.
Gate: replay/network-separation tests.

### M2 — Monetary policy
Freeze subsidy, halving, exact issuance, premine/allocation, tail emission, coinbase maturity and fee treatment.
Gate: full emission golden vectors and boundary/overflow tests.

### M3 — PoW
Freeze RandomX version, mining blob, seed schedule, target comparison, reorg behavior and dependency/build policy.
Gate: official RandomX tests, identical VALDR vectors across supported OSes and production validator integration tests.

### M4 — Difficulty Adjustment Algorithm
Freeze LWMA parameters, startup behavior, MTP/future-time rules, clamps and PowLimit interaction.
Gate: golden vectors plus stable and 2x/10x/100x hashrate increase/decrease simulations.

### M5 — InitialDifficulty / InitialTarget / PowLimit
Use at least one real user-owned machine plus evidence covering three CPU classes.
Gate: exact 256-bit InitialTarget and PowLimit with independent reproduction.

### M6 — Coinbase maturity / UTXO
Implement consensus maturity, wallet immature balance, premature-spend rejection and reorg handling.
Gate: maturity and reorg tests.

### M7 — Mainnet transaction / wallet separation
Verify address rules, replay rejection, signing, encrypted wallet, backup/restore and secret boundaries.
Gate: clean wallet create/send/receive/backup/restore tests.

### M8 — P2P and sync
Verify handshake separation, discovery, peer persistence, headers-first sync, chainwork/reorg and hostile-input limits.
Gate: multi-node convergence and restart tests.

### M9 — Storage / recovery
Verify atomic acceptance/reorg, UTXO/undo consistency, crash/restart, corruption detection and data-dir separation.
Gate: recovery matrix green.

### M10 — RPC / CLI / Desktop
Verify localhost security, authentication where exposed, input limits, diagnostics, Mainnet/Testnet separation, package lifecycle and duplicate-instance handling.
Gate: cross-platform functional tests plus manual acceptance.

### M11 — Mining product
Solo miner must use explicit reward address, fresh templates, stale-work handling, thread/memory controls and safe lifecycle.
At least two independent miner instances must produce valid blocks.
Pool/Stratum is optional for first Mainnet unless separately made mandatory.

### M12 — Security review
Required: fuzzing, race/static analysis, dependency audit, hostile P2P/RPC tests, wallet tamper tests, deep/repeated reorg tests, DoS/resource bounds, reproducible-build/provenance review and security contact.
Gate: no known unresolved critical/high issue and committed security report/checklist.

### M13 — Mainnet launch rehearsal
Rehearse clean bootstrap, mining, transactions, recovery, reorg, restart, bootstrap outage, upgrade/rollback, packaging, Explorer, website separation and incident communication.
Gate: documented PASS at an exact candidate commit.

### M14 — Genesis ceremony
Only after M0-M13 pass.
Freeze timestamp, message, target, nonce, hash, profile, source commit and release version.
Gate: two independent Genesis reproductions match exactly.

### M15 — Mainnet Release Candidate
Freeze source, binaries/installers, checksums, provenance, release notes, operator/miner docs, wallet backup docs, website downloads and Explorer configuration.
Gate: clean cross-platform verification and reproducible artifact-to-commit binding.

### M16 — Pre-launch distributed validation
Run multiple nodes and miners, fresh bootstrap, wallet send/receive, restart/reorg/node-loss recovery and sustained observation defined by the approved Mainnet spec.
Gate: no unresolved consensus/security/recovery blocker.

### M17 — Explicit launch authorization
Record exact Genesis, source commit, release, checksums, completed gates, known non-blocking issues and incident/rollback procedure.
Without this record Mainnet stays disabled.

### M18 — Mainnet launch
Require independently operated reachable nodes, bootstrap diversity, joinable mining software, public verifiable source/releases/checksums and active security contact.

### M19 — First 24-72 hours stabilization
Observe chain continuity, block times, DAA, reorgs/orphans, peer diversity, wallet incidents, miner behavior, storage/restart and release failures.
Critical issue pauses expansion work and triggers controlled incident/release procedure.

### M20 — Extended Mainnet stability gate
Default candidate: 14 consecutive days without unresolved consensus-critical or high-severity production defect.
Changing this duration requires a new Master-TZ revision with evidence.

### M21 — Explorer/API/operator readiness
Freeze factual lookup/API/operator interfaces, confirmations, reorg behavior and upgrade policy.
Gate: external clean-node integration procedure works.

### M22 — Exchange integration dossier
Prepare actual Mainnet Chain ID, Genesis, node version, ports, address format, UTXO model, transaction/broadcast rules, deposit/withdrawal flow, fees, coinbase maturity, reorg behavior, evidence-based confirmations, installation/upgrade/rollback, Explorer/API, source, checksums/provenance, security contact and Testnet procedure.
Gate: dossier matches live Mainnet behavior.

### M23 — Legal/compliance package
Prepare only factual information genuinely required by counterparties/jurisdictions: licensing, issuance/allocation facts, network description, risk disclosures and requested compliance information.
Never fabricate approvals or registrations.

### M24 — Exchange applications/support
Only after stable Mainnet and M22/M23.
Official applications, engineering due diligence, Testnet integration and deposit/withdrawal testing are allowed.
Forbidden: fake metrics, fake partners, fake listing claims, guaranteed-listing claims or bypassing failed technical integration.

## 9.4 Permanent post-listing rule

Exchange listing is not project completion.

Continue consensus maintenance, security response, versioned upgrades, reproducible releases, vulnerability handling, node/miner/wallet compatibility, backup/migration support and integration maintenance.

Any hard fork, consensus change, monetary-policy change, address-format change, PoW/DAA change or Genesis-related change requires a separate approved specification and activation plan.

## 9.5 Current position at v0.3.6-DRAFT

- Testnet2 remains governed by its frozen v0.2.14 values;
- Mainnet is disabled;
- RandomX and DAA remain candidate work until their freeze gates are formally recorded;
- RandomX cross-platform prototype is CI-verified;
- Mainnet PoW-blob prototype is CI-verified;
- Mainnet RandomX M3 boundary/reorg gate is CI-verified;
- Mainnet LWMA M4 candidate implementation/simulation gate is CI-verified;
- real user-owned CPU benchmark is measured and recorded;
- M5 evidence, policy math, DAA simulation and independent exact reproduction are CI-verified;
- consolidated M5 pre-freeze regression gate is green on Windows, Linux and macOS at commit `1a37b8abd702559f47b8df30075111642ab73ead`, workflow run `36668941106`;
- the M5 exact values above are pre-freeze candidates only; M0-M2 remain prerequisite gates before formal M3/M4/M5 freeze;
- Mainnet consensus is not frozen;
- Genesis is not frozen;
- Mainnet launch is not authorized;
- exchange work is future/blocked.

This file is the latest repository Master-TZ for Mainnet engineering until superseded by a later version.


# 17. v0.3.7 Mainnet engineering change log

Changes relative to `v0.3.6-DRAFT`:

1. Remove ambiguity from the Mainnet LWMA candidate when converting `nextWork` back to a target.
2. Define the exact consensus candidate inverse:
   `nextTarget = floor(2^256 / nextWork) - 1`, with `nextWork > 0`.
3. Define PowLimit interaction explicitly:
   `nextTarget = min(nextTarget, PowLimit)`.
4. Require exact integer/big-integer floor arithmetic and forbid floating-point arithmetic in this conversion.
5. Record already-observed CI evidence for the RandomX M3 boundary/reorg gate and the first M4 LWMA candidate/simulation gate.
6. Preserve all frozen Testnet2 v0.2.14 consensus/network values unchanged.
7. Preserve Mainnet disabled state; this revision does not freeze Genesis, InitialTarget or PowLimit and does not authorize Mainnet launch.

Why this revision is needed:
- the prior wording said only “derive target from nextWork”, which allowed multiple mathematically plausible inverse implementations;
- consensus code must produce one bit-for-bit identical target on every node.

Benefits:
- deterministic cross-platform consensus behavior;
- no implementation ambiguity at M4 freeze;
- easier golden-vector and independent implementation verification.

Risks:
- this formula becomes consensus-sensitive once M4 is frozen; changing it later would require another Master-TZ revision and an explicit activation plan.

No Mainnet launch, VDR sale or exchange-listing action is authorized by v0.3.7-DRAFT.


# 18. v0.3.8 Mainnet engineering change log

Changes relative to `v0.3.7-DRAFT`:

1. Record the real user-owned M5 RandomX benchmark:
   - CPU: Intel Core i5-8265U;
   - all-thread result: 418.723 H/s;
   - source class: real user machine.
2. Record three-class M5 calibration evidence and controlled CPU measurements.
3. Record the exact M5 candidate values:
   - InitialWork = `251233`;
   - InitialTarget =
     `000042c78dcd2f09f6673e99417b7b7d0dbb0ddcf732f964dc73fce1c6b50768`;
   - PowLimit policy candidate = 4x easier in work terms;
   - PowLimitWork = `62808`;
   - PowLimitTarget =
     `00010b1e7ce2d1034b183a2be3b26504779a86ab0ed6197fb90be6d3c07b200c`.
4. Record successful cross-platform M5 Evidence, Policy Candidate, DAA Simulation and Independent Reproduction gates.
5. Record successful consolidated M5 Pre-Freeze Gate #3:
   - run `36668941106`;
   - commit `1a37b8abd702559f47b8df30075111642ab73ead`;
   - Windows/Linux/macOS success.
6. Remove the provisional production hard-code for the not-yet-frozen Mainnet Chain ID from the candidate validator; expected Chain ID is now supplied explicitly from the network/storage context.
7. Align LWMA implementation with the v0.3.7 rule `nextWork > 0`: a non-positive derived value fails closed instead of being silently normalized to 1.
8. Move the RandomX storage test helper to test-only source compilation.
9. Preserve frozen Testnet2 v0.2.14 values unchanged.
10. Preserve the non-skippable staircase: this revision **does not** mark M3, M4 or M5 frozen while prerequisite gates M0-M2 remain open.
11. Mainnet remains disabled; Genesis remains unfrozen; launch remains unauthorized.

Why this revision is needed:
- M5 now has concrete reproducible evidence and exact candidate numbers that must be recorded in the authoritative specification;
- the pre-freeze cleanup changed consensus-candidate interfaces and fail-closed behavior and therefore must not exist only in source code;
- recording evidence without falsely advancing the staircase preserves auditability.

Benefits:
- exact M5 candidate values are no longer scattered across CI logs and experiments;
- the candidate Chain ID remains decoupled from unresolved M1 identity;
- DAA failure semantics match the written formula;
- future freeze work has a reproducible evidence trail.

Risks:
- the exact M5 values are consensus-sensitive once formally frozen;
- changing InitialTarget, PowLimit or the 4x policy after freeze would require another Master-TZ revision and activation plan;
- M0-M2 are still mandatory blockers to formal M3/M4/M5 freeze.

No Mainnet launch, Genesis ceremony, public sale or exchange-listing action is authorized by v0.3.8-DRAFT.


# 19. v0.3.9 Mainnet engineering change log

Changes relative to `v0.3.8-DRAFT`:

1. Supersede the three-independent-physical-client requirement specifically for **M0 of the Mainnet engineering staircase**.
2. Preserve historical `v0.2.14` unchanged; its Stage14A status remains historical and must not be relabeled as passed.
3. Replace the Mainnet M0 distributed-hardware blocker with a reproducible evidence package:
   - exact Testnet2 consensus regression;
   - Windows/Linux/macOS automated build/test coverage;
   - three-process Testnet2 network smoke/recovery;
   - Stage14A evidence-tooling tests;
   - one real user-owned Windows Testnet2 session covering startup, mining, persisted wallet state and restart recovery;
   - no unresolved consensus-critical defect or split.
4. This revision does not weaken M1-M5 consensus freeze requirements and does not authorize Mainnet.
5. M3/M4/M5 pre-freeze candidate evidence from v0.3.8 remains unchanged.

Why this revision is needed:
- the project owner removed the requirement to maintain three independent physical computers for this engineering gate;
- the old M0 wording still made that hardware topology a blocker even though the project already has automated multi-node and cross-platform validation machinery.

Benefits:
- M0 can be reproduced without requiring three separately maintained physical computers;
- one real user-owned machine remains in the evidence chain;
- automated three-node behavior, cross-platform regression and consensus checks remain mandatory;
- historical Testnet2 evidence is not falsified.

Risks:
- controlled multi-process/CI nodes do not provide the same ISP/NAT/hardware diversity as three genuinely independent operators;
- this reduces real-world peer-diversity evidence before Mainnet engineering advances;
- later M16 distributed Mainnet validation remains mandatory and must restore independent-node evidence before launch.

Master-TZ update is required because this changes a mandatory acceptance gate. This v0.3.9 revision is that update.

Mainnet remains disabled. Genesis remains unfrozen. Launch remains unauthorized.
