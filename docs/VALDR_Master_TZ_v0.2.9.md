# VALDR / MASTER TECHNICAL SPECIFICATION v0.2.9

**Desktop, cross-platform distribution and Testnet productization**  
**Date:** 26 September 2026  
**Status:** active master implementation baseline  
**Previous baseline:** `docs/VALDR_Master_TZ_v0.2.8.md`  
**Active branch:** `valdr-v0.2`

> v0.2.9 replaces v0.2.8 as the current implementation baseline. v0.2 through v0.2.8 remain immutable historical specifications. Frozen legacy v0.1 remains `release/valdr-devnet-v0.1`.

## 1. Product goal

VALDR (VDR) is its own cryptocurrency on its own blockchain.

VALDR is not an ERC-20/BEP/Solana token and must not depend on Ethereum, BNB Chain, Solana or another chain for consensus, transactions, mining or ownership.

The target product is not merely a collection of command-line binaries. A normal user must be able to:

1. open the official VALDR website;
2. choose Windows, macOS or Linux;
3. download a cryptographically verifiable installer/package with release provenance; OS-vendor code signing is additive hardening when obtainable, not a Testnet prerequisite;
4. install VALDR Desktop;
5. create/open an encrypted VALDR wallet;
6. start the local VALDR node automatically;
7. synchronize with the VALDR network;
8. send and receive VDR;
9. inspect synchronization, peers and transactions;
10. close and reopen the application without losing wallet or chain state.

The target user experience is comparable in maturity to established cryptocurrency desktop clients such as Bitcoin Core, Monero GUI and Litecoin Core, while preserving VALDR's own architecture and visual identity.

## 2. Scope and exclusions

The v0.2 line includes:

- blockchain/node hardening;
- Testnet;
- P2P v2;
- full synchronization;
- chainwork/reorg;
- fees/mempool;
- encrypted wallet;
- Explorer;
- Docker/Linux node deployment as optional operator tooling;
- VALDR Desktop;
- Windows/macOS/Linux packages;
- release verification, checksum and cryptographic provenance/attestation pipeline;
- distributed user-run Testnet;
- Bitcoin/Litecoin-style peer bootstrap: learned peer cache, peer exchange, optional fixed/DNS seeds and manual peers;
- Mainnet specification draft.

Not included before a separately approved Mainnet specification:

- Mainnet launch;
- ICO/presale;
- sale of VDR;
- exchange listing;
- staking;
- smart contracts;
- NFT;
- bridge;
- mobile wallet;
- investment promises.

## 3. Current implementation status

| Stage | Deliverable | Status |
| --- | --- | --- |
| 0 | v0.2 branch/spec freeze | implemented |
| 1 | Storage v2 + migration | implemented, CI-verified |
| 2 | Network profiles + P2P v2 | implemented, CI-verified |
| 3 | Chainwork + reorg/undo | implemented, CI-verified |
| 4 | Difficulty v2 + timestamps | implemented; Testnet v0.2.9 recalibration pending CI verification |
| 5 | Fees + consensus size limits | implemented, CI-verified |
| 6 | Mempool policy + miner ordering | implemented, CI-verified |
| 7 | Headers-first full sync | implemented, CI-verified |
| 8 | Seeds + P2P protection software | implemented, CI-verified |
| 9 | Wallet encryption v2 | implemented, CI-verified |
| 10 | Explorer + reorg-safe index | implemented, CI-verified |
| 11 | Testnet runtime + Docker + Linux | implemented, CI-verified |
| 12 | VALDR Desktop | in progress / implementation present; Testnet1 Windows evidence retained but final acceptance must rerun on Testnet2 |
| 13 | Cross-platform installers + release pipeline | in progress; prior Testnet1 development evidence retained, Testnet2 packaging/provenance must be reverified; public release blocked |
| 14 | Distributed user-run Testnet | planned / not started |
| 15 | Mainnet specification draft | planned / no launch |

Stages 0-11 must not be silently redesigned while implementing Desktop. Desktop consumes the existing node/wallet/RPC architecture instead of duplicating consensus logic.

## 4. Mandatory reference research

Before implementing or materially changing VALDR Desktop, the download website, first-run flow, installer UX, wallet UX or release-verification UX, the developer must review current official reference products/sites.

Minimum mandatory references:

1. **Bitcoin Core**
   - official project/download reference: `https://bitcoincore.org/en/download/`
   - study: OS-specific downloads, full-node explanation, storage/bandwidth warnings, checksums/signatures, version/release presentation.

2. **Monero GUI**
   - official downloads: `https://www.getmonero.org/downloads/`
   - study: GUI wallet, Simple/Advanced concepts, Windows/macOS/Linux distribution, hash/signature verification, wallet/send/receive workflows.

3. **Litecoin Core**
   - official project/download reference: `https://litecoin.org/`
   - study: Core vs simpler wallet presentation, full-node positioning, platform packages, signatures/download UX.

4. **Ethereum ecosystem**
   - official wallets: `https://ethereum.org/wallets/`
   - official node onboarding: `https://ethereum.org/run-a-node`
   - study: user education, wallet security language, device/platform selection, node onboarding and maintenance guidance.
   - Ethereum is a UX/documentation/security reference only. Its multi-client execution/consensus architecture must not replace VALDR's own single-chain architecture.

For each material Desktop/download design pass, create or update:

`docs/product/desktop_reference_review.md`

It must record:

- date reviewed;
- official URLs reviewed;
- screenshots/notes or feature observations;
- what VALDR adopts;
- what VALDR explicitly rejects;
- why;
- security/release implications.

Reference research is not permission to clone branding, artwork, copyrighted UI, text or incompatible architecture. VALDR must keep original branding/UI and review licenses before reusing any source code.

