# Bitcoin Core 31.0 reference architecture review for VALDR

**Reference only. VALDR remains an independent blockchain and cryptocurrency.**

## Source identity

- Binary package reviewed: `bitcoin-31.0-win64.zip`
- Package SHA-256: `82fd2c504a0f20a31d4d13bd407783d6fc7bf17622d0ce85228a9b92694e03f0`
- Official upstream source tag reviewed: `bitcoin/bitcoin v31.0`
- Tag resolves to source commit: `6574cb40869b96b9ffc79c19dc8f4e467d60f321`
- Upstream tree contains roughly 2,920 files, including about 2,003 under `src/` and 413 under `test/`.

This review extracts mature engineering patterns. It does not copy Bitcoin network identity, consensus constants, monetary policy, address formats, scripts, protocol magic, ports, Genesis, branding or release identity into VALDR.

## Independence invariants

The following remain VALDR-owned and must not be imported from any reference project:

- chain/network identifiers;
- Genesis block and message;
- P2P magic;
- P2P/RPC ports;
- address/network encoding;
- PoW target/difficulty schedule;
- block interval;
- subsidy/emission/supply policy;
- transaction serialization and signing rules;
- wallet file format;
- bootstrap seeds;
- RPC names and security policy;
- release/version identity;
- Desktop UX and branding.

Any future change to those items requires an explicit VALDR specification revision and test vectors.

## Bitcoin Core decomposition

### 1. Consensus and validation boundary

Bitcoin Core isolates validity-critical logic from GUI and wallet concerns. Notable upstream areas include `src/consensus/`, `src/kernel/`, `validation.cpp`, primitives and script verification.

**VALDR mapping:** `core/consensus`, `core/block`, `core/transaction`, `core/utxo`, `core/blockchain`.

**Adoption rule:** keep consensus deterministic and UI/RPC independent. No Desktop-only rule may influence block validity.

### 2. Chainstate, UTXO and reorg recovery

Bitcoin Core separates block storage, chainstate/UTXO state, undo/reorg handling and indexes.

**VALDR mapping:** Badger-backed storage, indexed schema, chainwork selection, reorg persistence and corruption/crash tests.

**Adopt:** explicit state-versioning, restart/reorg recovery and verification tools.

**Do not copy:** Bitcoin LevelDB layout or file formats merely for compatibility.

### 3. P2P transport and peer lifecycle

Bitcoin Core separates socket transport, peer processing, address management, eviction, permissions, orphan handling, transaction download, transport versions and discovery.

**VALDR mapping:** `p2p/node.go`, v2 protocol, bootstrap, learned-peer cache, protection limits, headers-first sync and runtime flood tests.

**Adopt:** strict message-size limits, bounded caches, peer lifecycle state, bootstrap diversity, learned-peer persistence and abuse controls.

**Deferred:** Tor/I2P/CJDNS and NAT traversal remain separate future decisions.

### 4. Mempool and relay policy

Bitcoin Core treats policy as distinct from consensus and contains fee policy, replacement/package handling, transaction request scheduling and bounded mempool logic.

**VALDR mapping:** bounded mempool, fee/min-relay policy and miner template selection.

**Adopt:** preserve the hard boundary: policy may reject relay/mempool admission without silently becoming a consensus rule.

**Deferred:** advanced package/RBF/TRUC-style policy requires a VALDR-specific need and specification.

### 5. Mining boundary

Bitcoin Core exposes mining through node/interfaces rather than coupling private wallet keys directly into consensus.

**VALDR mapping:** separate `valdr-miner` process using local node RPC.

**Adopt for Stage 16:** freeze an explicit external mining contract before Mainnet; keep wallet private keys outside miner protocol; test template freshness, stale work, authentication and DoS limits.

### 6. Wallet boundary

Bitcoin Core has a large wallet subsystem and a separate `bitcoin-wallet` tool for wallet-file operations independent of a running full node.

**VALDR mapping:** encrypted wallet v2, local signing, backup/restore, migration and Desktop wallet service.

**Adopt:** keep encrypted key storage and wallet-file maintenance separable from chain consensus; preserve offline backup/verification capability.

**Mainnet blocker to resolve later:** VALDR address encoding is currently project-wide rather than profile-specific. Mainnet/Testnet address separation must be specified before Mainnet enablement; do not silently change Testnet2 wallet addresses in the v0.2 acceptance candidate.

### 7. RPC and operator boundary

