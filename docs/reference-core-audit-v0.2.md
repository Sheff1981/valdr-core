# VALDR Core Reference Audit — Bitcoin / Litecoin / Dogecoin / Kaspa / Monero

**Date:** 27 September 2026  
**VALDR branch:** `valdr-v0.2`  
**Working Master-TZ:** `docs/VALDR_Master_TZ_v0.2.13.md`  
**Purpose:** architecture/security review only. This document does not change consensus, wire, storage or wallet format.

## Pinned upstream references

- Bitcoin Core: `ed7dd7cf4e1561a97edf72eba29a67b14e28c717`
- Litecoin Core: `ec1b6489a900d09cf5991e220dce089c77a232a2`
- Dogecoin Core: `47a303a449999314abcf83bd1fc7e77dd4540fcd`
- rusty-kaspa: `01b532e8b553523216471682649693af92f0fd16`
- Monero: `e4fbca88b15f3cb2b6aa2d0e98639649f7da3d21`

All are pinned as Git submodules under `references/`.

## License boundary

- Bitcoin Core: MIT.
- Litecoin Core: MIT.
- Dogecoin Core: MIT.
- rusty-kaspa: ISC.
- Monero: BSD-3-Clause, with some inherited MIT components.

Reference reading is unrestricted by architecture. Any direct code reuse must preserve the applicable copyright/license notices. Prefer independent Go implementations of the ideas rather than mechanical copying from C++/Rust code.

---

# 1. VALDR current architecture checked against the references

The current VALDR implementation already contains the correct v0.2 structural direction:

- UTXO accounting and implicit fees;
- cumulative chainwork;
- side branches and reorganization;
- persisted undo information;
- BadgerDB storage v2;
- exact 256-bit target for v2;
- Median-Time-Past and future-time checks;
- P2P v2 framing with network magic/version/type/length/checksum;
- headers-first synchronization primitives and block locator;
- bounded mempool and deterministic fee-rate ordering;
- encrypted wallet v2 using scrypt + AES-256-GCM;
- local signing and no private keys over RPC/P2P;
- localhost-first RPC;
- persistent peer/bootstrap work;
- dedicated tests around reorg, corruption, crash atomicity, P2P parsing and wallet security.

This means the correct strategy is **hardening and extracting proven invariants**, not replacing VALDR with a fork of another coin.

---

# 2. Bitcoin Core — primary reference

## 2.1 Chainstate / validation

Primary areas inspected:

- `src/validation.cpp`
- `src/validation.h`
- `src/node/chainstate.cpp`
- `src/pow.cpp`
- `src/net_processing.cpp`
- `src/headerssync.*`
- `src/txmempool.*`
- `doc/files.md`
- policy documentation and fuzz tests.

Bitcoin separates:
1. header/block validity,
2. chain index,
3. UTXO state,
4. block/undo persistence,
5. mempool policy,
6. peer processing.

Its `ConnectBlock` / `DisconnectBlock` model is the closest conceptual match to VALDR reorg handling.

### Adopt/keep in VALDR

- Reorg must be state transition, not “replace slice of blocks”.
- Disconnect and reconnect must be individually deterministic and testable.
- UTXO updates and active-tip switch must be atomic at the storage transaction boundary.
- Mempool reconsideration belongs after chain state changes.
- Side branches remain validated/persisted instead of discarded merely because they are not active.
- Equal-work tie must not cause unstable oscillation.
- Header-first sync must validate as much as possible before downloading bodies.
- Chainwork, not height, decides the best branch.
- Keep separate consensus validity from relay/mining policy.
- Persist enough undo data to reverse active blocks without reconstructing history from scratch.

### Hardening opportunity for VALDR

VALDR currently stores per-node UTXO snapshots in memory for branch validation. This is useful for correctness and the current scale, but it is not a long-term scalable representation. Do **not** redesign this during current v0.2 productization unless profiling shows a real blocker. Record it for a future scalability review.

## 2.2 Storage

Bitcoin uses distinct raw block data, undo data, block index and UTXO chainstate.

VALDR intentionally uses BadgerDB and should not imitate Bitcoin's exact files. The transferable principle is separation of logical state and explicit recovery metadata.

### Keep

- `block/<hash>`, active height mapping, header metadata, UTXO, undo, tx index, active tip and chainwork metadata.
- Startup integrity checks before network/RPC service.
- Crash/restart and corruption tests.
- Never silently “repair” network identity mismatch.

### Add to hardening checklist

- Fault injection at every logical write boundary of candidate-block commit.
- Verify no active height maps to a side-chain block after interrupted commit.
- Verify undo record and UTXO hash remain mutually consistent after restart.
- Verify DB rollback/retry cannot expose partially switched active tip.

## 2.3 P2P

Bitcoin's modern P2P design separates socket/network management from peer message processing, address management and ban policy.

### Transferable

- handshake completion gates normal traffic;
- bounded message processing;
- per-peer state;
- address manager independent from consensus;
- discouragement/ban policy independent from block validity;
- duplicate announcement suppression;
- headers synchronization resistant to peers advertising unusable chains;
- fuzzing of P2P decoders.