## 5. Product simplification rules

Keep:

- one Go consensus implementation;
- one canonical node implementation;
- encrypted wallet v2;
- localhost RPC boundary;
- P2P v2;
- Testnet/Mainnet network profiles;
- Explorer as a separate read-only service;
- CLI tools for operators/developers;
- Docker/systemd as optional operator tooling; no project-owned/rented public server is required.

Do not add merely for Desktop:

- a second blockchain implementation;
- a second wallet key format without an explicit migration design;
- embedded web services that expose private keys;
- Electron unless Wails becomes technically impossible;
- a browser extension wallet;
- mobile apps;
- a mandatory cloud account;
- custody/server-side private keys;
- exchange/buy/sell integrations before Mainnet policy;
- automatic unsigned self-update.

## 6. Network profiles

Historical profiles remain readable/testable. New public product work uses the active v0.2.9 Testnet profile and a fresh chain identity.

| Parameter | Legacy v0.1 | Devnet v0.2 | Historical Testnet v0.2 | Active Testnet v0.2.9 |
| --- | --- | --- | --- | --- |
| Profile name | `legacy-v0.1` | `devnet2` | `testnet` | `testnet2` |
| Chain ID | `valdr-devnet-1` | `valdr-devnet-2` | `valdr-testnet-1` | `valdr-testnet-2` |
| P2P | v1 | v2 | v2 | v2 |
| P2P port | 7333 | 7333 | 17333 | 17333 |
| RPC port | 7332 | 7332 | 17332 localhost | 17332 localhost |
| Address prefix | VDR1 | VDR1 | VDR1 | VDR1 |
| Target block | 60 s | 60 s | 60 s | 60 s |
| Initial subsidy | 50 VDR | 50 VDR | 50 Testnet VDR | **1 Testnet VDR** |
| Header version | 1 | 2 | 2 | 2 |

Consensus-critical parameters are compile-time/network-profile only.

The `1 Testnet VDR` subsidy is a Testnet-only testing parameter. It does **not** define Mainnet emission, halving or maximum supply. Those remain Stage 15 decisions.

### 6.1 Historical Testnet Genesis

`valdr-testnet-1` remains frozen for historical compatibility:

- version: 2;
- timestamp: `1790208000` (2026-09-24 00:00:00 UTC);
- target: `000fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff`;
- nonce: `12480`;
- message: `VALDR genesis block | valdr-testnet-1 | 2026-09-24`;
- hash: `0009d956448a8caefcd798af1a7957840d0aa7b72f8350362909f241ea100122`.

### 6.2 Active Testnet v0.2.9 Genesis

The consensus recalibration intentionally starts a new Testnet chain. `valdr-testnet-2` Genesis is frozen as:

- version: 2;
- timestamp: `1790380800` (2026-09-26 00:00:00 UTC);
- target: `0000031b5d43afe99ee43470e1337c3642e9d9254926038fdf6d1a2e57aaa21f`;
- nonce: `22759786`;
- message: `VALDR genesis block | valdr-testnet-2 | 2026-09-26`;
- hash: `000000a065ed224c03ff3107b2d3a073906415b347600f2e83a8874371f15485`;
- P2P magic: first four SHA-256 bytes of Chain ID = `900d7c51`.

Existing `valdr-testnet-1` blocks and Testnet balances do not migrate into `valdr-testnet-2`. Wallet keys may still be used locally, but the new chain starts from its own Genesis and UTXO state. Testnet VDR has no promised monetary value.

Mainnet Genesis does not exist yet and must not be fabricated.

## 7. Storage v2

Backend: BadgerDB pinned by `go.mod`.

Required logical indexes:

- `meta/*`;
- `block/<hash>`;
- `height/<height>`;
- `header/<hash>`;
- `tx/<txid>`;
- `utxo/<txid>:<index>`;
- `undo/<blockhash>`;
- `migration/*`.

Block acceptance/reorg state changes must remain atomic.

Legacy migration remains explicit, one-way and verification-gated. Original v0.1 data is preserved.

Desktop must use a platform-correct user data directory and must not change storage schema merely for GUI convenience.

## 8. Consensus block/header rules

Consensus limits:

- max canonical block size: 1,000,000 bytes;
- max canonical transaction size: 100,000 bytes;
- SHA-256 block hashes.

Header v2:

- exact unsigned 256-bit target;
- JSON target = exactly 64 lowercase hex chars;
- hashed target = exactly 32 raw big-endian bytes;
- target > 0 and <= network PoW limit;
- v2 `difficulty uint64` is non-consensus and must be 0;
- target is part of the hashed header preimage.

Canonical v2 header order remains:

```text
version uint32 BE
height uint64 BE
previous_block_hash length uint64 + ASCII bytes
merkle_root length uint64 + ASCII bytes
timestamp uint64 BE
target 32 raw BE
nonce uint64 BE
chain_id length uint64 + UTF-8 bytes
extra_data length uint64 + UTF-8 bytes
```

Legacy v1 encoding remains frozen.

## 9. Timestamp, difficulty and fork choice

Timestamp rules remain:

- > Median-Time-Past previous 11;
- <= local system time + 2 hours.

Historical v0.2 Testnet keeps its frozen difficulty parameters for compatibility.

Active `valdr-testnet-2` difficulty:

- target block interval: **60 s**;
- calibrated Genesis/initial target: `0000031b5d43afe99ee43470e1337c3642e9d9254926038fdf6d1a2e57aaa21f`;
- calibrated initial expected work: **5,400,000 SHA-256 trials/block**;
- retarget: every **10 blocks**;
- preceding 10-header measurement covers 9 block intervals, therefore target timespan = **540 s**;
- clamp: **135..2,160 s** (1/4x..4x adjustment bound);
- active Testnet PoW limit uses 22 leading zero bits: `000003ffffffffffffffffffffffffffffffffffffffffffffffffffffffffff`;
- Testnet min-difficulty escape remains after 10 minutes with deterministic recovery;
- exact integer target arithmetic remains mandatory.