Bitcoin Core treats RPC as a privileged operator surface with explicit bind/auth/whitelist controls.

**VALDR mapping:** localhost RPC, explicit request types and security tests.

**Adopt:** localhost-by-default, bounded request decoding, no implicit public exposure, explicit future authentication if remote operator RPC is ever introduced.

### 8. GUI boundary

Bitcoin Core GUI is a presentation/client layer and does not own consensus.

**VALDR mapping:** Wails Desktop controlling managed node/miner and encrypted wallet workflows.

**Adopt:** UI state must be derived from stable backend interfaces; GUI must not duplicate consensus calculations.

### 9. Interfaces and process separation

Bitcoin Core v31 contains formal node/wallet/mining interfaces plus IPC/multiprocess work.

**VALDR mapping:** current separate binaries already provide a useful process boundary: Desktop, `valdrd`, `valdr-miner`, Explorer and CLI.

**Adopt incrementally:** prefer narrow interfaces between components before adding IPC machinery. Do not import Cap'n Proto/libmultiprocess unless VALDR has a concrete requirement.

### 10. Storage/index separation

Bitcoin Core separates canonical chainstate from optional indexes.

**VALDR mapping:** canonical node state plus read-only Explorer.

**Adopt:** optional query/index services must be rebuildable and must not become consensus authority.

### 11. Release engineering

Bitcoin Core distributes distinct binaries and uses strong deterministic/reproducible-build discipline.

**VALDR mapping:** cross-platform Desktop packaging, exact commit binding, SHA-256 manifest, GitHub/Sigstore provenance and deterministic Core binary checks.

**Adopt:** maintain exact artifact/commit identity, clean verification and package/data separation.

### 12. Security/test architecture

Bitcoin Core has dedicated fuzz, sanitizer, cross-platform, functional, compatibility and release-test infrastructure.

**VALDR mapping:** unit/integration/runtime tests, fuzzing, race, static analysis, dependency audit, crash/corruption/reorg tests and cross-platform package gates.

**Adopt next:** grow adversarial scenario coverage by subsystem rather than merely increasing unit-test count.

## Adoption matrix

| Pattern | VALDR status | Action |
| --- | --- | --- |
| Consensus separated from GUI | implemented | preserve |
| Separate node binary | implemented | preserve |
| Separate miner process | implemented | formalize Mainnet mining contract in Stage 16 |
| Local encrypted wallet/signing | implemented | preserve |
| Node/wallet process boundary | implemented | narrow interfaces further only when needed |
| Policy distinct from consensus | implemented in basic form | preserve and test |
| Headers-first / chainwork sync | implemented | preserve |
| Bounded P2P/mempool resources | implemented | continue adversarial tests |
| Crash/corruption/reorg recovery | implemented/tested | preserve |
| Optional indexes/read-only Explorer | implemented | keep non-authoritative |
| Deterministic build analysis | implemented for Core binaries | preserve |
| Reproducible final package guarantee | not claimed | do not overclaim |
| Profile-specific address encoding | not implemented | specify before Mainnet; no v0.2 silent migration |
| Remote authenticated operator RPC | not needed today | deferred |
| Tor/I2P/CJDNS | not needed today | deferred |
| Pruning/snapshots | not needed for v0.2 | future evidence-based decision |
| PSBT/watch-only/hardware wallet | not needed for v0.2 | future product decision |
| Multiprocess IPC framework | not required | do not import complexity without need |

## Immediate implementation policy

1. No Bitcoin code is vendored into VALDR merely because it is mature.
2. Reference projects provide failure modes, interface patterns and test ideas.
3. Every adopted behavior is re-specified in VALDR terms and implemented against VALDR formats.
4. Consensus/network identity changes require a new Master-TZ revision and golden vectors.
5. Testnet2 acceptance candidate remains frozen except for defects; reference review must not destabilize Stage 12C.
6. Stage 15/16 use this review to design Mainnet identity and mining interfaces while Mainnet remains disabled.

## Next reference set

For useful comparison, prioritize mature UTXO/PoW projects with different design choices:

- Litecoin Core — Bitcoin-family architecture with different PoW/network/economic choices;
- Dogecoin Core — Bitcoin-family node with distinct issuance and AuxPoW/merged-mining history;
- Bitcoin Cash Node — UTXO architecture with materially different transaction/block policy evolution.

Other projects should be added only when they answer a concrete VALDR design question. Reference quantity is less important than extracting a tested reason for each adopted pattern.