### VALDR-specific caution

VALDR P2P v2 currently uses strict UTF-8 JSON. That is acceptable for v0.2 and frozen by the Master-TZ. Do not replace with Bitcoin wire serialization now. Binary serialization may be considered only in a later protocol revision.

## 2.4 Mempool

Bitcoin has substantially more complex package/RBF/ancestor policy than VALDR needs.

### Keep VALDR simpler

Current v0.2 rules remain correct:
- no RBF;
- no unconfirmed-parent relay;
- bounded memory;
- deterministic conflicts;
- fee-rate ordering;
- local policy must never become consensus.

Do not import package relay, CPFP/RBF or ancestor graphs into v0.2.

## 2.5 Wallet

Bitcoin wallet functionality is far beyond current VALDR scope.

Useful principles:
- encrypted secret material;
- lock/unlock boundary;
- never expose secrets through ordinary RPC/UI surfaces;
- migrations must be explicit and testable;
- metadata and key material must remain consistent.

VALDR's scrypt + AES-256-GCM v2 wallet is aligned with its own specification and should not be replaced with Bitcoin wallet architecture.

---

# 3. Litecoin Core — divergence reference

Litecoin is valuable because it demonstrates how a Bitcoin-derived chain changes consensus while retaining many upstream architectural patterns.

## Useful lessons

- Network identity and consensus parameters must be frozen per network profile.
- PoW/difficulty changes are consensus-critical and must have deterministic vectors.
- A mature fork accumulates substantial divergence; copying patches blindly from upstream is dangerous.
- Optional protocol extensions such as MWEB create large cross-cutting validation/storage/wallet complexity.

## For VALDR

Adopt:
- disciplined separation of per-network constants;
- explicit testnet/mainnet parameter review;
- tests for difficulty boundaries and network separation.

Reject for current scope:
- Scrypt PoW migration;
- MWEB;
- Litecoin-specific address/script behavior;
- direct Bitcoin/Litecoin code inheritance.

VALDR's SHA-256 PoW and current Testnet2 rules remain unchanged.

---

# 4. Dogecoin Core — mining and policy reference

Dogecoin is Bitcoin-derived but contains materially different economics/mining behavior and older inherited components.

## Useful areas

- miner transaction / reward accounting;
- practical fee/dust policy documentation;
- peer handshake limits;
- block/UTXO/undo persistence;
- network-level robustness work;
- AuxPoW demonstrates how merged mining radically changes block validation.

## For VALDR

Adopt conceptually:
- explicit fee policy documentation;
- miner template/reward validation tests;
- strict P2P handshake timeout and message-size rejection;
- operational clarity around relay policy vs consensus.

Do not adopt:
- AuxPoW / merged mining;
- Dogecoin emission schedule;
- DOGE dust rules;
- legacy network/version assumptions;
- old Bitcoin-derived RPC behavior simply because it is battle-tested.

If merged mining is ever considered, it requires a separate consensus specification and is not a “miner-only” change.

---

# 5. rusty-kaspa — concurrency/P2P/wallet reference

Kaspa's consensus is a BlockDAG/GHOSTDAG design. It is **not** a chainwork UTXO-chain template for VALDR.

## Do not transfer

- GHOSTDAG;
- pruning proofs;
- DAG reachability;
- selected-parent/mergeset logic;
- DAG-specific difficulty/finality assumptions.

Those would replace VALDR's core consensus model.

## Useful engineering patterns

### P2P
Kaspa separates handshake, routing and protocol flows cleanly.

Useful ideas for VALDR:
- explicit connection state machine;
- message routes with bounded queues;
- backpressure instead of unbounded goroutine/message growth;
- protocol-version checks before expensive work;
- clear separation between transport, handshake and application flows.

### Mempool
Kaspa has explicit policy objects and RBF logic.

For VALDR the architectural separation is useful; RBF itself is not.

### Wallet
Kaspa has a dedicated wallet encryption layer and uses Argon2 in its wallet code.

VALDR v2 explicitly freezes scrypt parameters. Do not change KDF in the current wallet format. A future wallet-v3 review could benchmark Argon2id vs scrypt and define a migration if there is a concrete security/UX reason.

---

# 6. Monero — DB/P2P/security reference

Monero's privacy transaction model is very different from VALDR, but its daemon engineering contains useful lessons.

## Storage

Monero's LMDB-backed blockchain implementation emphasizes transactional DB operations and batch behavior.

Useful for VALDR:
- crash consistency as a first-class property;
- clearly bounded read/write transactions;
- avoid holding long-lived write transactions across expensive validation;
- verify startup DB state before serving peers.

BadgerDB remains the VALDR backend.

## P2P

Monero's Levin protocol and network-zone machinery are not directly applicable, but useful principles include:
- explicit connection context;
- peer-zone separation;
- bounded packet processing;
- disconnecting malformed/invalid peers;
- network privacy/transport policy kept separate from consensus.