The bootstrap target is intentionally harder than the superseded Testnet1 target but still practical for CI and ordinary CPU testing. Windows QA on a real single-thread miner measured roughly 330–379 kH/s and observed individual Testnet1 solves from about 89 ms to 1.24 s; 94 blocks were accepted in a short run. That evidence showed the old 60-block retarget/very-easy initial target was unsuitable for distributed Testnet use.

At approximately 360 kH/s the new initial target implies roughly 15 seconds expected bootstrap solve time. If early blocks arrive near that rate, the first 10-block retarget reaches approximately one quarter of the initial target, corresponding to roughly 21.6 million expected hashes/block, near the 60-second objective at that observed hashrate. This is a Testnet calibration point, not an assumption about future network hashrate.

Chainwork remains:

`work = floor(2^256 / (target + 1))`.

Active chain = valid branch with greatest cumulative chainwork. Equal work keeps current active tip.

Reorg remains common-ancestor + undo disconnect + normal reconnect + atomic tip switch + mempool reconsideration.

## 10. Transactions, network binding, fees and mempool

### 10.1 Legacy v0.1 transaction format

Frozen legacy v0.1 transaction behavior remains byte-for-byte compatible:

- transaction version = 1;
- no explicit `chain_id` JSON field;
- canonical/signing preimage keeps the historical implicit `valdr-devnet-1` domain prefix;
- v0.1 validation continues to accept only that legacy format.

### 10.2 v2 network-bound transaction format

Devnet2 and Testnet transactions must be network-bound and replay-resistant.

For profiles with block/header protocol v2:

- transaction version = 2;
- transaction JSON includes explicit `chain_id`;
- `chain_id` must exactly equal the active network profile Chain ID;
- signature preimage and txid canonical serialization begin with that explicit Chain ID using the existing length-prefixed UTF-8 encoding;
- a v2 transaction signed for `valdr-testnet-1` must be invalid on `valdr-devnet-2` and vice versa;
- missing, legacy or foreign Chain ID is consensus-invalid on v2 networks;
- coinbase transactions are also network-bound so coinbase txids cannot be replayed across networks.

The remaining canonical transaction field order is unchanged:

```text
chain_id length uint64 + UTF-8 bytes
version uint32 BE
timestamp uint64 BE
input_count uint64 BE
inputs...
output_count uint64 BE
outputs...
public_key length uint64 + bytes
signature length uint64 + bytes
```

`transaction_id` itself is derived from canonical bytes and is not serialized into its own hash preimage.

### 10.3 Network-aware validation

Consensus/policy callers must validate against the active network profile:

- blockchain block-connect validation;
- active UTXO application;
- fee calculation;
- mempool admission/revalidation;
- miner template construction;
- RPC transaction broadcast;
- wallet/CLI/Desktop transaction creation.

Generic legacy validation helpers may remain for frozen v0.1 tests/APIs, but v2 nodes must not rely on them.

### 10.4 Fees

Normal tx:

- inputs >= outputs;
- fee = inputs - outputs;
- no fee field.

Coinbase maximum = subsidy(height) + block fees. Under-claim is valid.

Mempool defaults:

- max 64 MiB;
- expiry 72 h;
- Testnet min relay = 1 val/byte;
- no RBF;
- no unconfirmed-parent relay;
- conflict rejection;
- lowest-feerate/oldest deterministic eviction;
- highest-feerate/txid deterministic miner ordering;
- revalidation after active-chain changes.

## 11. P2P v2 and synchronization

Frame remains:

- first 4 bytes SHA-256(chain_id) magic;
- protocol uint16;
- type uint16;
- payload length uint32;
- 4-byte SHA-256 checksum prefix;
- strict UTF-8 JSON.

Supported v2 messages remain:

`hello, hello_ack, ping, pong, inv, get_data, block, tx, get_headers, headers, get_blocks, get_peers, peers, reject`.

Synchronization:

- block locator recent then exponential to Genesis;
- locator <=128;
- headers <=2,000;
- validate chain/version/linkage/target/timestamp/difficulty/PoW before bodies;
- <=32 body requests per batch;
- persist side branches;
- select greatest chainwork;
- resume from persisted chain after restart.

## 12. P2P protection and Desktop outbound mode

Existing protection stays:

- handshake timeout 5 s;
- idle timeout 5 min + ping;
- max inbound 64;
- outbound target 8;
- max inbound/IP 4;
- 4 MiB frame ceiling + per-message limits;
- token buckets;
- malformed scoring/temp bans;
- bounded duplicate caches;
- public gossip address filtering.

### 12.1 Bind vs advertise address

Public infrastructure may bind `0.0.0.0:17333` but must advertise an explicitly routable DNS/IP endpoint.

### 12.2 Desktop default: outbound-only

VALDR Desktop is a user application, not automatically a public server.

Default Desktop mode must:

- connect outbound to cached/seed/discovered peers;
- not require router/NAT configuration;
- not require opening firewall port 17333;
- not gossip loopback/private/bind-all addresses as a public endpoint;
- expose node RPC only to the local Desktop process;
- allow an explicit Advanced option to run as a publicly reachable full node.

Implementation may add an explicit non-listening/outbound-only P2P mode. This is operational P2P behavior, not a consensus change. Frozen v0.1 behavior remains unchanged.

### 12.3 Bitcoin/Litecoin-style bootstrap and peer persistence

VALDR does not require a central server or mandatory project-owned public full nodes. Internet P2P discovery follows the same general pattern used by established UTXO networks: a node remembers previously learned peers, may consult optional bootstrap sources when its address book is insufficient, then expands through peer exchange.

Required v0.2.6 behavior:

- persist a bounded non-consensus peer cache in the node data directory;
- try learned peers again after restart;
- support optional compiled fixed seed endpoints;
- support optional DNS seed hostnames that resolve to ordinary reachable peers;
- keep manual `--peer` / `--seed` operator overrides;
- request/relay peer advertisements after connection;
- periodically maintain outbound connectivity instead of bootstrapping only once at process start;
- treat all seeds as discovery helpers only: they never decide chain validity, balances, mining or fork choice;
- tolerate individual seed/peer failure;
- never fabricate placeholder public endpoints.

A cold Internet node cannot discover a network from literally zero reachable contacts. Therefore at public rollout time at least one real reachable bootstrap path must exist, but it may be supplied by any independently operated user full node or volunteer seed operator; it does not have to be a paid VALDR server. Once peers are learned, the persisted peer cache and peer exchange reduce dependency on bootstrap sources.

This change is operational P2P/product behavior only. It does not change consensus, the P2P v2 wire frame, Genesis, PoW, UTXO, emission or wallet cryptography.

## 13. RPC security

RPC is localhost by default.

Desktop must use a dedicated localhost endpoint. Privileged actions must not become remotely accessible because a GUI exists.

Private keys/passphrases must never cross P2P or node RPC.

Any additional Desktop RPC method must be:

- read-only unless strictly necessary;
- versioned/documented;
- bounded;
- tested;
- localhost-only by default.

Useful Desktop read APIs may include:

- sync status/progress;
- network/profile identity;
- local/peer height;
- chainwork;
- peers;
- mempool;
- disk/DB state;
- fee estimate;
- address history/UTXO.

## 14. Wallet v2

Wallet file v2 remains:

- no plaintext private key;
- scrypt-derived 256-bit key;
- current file parameters: N=32768, r=8, p=1;
- random 16-byte salt;
- AES-256-GCM;
- fresh random nonce;
- metadata authenticated as AAD;
- wallet file mode 0600 where supported;
- wallet directory mode 0700 where supported.

Desktop wallet unlock/decrypt/signing happens locally in the Desktop process/library.

No password may be passed as a command-line string.

Desktop must zero temporary passphrase/key byte buffers where practical and must never log secrets.

## 15. Explorer

Explorer stays a separate read-only service for public/operator use.

Desktop must not embed the Explorer server merely to render ordinary wallet screens. Desktop may reuse the same read models or local RPC data.

Existing Explorer REST/reorg-safe index remain intact.

## 16. Stage 12 — VALDR Desktop architecture

### 16.1 Framework

Initial desktop framework: **Wails v2 stable**.

Reason:

- existing core is Go;
- supports Windows/macOS/Linux;
- uses native platform WebView instead of shipping a complete browser runtime;
- can call Go application methods from the frontend;
- has packaging support including Windows installer generation.

Wails v3 beta is not the release baseline until it becomes stable and a deliberate migration is approved.

Frontend assets must be locally bundled. No remote CDN JavaScript/CSS is required for the wallet to function.

Use TypeScript for frontend application logic. Avoid unnecessary frontend dependencies. A UI framework may be introduced only if it materially reduces complexity and is pinned/audited.

### 16.2 Process model

Preferred Desktop process model:

```text
VALDR Desktop
  |
  +-- wallet library (local encrypted wallet, signing)
  |
  +-- managed valdrd child process
  |     +-- blockchain/storage
  |     +-- P2P/sync
  |     +-- mempool
  |     +-- localhost RPC
  |
  +-- optional managed valdr-miner process (Advanced/Testnet)
```

Desktop must not fork/duplicate consensus code.

Node lifecycle:

- start automatically after Desktop initialization unless explicitly disabled;
- detect existing managed node;
- avoid two nodes opening the same DB;
- graceful shutdown;
- timeout then controlled termination only when needed;
- preserve DB across application restart;
- expose logs to user without secrets.

### 16.3 First-run flow

On first launch:

1. show VALDR/Testnet identity and short explanation;
2. select/create data directory if needed;
3. explain required disk/network resources;
4. create a new encrypted wallet or open/restore an existing encrypted wallet backup;
5. require wallet passphrase;
6. start local node;
7. begin outbound-only synchronization;
8. show synchronization progress and connection status;
9. enter main application only after required initialization succeeds.

Mainnet must not appear as a usable option before a separately approved Mainnet spec/release.

### 16.4 Simple and Advanced modes

Default **Simple** mode:

- Overview;
- Send;
- Receive;
- Transactions;
- Wallet backup;
- synchronization status;
- basic settings.

**Advanced** mode may expose:

- peers;
- chain height/tip/chainwork;
- mempool;
- logs;
- node data directory;
- inbound/public-node mode;
- mining on Devnet/Testnet;
- Explorer link;
- diagnostics.

Protocol/consensus parameters are never editable.

### 16.5 Main screens

Minimum UI:

**Overview**
- spendable balance;
- wallet locked/unlocked state;
- network name;
- sync percentage/state;
- local height;
- best known height when available;
- peer count;
- latest transaction summary.

**Send**
- destination VDR address validation;
- amount;
- estimated/actual fee;
- total spend;
- confirmation dialog before signing/broadcast;
- clear irreversible-transaction warning.

**Receive**
- current wallet address;
- copy button;
- QR code generated locally;
- no web API needed for QR.

**Transactions**
- pending/confirmed;
- txid;
- timestamp;
- direction;
- amount;
- fee when applicable;
- confirmations/block;
- detail view.