VALDR should retain its simpler current network model. Tor/I2P and privacy zones remain future decisions.

## Mempool

Monero revalidates pool entries and distinguishes relay behavior from block acceptance.

This supports the existing VALDR rule:
- chain changes trigger mempool revalidation;
- relay policy is not consensus.

## Wallet

Monero has mature encrypted wallet/key handling and secret lifetime discipline.

Useful hardening targets for VALDR:
- minimize lifetime of plaintext private-key material after decrypt;
- clear temporary byte buffers where practical;
- ensure wrong password and modified ciphertext are indistinguishable at the authentication boundary;
- never log decrypted key material or password-derived data.

---

# 7. Cross-reference matrix

| VALDR subsystem | Primary reference | Secondary reference | Do not import |
|---|---|---|---|
| chainwork/fork choice | Bitcoin | Litecoin | Kaspa GHOSTDAG |
| reorg/undo | Bitcoin | Dogecoin | DAG logic |
| UTXO validation | Bitcoin | Litecoin/Dogecoin | Monero privacy tx semantics |
| storage atomicity | Bitcoin concepts | Monero LMDB patterns | exact foreign DB layout |
| difficulty | Bitcoin methodology | Litecoin divergence lessons | foreign constants |
| P2P framing/state | Bitcoin | Kaspa | foreign wire protocol |
| peer discovery | Bitcoin/Litecoin | Monero | trusted seed consensus |
| mempool | Bitcoin separation | Kaspa/Monero | RBF/package relay now |
| mining | Bitcoin | Dogecoin | AuxPoW now |
| wallet security | VALDR spec first | Kaspa/Monero | replacing v2 KDF silently |
| RPC security | Bitcoin operational patterns | Monero daemon patterns | public privileged RPC |
| fuzz/security tests | Bitcoin/Kaspa/Monero | all | copying test vectors without adapting |

---

# 8. Concrete VALDR hardening backlog derived from the review

These are **implementation/test hardening ideas**, not protocol changes.

## P0 — current Master-TZ priority compatible

1. P2P state-machine tests:
   - pre-handshake message rejection;
   - duplicate hello/hello_ack;
   - version intersection edge cases;
   - wrong magic/chain ID;
   - slow/partial header frame;
   - peer that advertises height/work but cannot provide a valid header chain.

2. P2P resource tests:
   - bounded concurrent inbound handshakes;
   - queue/backpressure behavior;
   - repeated inv duplicates;
   - invalid peers list amplification attempt;
   - ban/score expiry after restart if persisted, or explicit non-persistence if not.

3. Reorg tests:
   - repeated A->B->A-like best-chain changes driven by increasing chainwork;
   - deeper valid branch with fewer blocks but more work;
   - reorg while mempool contains spends from disconnected blocks;
   - transaction present in both old and new branches;
   - disconnected coinbase never re-enters mempool;
   - crash during candidate persistence and crash during active-tip switch.

4. Storage tests:
   - fault injection around block/header/undo/UTXO/height/tip writes;
   - missing/corrupt undo record;
   - active-tip metadata mismatch;
   - side-branch block present but header metadata missing;
   - deterministic UTXO-state hash across restart.

5. Wallet:
   - repeated unlock/lock cycles;
   - malformed KDF metadata;
   - nonce/ciphertext/AAD modification;
   - truncated file;
   - atomic save/replace failure;
   - verify temporary secrets are never included in exported diagnostics/logs.

6. RPC:
   - reject multiple JSON values/unknown fields/oversize bodies;
   - browser-origin rejection remains covered;
   - privileged remote binding remains fail-closed;
   - slow-client/body resource behavior;
   - all error paths avoid secret/config leakage.

## P1 — useful after P0

- explicit internal peer connection state enum instead of implicit boolean combinations;
- bounded worker/queue instrumentation for P2P;
- richer per-peer diagnostics;
- DB consistency command that verifies cross-index invariants;
- deterministic property tests for connect/disconnect symmetry;
- fuzz corpus seeded with real valid v2 frames then mutated.

## Deferred

- binary P2P serialization;
- RBF/package relay;
- pruning;
- compact blocks;
- Tor/I2P;
- Argon2 wallet-v3 migration;
- AuxPoW/merged mining;
- Kaspa/DAG consensus concepts.

---

# 9. Architectural conclusion

The five references do **not** justify changing VALDR's architecture.

The strongest match is Bitcoin Core for chainstate, validation, chainwork, reorg, UTXO and headers-first synchronization. Litecoin and Dogecoin are useful primarily as evidence about long-lived Bitcoin-derived divergence and mining/policy choices. Kaspa is useful for modern asynchronous P2P separation and wallet engineering but not consensus. Monero is useful for DB/P2P/security discipline but not transaction/consensus semantics.

The correct next engineering action remains the active Master-TZ v0.2.13 queue. Reference-derived work should be folded only into the permitted hardening slices and must not silently alter frozen Testnet2 consensus/network/wallet-format values.