**Wallet**
- lock/unlock;
- create/open;
- encrypted backup;
- restore;
- explicit high-risk private-key export behind additional warning.

**Network/Node**
- sync state;
- peers;
- tip;
- chainwork;
- disk path/state;
- public-node toggle in Advanced mode.

**Mining**
- Testnet/Devnet only by default;
- explicit start/stop;
- reward address;
- accepted blocks/hashrate if available;
- never silently auto-mine.

**Settings**
- language;
- theme;
- startup behavior;
- data directory where safe;
- network selection limited to released profiles;
- Advanced mode;
- logs/diagnostics.

## 17. Desktop security requirements

Mandatory:

- wallet secrets remain local;
- no telemetry by default;
- no analytics SDK by default;
- no remote scripts/content required for core UI;
- no ads;
- no automatic clipboard reading;
- no automatic private-key export;
- sanitize addresses/amounts before actions;
- confirmation before irreversible send;
- protect against HTML/script injection in tx/address metadata displayed by UI;
- CSP/local asset restrictions;
- lock wallet after configurable inactivity;
- OS keychain may store only non-secret convenience metadata initially unless a separate reviewed design approves secret storage;
- crash reports must not include keys/passphrases/auth tokens;
- Desktop logs must redact secrets.

## 18. Cross-platform target matrix

Mandatory release targets before public Testnet:

| Platform | Architecture | Required artifact |
| --- | --- | --- |
| Windows 10/11 | AMD64 | installer `.exe` + portable archive; SHA-256 + release provenance attestation mandatory; Authenticode optional when obtainable |
| macOS | Apple Silicon ARM64 | `.app` / `.dmg`; SHA-256 + release provenance attestation mandatory; Developer ID/notarization optional when obtainable |
| macOS | Intel AMD64 | `.app` / `.dmg` or universal binary; SHA-256 + release provenance attestation mandatory; Developer ID/notarization optional when obtainable |
| Linux | AMD64 | `.AppImage` + `.deb`; SHA-256 + release provenance attestation mandatory |

Optional after mandatory gates:

- Windows ARM64;
- Linux ARM64;
- other package formats.

Exact minimum OS versions are pinned by the release CI/toolchain and release notes. They must not exceed what the selected Wails stable line supports without an explicit compatibility decision.

## 19. Stage 13 — installer and release pipeline

### 19.1 Website download UX

Official VALDR download page must:

- prominently show current version;
- auto-suggest current OS without hiding other platforms;
- offer Windows/macOS/Linux explicitly;
- state architecture;
- show file size;
- link release notes;
- publish SHA-256 checksums;
- publish SHA-256 checksums and cryptographically verifiable release provenance/attestation for the release artifacts and canonical manifest;
- link source code;
- explain how to verify downloads;
- show Testnet status prominently before Mainnet exists;
- show system requirements and synchronization/disk warning;
- never present an unofficial mirror as the primary download.

During Testnet there must be no buy/sell/exchange call-to-action implying monetary value.

### 19.2 Release authenticity and provenance

Mandatory Testnet authenticity model:

- every distributable artifact has SHA-256 and byte size in the canonical release manifest;
- the canonical build/release workflow creates a cryptographic provenance attestation binding artifact digests to the public VALDR repository, exact git commit and workflow identity;
- the preferred current implementation is GitHub Artifact Attestations backed by Sigstore keyless OIDC signing for the public repository;
- verification instructions must enforce the expected repository and signer workflow and should enforce the exact source commit for a frozen release;
- the attestation is a provenance/integrity claim, not a claim that Microsoft, Apple, GitHub or Sigstore audited VALDR for safety;
- a clean verification environment must validate both SHA-256 and provenance before public Testnet distribution.

OS-vendor signing:

- Windows Authenticode is optional hardening when a legitimate code-signing identity becomes obtainable; absence of Authenticode must be disclosed and must not be reported as signed;
- macOS Developer ID signing/notarization is optional hardening when a legitimate Apple identity becomes obtainable; absence must be disclosed and must not be reported as notarized;
- unsigned Windows/macOS Testnet packages may trigger operating-system security warnings or require an explicit user override; verification documentation must explain this without instructing users to disable platform security globally;
- Linux remains checksum/provenance verified.

No fake, borrowed, misleading or inaccessible third-party identity may be used merely to satisfy a checklist.

Any long-lived signing key introduced later must not be stored in the repository, package or ordinary CI logs. The keyless attestation path must not require a private signing secret in the repository.

### 19.3 Release manifest

Every release publishes a machine-readable manifest containing at minimum:

- VALDR version;
- git commit;
- network/protocol version;
- artifact filenames;
- OS/arch;
- SHA-256;
- size;
- OS-vendor signing/notarization status, truthfully including unavailable/not-used states;
- provenance/attestation status;
- release date;
- minimum supported OS.

## 20. Update policy

Initial Desktop release may check for a newer version but must not silently install updates.

Until a signed update mechanism is separately reviewed:

- notification only;
- user opens official download/release page;
- downloaded release remains signature/checksum verifiable.

Never execute an unsigned downloaded binary automatically.

## 21. CI for Desktop

Required CI jobs:

```text
Go core build/test/race
desktop frontend lint/typecheck/test
Wails build Windows AMD64
Wails build macOS ARM64
Wails build macOS AMD64 or universal
Wails build Linux AMD64
installer/package creation
artifact checksum generation
artifact provenance attestation generation
clean provenance verification
clean-launch smoke per OS
wallet create/unlock/backup/restart smoke
node start/sync/restart smoke
send/receive Testnet integration
no-private-key-on-disk regression
no-secret-in-logs regression
```

Platform packaging/signing jobs may require native runners.

A Linux build of a Windows/macOS artifact is not a substitute for native clean-machine smoke.

## 22. Desktop acceptance gate

Stage 12 is complete only when:

- Desktop builds on all mandatory target OSes;
- app starts without terminal use;
- first-run flow creates encrypted wallet;
- managed node starts and stops correctly;
- outbound-only Testnet sync works;
- restart resumes without DB deletion;
- balance/send/receive/transaction history work;
- wallet backup and restore work;
- wrong password fails safely;
- node crash is surfaced and recoverable;
- UI never displays Mainnet as available;
- no private key/passphrase appears in node RPC/P2P/logs.

## 23. Installer/release acceptance gate

Stage 13 is complete only when:

- Windows clean installer/uninstaller passes;
- macOS clean install/launch passes on ARM64 and Intel/universal target; if Developer ID/notarization is unavailable, the release is explicitly labelled unsigned/unnotarized and provenance verification must still pass;
- Linux AppImage and deb clean install pass;
- checksums match;
- release manifest is generated;
- release assets map to exact git commit;
- cryptographic provenance attestation exists for every release artifact and verifies against the expected VALDR repository/workflow;
- a frozen public release verifies against its exact source commit;
- download page presents correct platform artifacts and truthfully discloses Windows/macOS vendor-signing status;
- verification instructions are tested by a second clean environment.

## 24. Stage 14 — Distributed user-run Testnet

Only after Stages 12 and 13 are green.

There is no requirement to rent or maintain project-owned public servers.

Network requirements:

- >=3 independently operated Testnet nodes/clients participate in the same chain;
- nodes may run on ordinary user computers, home Internet, volunteer infrastructure or optional operator servers;
- at least one real reachable bootstrap path exists for a cold Internet client during the rollout test;
- bootstrap may use learned peer cache, a volunteer fixed seed, a volunteer DNS seed or an explicitly shared peer address;
- after first contact, peer exchange and persisted peer cache must allow the client to learn additional peers;
- loss of any single bootstrap source must not change consensus and must not stop already connected peers from operating;
- no fake seed addresses are committed merely to satisfy a checklist.

Sequence:

1. multi-user/private soak >=24 h using independently launched clients;
2. fix all consensus/storage/sync/wallet/desktop/bootstrap blockers;
3. freeze Testnet release candidate;
4. publish checksum-verified and cryptographically provenance-attested Desktop installers/packages;
5. distribute Testnet client to additional users;
6. >=7 days operation without consensus split or manual DB repair.

Monitor:

- height/tip/chainwork;
- peer diversity and bootstrap success;
- mempool;
- block intervals;
- reorgs;
- disk;
- RPC;
- restarts;
- peer-cache recovery after restart;
- Desktop crash/startup/sync failures;
- installer/download verification failures.

## 25. Observability

Keep categories:

`NODE/P2P/BLOCK/TX/MINER/MEMPOOL/SYNC/ERROR/STORAGE/RPC/EXPLORER/REORG`.

Desktop adds:

`DESKTOP/WALLET/UPDATE`.

Logs use UTC and structured key/value context. Never log keys, passphrases, auth tokens or complete secret backup material.

## 26. Test/security requirements

Keep all existing regression suites.

Add before public Testnet:

- decoder fuzzing;
- DB crash/restart/corruption;
- deep/repeated reorg;
- oversize/checksum/flood P2P;
- RPC exposure checks;
- wallet wrong password/tamper/migration;
- Desktop managed-node crash/restart;
- duplicate Desktop instance/data-lock handling;
- malicious/invalid address input;
- frontend injection tests;
- packaging/install/uninstall;
- checksum/manifest and provenance-attestation verification;
- upgrade/rollback from previous Desktop release candidate.

## 27. Source control and release discipline

- frozen v0.1 remains unchanged;
- v0.2 work remains on `valdr-v0.2`;
- v0.2.9 is the active master specification after its commit;
- previous master specs remain preserved;
- stage flow remains:
  `spec -> implementation -> build -> tests -> runtime verification -> commit -> next stage`;
- no stage is declared complete only because code compiles;
- consensus/wire/storage changes require a new spec revision;
- UX-only changes normally do not require a master-spec revision unless they change security boundaries or mandatory product behavior;
- signed release refs only on fully tested commits.

## 28. Mainnet specification draft

Stage 15 creates a draft only. Mainnet remains disabled.

Before Mainnet launch a separate approved spec must freeze:

- Chain ID;
- Genesis;
- magic/ports/seeds;
- emission/halving/max supply;
- difficulty;
- block/tx limits;
- fee/mempool defaults;
- coinbase maturity;
- address/network policy;
- fork/reorg/checkpoint policy;
- public-node diversity;
- incident response;
- reproducible builds;
- release signing/provenance policy;
- Desktop Mainnet enablement/migration plan.

## 29. Acceptance criteria for v0.2 productization

v0.2 productization is accepted only when:

- clean clone passes full core CI;
- Testnet node sync/restart/reorg is stable;
- encrypted wallet v2 is stable;
- Explorer is reorg-safe;
- Docker/Linux infrastructure is stable;
- VALDR Desktop passes Windows/macOS/Linux gates;
- cryptographically verifiable, provenance-attested release artifacts exist;
- ordinary user can install and use Testnet without terminal commands;
- at least 3 independently operated Testnet nodes/clients participate without requiring project-owned paid servers;
- a cold client can reach at least one real bootstrap contact, learn more peers, persist them and reconnect after restart;
- distributed Testnet remains stable for >=7 days without consensus split/manual DB repair;
- Mainnet remains disabled.

## 30. v0.2.4 change log

Changes relative to v0.2.3:

1. Preserve the Desktop/cross-platform product plan introduced by v0.2.3.
2. Fix a consensus-domain defect discovered while enabling Desktop Send: the existing transaction canonical serializer used the legacy global `valdr-devnet-1` prefix even when a node was running Devnet2/Testnet.
3. Freeze legacy v0.1 transaction version 1 byte-for-byte.
4. Define transaction version 2 for Devnet2/Testnet with explicit network `chain_id`.
5. Require blockchain, UTXO, mempool, miner, RPC, CLI and Desktop paths to validate/create transactions against the active profile.
6. Require cross-network replay tests proving Testnet transactions fail on Devnet2 and vice versa.
7. Network-bind coinbase txids for v2 networks.
8. No block header, PoW, emission, wallet-encryption or storage architecture is changed by this revision.

Benefits:

- VALDR becomes installable software for ordinary users rather than an operator-only CLI stack;
- core consensus architecture remains unchanged;
- security boundaries remain explicit;
- public Testnet can test the actual product people will run;
- releases become verifiable and reproducible enough for wider distribution.

Risks:

- cross-platform GUI/installer/signing substantially expands the test matrix;
- managed-node lifecycle introduces desktop-specific process/state edge cases;
- platform signing requires protected external credentials;
- Wails/WebView behavior differs across OSes;
- public Testnet must not begin until these gates are stable.

No Mainnet launch is authorized by v0.2.4.


## 31. v0.2.5 change log

Changes relative to v0.2.4:

1. Reconcile the normative Stage 12 status with the repository: VALDR Desktop is now explicitly **in progress**, not “planned / not started”.
2. Record that the existing Stage 12 implementation continues to consume the current node/wallet/RPC architecture; no duplicate consensus implementation is authorized.
3. Preserve the v0.2.4 transaction network-binding rules unchanged.
4. Confirm Advanced/Testnet mining as an implementation of the already-required Stage 12 Mining screen: explicit start/stop, explicit reward address, managed local `valdr-miner`, no automatic mining, and Mainnet disabled.
5. Stage 12 remains incomplete until its required core/Desktop/runtime and mandatory OS acceptance gates are actually executed and green.
6. A GitHub Actions run that never receives a runner and executes zero steps is infrastructure failure only; it is not test evidence and must not be reported as CI verification.

No consensus, P2P wire, storage schema, wallet encryption format, emission rule, Genesis, or Mainnet enablement changes are introduced by v0.2.5.


## 32. v0.2.6 change log

Changes relative to v0.2.5:

1. Remove the mandatory project-owned/rented Public Testnet server requirement.
2. Replace Stage 14 with a distributed user-run Testnet rollout.
3. Adopt a Bitcoin/Litecoin-style discovery model: persisted learned-peer cache, peer exchange, optional fixed seeds, optional DNS seeds and explicit manual peers.
4. Require periodic outbound maintenance so the node retries discovery after peer loss rather than only at process startup.
5. Keep VALDR Desktop outbound-only by default; any user may explicitly enable public full-node mode.
6. Keep Docker/Linux deployment as optional operator tooling rather than a mandatory project expense.
7. Preserve the rule that bootstrap sources are not trusted consensus authorities.
8. Explicitly document the unavoidable cold-start constraint: an Internet node needs at least one reachable contact the first time it joins; this contact may be any independently operated peer and does not need to be a VALDR-paid server.
9. No fake seed endpoint may be committed before a real operator exists.
10. No consensus, P2P v2 frame, storage schema, transaction format, wallet encryption, Genesis, PoW, emission or Mainnet enablement changes are introduced by this revision.

Benefits:

- VALDR can spread through user-run nodes instead of depending on a paid central infrastructure budget;
- restart recovery improves because learned peers survive process restarts;
- the network becomes less dependent on any single seed after peers have been learned;
- Desktop remains simple for ordinary NAT/firewall users while advanced users can contribute reachable full nodes.

Risks:

- a brand-new network still cannot bootstrap from literally zero reachable peers;
- home NAT/CGNAT means many Desktop nodes will be outbound-only, so some independently operated reachable peers are still necessary for a healthy Internet topology;
- DNS/fixed seed operators, when used, must be diversified because they are discovery infrastructure even though they are not consensus authorities.

Reference behavior reviewed from current Bitcoin Core and Litecoin Core source/documentation before this revision:
- Bitcoin Core exposes both DNS seeds and fixed seeds and documents minimizing trust in DNS seed operators;
- Litecoin Core likewise ships multiple DNS seeds plus fixed seeds;
- VALDR adopts the discovery pattern, not their branding or codebase.


## 33. v0.2.7 change log

Changes relative to v0.2.6:

1. Authorize Stage 13 to proceed incrementally by explicit project direction while the remaining Stage 12 manual Windows acceptance is still tracked as open.
2. Stage 12 is **not** reclassified as complete. Its outstanding manual acceptance evidence remains mandatory.
3. While Stage 12 is open, Stage 13 work is limited to CI-safe development plumbing such as deterministic release metadata/checksum generation, packaging configuration, reproducible dry-runs and non-public clean-install automation.
4. Public Testnet release tags, production signing/notarization claims, public release publication and Stage 14 rollout remain blocked until Stage 12 acceptance is green.
5. The application version remains `config.Version = "0.2.0-dev"` until a Testnet release candidate is deliberately frozen. Development dry-runs must be labelled as such and cannot satisfy the signed-release acceptance gate.
6. Release metadata must continue to bind every artifact to the exact git commit, Testnet network identity, Chain ID, protocol version, byte size and SHA-256.
7. No consensus, transaction, P2P wire, storage schema, wallet encryption, Testnet Genesis, PoW, emission or Mainnet enablement changes are introduced by this revision.

Benefits:

- Stage 13 implementation can advance without falsely declaring Stage 12 complete;
- release tooling can be tested early against real CI artifacts;
- public release safety gates remain explicit and enforceable.

Risks:

- parallel Stage 12/13 work increases status-tracking complexity;
- development artifacts could be mistaken for releasable builds unless the pipeline labels and blocks them correctly.

Mitigation: development-only release tooling must fail closed for production release mode while `config.Version` is still a development version, and Stage 14 remains prohibited until both Stage 12 and Stage 13 acceptance gates are closed.


## 34. v0.2.8 change log

Changes relative to v0.2.7:

1. Record explicit project direction that Microsoft Authenticode and Apple Developer ID/notarization identities are not obtainable for the project owner and therefore cannot remain mandatory Testnet release gates.
2. Replace mandatory OS-vendor code-signing prerequisites with a mandatory cryptographic provenance model: SHA-256 + canonical release manifest + keyless GitHub/Sigstore artifact attestation bound to the public repository, workflow and exact commit.
3. Keep Windows Authenticode and macOS Developer ID/notarization as optional future hardening only when legitimately obtainable. The project must never fake, borrow or misrepresent those identities.
4. Require unsigned/unnotarized Windows/macOS Testnet artifacts to be labelled truthfully and require user-facing verification instructions. Platform security warnings may be bypassed only per normal per-application OS controls; users must not be told to disable security protections globally.
5. Require a clean environment to verify checksums and provenance attestations before public Testnet distribution. A frozen release must verify against its exact source commit and expected VALDR workflow identity.
6. Preserve the Stage 12 manual Windows acceptance requirement. Removing inaccessible third-party certificates does not waive functional/manual Desktop acceptance.
7. Preserve `config.Version = "0.2.0-dev"` until a Testnet release candidate is deliberately frozen. Development attestations prove provenance only and do not convert a development build into a public Testnet release.
8. No consensus, transaction, P2P wire, storage schema, wallet encryption, Testnet Genesis, PoW, emission or Mainnet enablement changes are introduced by this revision.

Benefits:

- Stage 13 no longer depends on commercial/platform identities that the project owner cannot legitimately obtain;
- every downloadable artifact can still be tied cryptographically to the public VALDR source repository, exact workflow and exact commit;
- no long-lived private signing key is required for the keyless GitHub/Sigstore path;
- release verification remains independently checkable by users.

Risks:

- Windows SmartScreen/Smart App Control and macOS Gatekeeper may warn about or block unsigned/unnotarized binaries;
- provenance attestation proves origin/integrity, not that the software is safe or endorsed by Microsoft/Apple/GitHub/Sigstore;
- public-good attestation infrastructure remains an external dependency for online verification, so offline verification material/instructions should be retained for frozen releases.

Mitigation: prominently disclose unsigned OS-vendor status, publish SHA-256 and provenance verification commands, preserve exact release metadata, and keep all Stage 12 functional/security gates mandatory.

No Mainnet launch is authorized by v0.2.8.


## 35. v0.2.9 change log

Changes relative to v0.2.8:

1. Replace `valdr-testnet-1` as the active product Testnet with a fresh profile `testnet2` / Chain ID `valdr-testnet-2`; historical Testnet1 remains preserved for compatibility and evidence.
2. Freeze a new Testnet2 Genesis with a calibrated initial target based on measured Windows single-thread mining evidence.
3. Keep the 60-second target block interval but reduce retarget interval from 60 to 10 blocks for the active Testnet; use a 540-second measured window and 135..2,160 second clamp.
4. Tighten the active Testnet PoW limit to 22 leading zero bits while preserving the 10-minute Testnet min-difficulty escape and deterministic recovery.
5. Change only the **active Testnet** fixed subsidy from 50 to **1 Testnet VDR per block** and make consensus reward lookup network-profile aware.
6. Keep legacy/devnet/Testnet1 subsidy behavior unchanged.
7. Extend the miner RPC timeout so a correct 60-second-target block does not fail merely because solving takes longer than 30 seconds.
8. Require Desktop, packaging metadata, deployment examples and official website to move to Testnet2 after the new Core baseline is CI-verified.
9. Supersede Testnet1 Stage 12/13 acceptance artifacts for final acceptance. They remain useful historical evidence but cannot certify the new Testnet2 consensus.
10. Do **not** freeze Mainnet reward, halving, maximum supply or coinbase maturity. Those remain Stage 15 specification work.

Why this changes the previous specification:

Windows manual QA demonstrated that Testnet1 was far easier than intended: a single CPU thread around 330–379 kH/s found blocks in fractions of a second to around one second and accepted dozens of blocks in a short run. Leaving those parameters unchanged would allow a single ordinary machine to advance the early Testnet chain hundreds of times faster than the intended 60-second cadence.

Benefits:

- early Testnet mining is materially harder and difficulty responds much sooner;
- Testnet issuance is reduced from 50 to 1 VDR/block without pretending to define Mainnet economics;
- old Testnet state cannot be mistaken for the recalibrated chain because Chain ID and Genesis both change;
- CI remains practical because bootstrap work is calibrated in millions, not hundreds of millions, of hashes.

Risks:

- this is a consensus break and requires a fresh Testnet chain/data namespace;
- all final Stage 12/13 acceptance evidence must be regenerated on Testnet2;
- calibration is based on a limited real-machine sample and must be validated again during distributed Testnet;
- a large hashrate shock can still temporarily move block cadence until the next 10-block retarget.

Mitigation:

- retain historical Testnet1 support rather than mutating it in place;
- keep chainwork-based fork choice, bounded retarget adjustment and Testnet min-difficulty recovery;
- monitor block intervals and hashrate during Stage 14 before any Mainnet specification is frozen.

No Mainnet launch is authorized by v0.2.9.
