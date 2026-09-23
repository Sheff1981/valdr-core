# VALDR / MASTER TECHNICAL SPECIFICATION v0.2.1

**Расширение Testnet, усиление протокола и подготовка публичных нод**  
**Дата:** 23 сентября 2026  
**Статус:** рабочий мастер-ТЗ / implementation baseline; consensus-format amendment v0.2.1  
**Baseline:** VALDR Devnet v0.1 commit `3f322cf8f519cfe8dc350ffc83081c8b82bcd4be`

> v0.2.1 не переписывает завершённый v0.1. Frozen reference v0.1: `release/valdr-devnet-v0.1`. v0.2 остаётся сохранённой предыдущей редакцией; эта версия уточняет consensus-format, необходимый для Difficulty v2 и headers-first sync.

## 1. Цель v0.2

Развить собственный VALDR blockchain из локального Devnet v0.1 в hardened Devnet/Testnet с public nodes, full synchronization, chainwork/reorg, fees, mempool policy, explorer, Docker/Linux deployment и draft будущей Mainnet-спецификации.

В v0.2 не входят: запуск Mainnet, ICO/presale, продажа VDR, exchange listing, staking, smart contracts, NFT, bridge, mobile app и инвестиционные обещания.

## 2. База v0.1

Уже реализованы: Go blockchain, fixed Genesis, SHA-256 PoW, UTXO, ECDSA P-256 wallet/VDR1 addresses, signed transactions, 50 VDR coinbase, mempool, P2P, three-node sync, RPC/CLI, miner, restart-safe persistence, structured logs, security tests и final executable integration.

## 3. Обязательные направления v0.2

- полноценный block explorer;
- seed nodes;
- improved difficulty adjustment;
- transaction fees;
- explicit mempool policy;
- full synchronization;
- network protocol versioning;
- P2P protection;
- testnet profile;
- Docker;
- Linux server deployment;
- public test nodes;
- future Mainnet specification draft.

Дополнительно перед public Testnet обязательны storage v2 и encrypted wallet files.

## 4. Network profiles

| Параметр | Legacy v0.1 | Devnet v0.2 | Testnet v0.2 |
| --- | --- | --- | --- |
| Chain ID | valdr-devnet-1 | valdr-devnet-2 | valdr-testnet-1 |
| P2P protocol | v1 | v2 | v2 |
| P2P port | 7333 | 7333 | 17333 |
| RPC port | 7332 | 7332 | 17332 localhost |
| Address prefix | VDR1 | VDR1 | VDR1 |
| Target block | 60 s | 60 s | 60 s |
| Initial subsidy | 50 VDR | 50 VDR | 50 testnet VDR |
| Block header version | 1 | 2 | 2 |

Consensus-critical параметры compile-time/network-profile only. Runtime config не может менять правила валидности.

## 5. Storage v2

Target backend: BadgerDB, версия фиксируется в `go.mod` при реализации.

Schema:
- `meta/*`: schema/network/tip/height/chainwork;
- `block/<hash>`: full block;
- `height/<height>`: active-chain hash;
- `header/<hash>`: parent/height/target/chainwork/status;
- `tx/<txid>`: confirmed tx location;
- `utxo/<txid>:<index>`: UTXO;
- `undo/<blockhash>`: disconnect data;
- `migration/*`: migration state.

Принятие блока и reorg должны быть atomic DB transactions.

### Migration v0.1 -> v0.2

`valdrd migrate --from-v0.1 <data> --network valdr-devnet-1`

Migration читает `blockchain.json`, replay-ит блоки через validator, пишет новую DB отдельно и перед success сравнивает height, tip hash, txids и UTXO-set hash. Original v0.1 data автоматически не удаляется. Public Testnet имеет fresh Genesis.

## 6. Block/transaction limits

- max block serialized size: 1,000,000 bytes;
- max transaction serialized size: 100,000 bytes;
- canonical serialization документируется и покрывается golden vectors;
- SHA-256 block hash сохраняется в v0.2.

### 6.1 Block header v2 / exact target encoding

Legacy `valdr-devnet-1` сохраняет block header version 1 и его байтовый формат без изменений.

`valdr-devnet-2` и `valdr-testnet-1` используют block header version 2.

Для header v2:
- consensus target хранится как exact unsigned 256-bit integer;
- JSON/API representation: ровно 64 lowercase hexadecimal symbols;
- canonical hashed representation: ровно 32 big-endian bytes, без length prefix;
- target должен быть `> 0` и `<= network PoW limit`;
- legacy field `difficulty uint64` не является consensus-полем header v2 и должен иметь значение 0 для v2 blocks;
- block hash = SHA-256(canonical header v2 bytes).

Canonical header v2 bytes:

```text
version uint32 big-endian (=2)
height uint64 big-endian
previous_block_hash length uint64 + ASCII hex bytes
merkle_root length uint64 + ASCII hex bytes
timestamp uint64 big-endian
target 32 raw big-endian bytes
nonce uint64 big-endian
chain_id length uint64 + UTF-8 bytes
extra_data length uint64 + UTF-8 bytes
```

Target входит в hash preimage. Поэтому изменение target меняет block hash и не является unhashed metadata.

Network profile определяет допустимую block-header version. Смешивание v1/v2 blocks внутри одной Devnet2/Testnet chain запрещено.

Devnet2/Testnet Genesis используют version 2 и fresh network-specific hash. Testnet Genesis окончательно замораживается отдельным launch commit после прохождения consensus suite, как указано в §19.

## 7. Timestamp + difficulty v2

Timestamp:
- > Median-Time-Past previous 11 blocks;
- <= local system time + 2 hours.

Difficulty:
- target interval 60 s;
- retarget every 60 blocks;
- target timespan 3,600 s;
- actual timespan = timestamp последнего header предыдущего 60-block window минус timestamp первого header этого же window;
- retarget candidate не участвует в измерении timespan;
- actual timespan clamp 900..14,400 s;
- `new_target = old_target * actual / target`, bounded by PoW limit;
- результат хранится в exact 256-bit `target` header v2 без округления/compact conversion;
- integer/big-int deterministic arithmetic;
- Testnet may use min-difficulty escape after 10 minutes without block, with deterministic return to last non-special target.

## 8. Chainwork, fork choice, reorg

`work = floor(2^256 / (target + 1))`.

Active chain = valid branch with greatest cumulative chainwork. Equal-work tie keeps current active tip.

Side branches are stored/validated.

Reorg:
1. find common ancestor;
2. disconnect old active blocks using `undo/*`;
3. connect new branch through normal validation;
4. switch tip atomically;
5. reconsider non-coinbase disconnected transactions for mempool;
6. log depth, old/new tip, ancestor, chainwork delta.

## 9. Fees

No new fee field.

For normal tx:
- `input_total >= output_total`;
- `fee = inputs - outputs`;
- overspend invalid.

Coinbase may claim at most `subsidy(height) + block fees`. Under-claim allowed.

Testnet default min relay fee = 1 val/byte; this is policy, not consensus.

## 10. Mempool policy

Defaults:
- max 64 MiB;
- expiry 72 h;
- min Testnet relay 1 val/byte;
- conflicts: reject second unconfirmed spend;
- RBF disabled in v0.2;
- unconfirmed-parent spending not relayed in v0.2;
- eviction: lowest fee-rate then oldest;
- miner order: fee-rate descending, txid tie-break.

После connect/disconnect блока mempool revalidate against active UTXO.

## 11. Coinbase/miner v0.2

Coinbase remains index zero and exactly one per non-Genesis block.

Maximum claim = subsidy + fees.

Devnet/Testnet subsidy 50 VDR is for protocol testing only. Mainnet halving/emission remains separate Mainnet-spec decision.

Miner uses node template/mining API, selects tx by fee rate, respects block-size limit and exposes status: hashrate, accepted blocks, template height/difficulty/fees.

## 12. P2P protocol v2

Frame:
- network magic: first 4 bytes SHA-256(chain_id);
- protocol version uint16 (v2=2);
- message type uint16;
- payload length uint32;
- checksum first 4 bytes SHA-256(payload);
- payload strict UTF-8 JSON for v0.2.

hello fields:
`chain_id, protocol_min, protocol_max, node_id, services, listen_address, height, tip_hash, cumulative_chainwork, user_agent, timestamp, nonce`.

Highest mutually supported protocol wins. Wrong network/version disconnects before expensive processing.

Message set:
`hello, hello_ack, ping, pong, inv, get_data, block, tx, get_headers, headers, get_blocks, get_peers, peers, reject`.

## 13. Full synchronization

- block locator with recent hashes then exponential backoff to Genesis;
- headers batches up to 2,000;
- validate header continuity/PoW/timestamp/difficulty before bodies;
- determine common ancestor;
- request bounded block batches, default 32 outstanding;
- persist side branches;
- activate only greatest-chainwork branch;
- interrupted sync resumes after restart;
- offline node must converge without deleting DB.

## 14. Seed nodes

Public Testnet: minimum 3 stable seeds in >=2 independent regions/providers.

Seeds are normal full nodes, not trusted consensus authorities.

Compiled seed list + operator overrides. Network remains functional if a seed is offline.

## 15. P2P protection

Defaults:
- handshake timeout 5 s;
- idle timeout 5 min + ping;
- max inbound 64;
- outbound target 8;
- max inbound per IP 4;
- global frame payload max 4 MiB;
- per-message limits;
- token-bucket rate limiting;
- malformed score + disconnect threshold;
- temporary ban 1 h;
- bounded duplicate caches;
- public discovery rejects invalid/multicast/private/loopback gossip by default.

## 16. RPC v0.2 security

RPC localhost by default.

Methods split into read-only and privileged. `sendTransaction` and `mineBlock` privileged.

Remote privileged RPC requires bearer token from mode-0600 file. CORS off by default.

New read methods:
`getChainInfo, getBlockHeader, getFeeEstimate, getAddressUTXOs, getAddressHistory, getSyncStatus, getNodeInfo`.

## 17. Wallet v2

New wallet files must not contain plaintext private keys.

Target: scrypt-derived 256-bit key + AES-256-GCM, random salt and nonce.

Passphrase interactive or protected password-file descriptor; no `--password` CLI argument.

`wallet migrate` converts v0.1 plaintext wallet to encrypted v2. Private keys never cross RPC/P2P.

## 18. Block explorer

Add `cmd/valdr-explorer` as separate read-only service.

Search:
- block height/hash;
- txid;
- VDR address.

Views/API:
- latest blocks;
- block detail;
- transaction detail;
- address balance/history/UTXO;
- mempool;
- peers/network/difficulty/block intervals.

REST:
- `GET /api/v1/status`
- `GET /api/v1/blocks`
- `GET /api/v1/block/{height-or-hash}`
- `GET /api/v1/tx/{txid}`
- `GET /api/v1/address/{address}`
- `GET /api/v1/mempool`

Explorer detects reorg and rollback/reindexes. No signing/mining/privileged controls.

## 19. Testnet

Chain ID: `valdr-testnet-1`. P2P v2. P2P 17333. RPC 17332 localhost. Explorer local 8080 behind HTTPS reverse proxy.

Genesis is frozen once in a dedicated launch commit after consensus suite passes.

Testnet VDR has no promised monetary value.

## 20. Docker

- multi-stage image for valdrd/cli/miner/explorer;
- unprivileged runtime user;
- data `/var/lib/valdr`, config `/etc/valdr`;
- no embedded secrets;
- RPC not public by default;
- compose multi-node smoke with health checks.

## 21. Linux deployment

- current LTS Linux reference, x86_64 first;
- dedicated `valdr` service account;
- data `/var/lib/valdr`, config `/etc/valdr`;
- journald/stdout logs;
- systemd `Restart=on-failure`, `NoNewPrivileges=true`;
- firewall exposes P2P; RPC localhost; explorer via HTTPS reverse proxy;
- documented upgrade/backup/restore/verify/rollback.

## 22. Public Testnet nodes

Minimum 3 public full nodes, >=2 regions/providers. Recommended separate observer.

Private soak >=24 h before announcement. v0.2 stable requires >=7 days public operation without consensus split or manual DB repair.

Monitor height, tip, chainwork, peers, mempool, block interval, disk, RPC and restarts.

## 23. Observability/logging

Keep `NODE/P2P/BLOCK/TX/MINER/MEMPOOL/SYNC/ERROR`; add `STORAGE/RPC/EXPLORER/REORG`.

UTC timestamp + category + key/value context. Never log keys/passphrases/tokens. Optional JSON log mode and localhost metrics endpoint.

## 24. Security tests

Keep all v0.1 regression tests and add:
- fuzz P2P/transaction/RPC decoders;
- DB crash/restart/corruption;
- reorg/undo;
- oversize/bad-checksum/flood peer abuse;
- wallet wrong-password/ciphertext-tamper/migration;
- remote RPC exposure checks.

## 25. Mandatory v0.2 tests

- `difficulty_v2_test`;
- `fee_test`;
- `mempool_policy_test`;
- `chainwork_test` / `reorg_test`;
- `sync_v2_test`;
- `protocol_v2_test`;
- `seed_test`;
- `storage_v2_test`;
- `wallet_v2_test`;
- `explorer_test`;
- Docker/Linux smoke where CI permits.

## 26. CI readiness gate

Required semantics:

```text
go build ./...
go test ./...
go test -race ./...
bounded decoder fuzz runs
start-devnet-v0.2 smoke
sync test
reorg test
fees test
docker build
docker compose smoke
```

No v0.2 release reference before all gates pass.

## 27. Development sequence

| Stage | Deliverable | Gate |
| --- | --- | --- |
| 0 | Freeze v0.2 spec + branch valdr-v0.2 | spec committed, v0.1 unchanged |
| 1 | Storage v2 + migration | restart/migration/atomicity |
| 2 | Network profiles + P2P v2 | protocol vectors |
| 3 | Chainwork + reorg/undo | competing-chain tests |
| 4 | Difficulty v2 + timestamps | golden retarget tests |
| 5 | Fees + size limits | fee/coinbase tests |
| 6 | Mempool policy + miner ordering | policy tests |
| 7 | Full headers-first sync | fresh/resume/reorg sync |
| 8 | Seeds + P2P protection | abuse/bootstrap |
| 9 | Wallet encryption | no plaintext + migration |
| 10 | Explorer | API/UI/reorg tests |
| 11 | Testnet + Docker + Linux | container/service smoke |
| 12 | Private/public Testnet | 24h private + 7-day public stability |
| 13 | Mainnet spec draft | unresolved decisions explicit; no launch |

## 28. v0.2 acceptance criteria

- clean clone builds all binaries and passes full CI;
- fresh node seed-bootstrap + sync Genesis->tip;
- restart/resume after offline period;
- fork resolution by cumulative chainwork with correct UTXO after reorg;
- fees/miner collection correct;
- bounded mempool;
- encrypted wallet v2 + legacy migration;
- explorer search + reorg safety;
- >=3 public Testnet nodes and 7-day stability without split/manual DB repair;
- Docker/Linux restart with preserved chain;
- future Mainnet draft exists but Mainnet remains disabled.

## 29. Mainnet specification draft deliverables

Must explicitly freeze or resolve before any Mainnet launch:
- Chain ID, Genesis, magic, ports, seeds;
- block time/difficulty params;
- subsidy, halving, max supply enforcement, fees;
- coinbase maturity decision;
- block/tx limits and mempool defaults;
- address/network policy;
- fork/reorg/checkpoint policy;
- security review, reproducible builds, release signing, incident response;
- minimum public-node diversity and rollback plan.

## 30. Risks

| Risk | Mitigation |
| --- | --- |
| Consensus split | golden vectors, deterministic arithmetic, no runtime consensus override |
| DB corruption | transactional DB, schema version, integrity/crash tests |
| Reorg corruption | persisted undo + atomic reorg |
| P2P exhaustion | limits/rate score/bans/caches |
| Wallet theft | encrypted v2 + strict permissions |
| RPC abuse | localhost + privilege split + remote token |
| Explorer dependency | explorer read-only; node source of truth |
| Premature Mainnet assumptions | Mainnet disabled until separate approved spec |

## 31. Source control

- Frozen v0.1: `release/valdr-devnet-v0.1` -> `3f322cf8f519cfe8dc350ffc83081c8b82bcd4be`.
- v0.2 work: `valdr-v0.2`.
- Master spec v0.2 remains preserved; v0.2.1 is the active implementation baseline from the commit that adds this file.
- One stage at a time: spec -> implementation -> build -> tests -> runtime verification -> commit.
- Material consensus/wire/storage changes require a new spec revision/change log.
- Release refs only on fully tested commits.

### v0.2.1 change log

Consensus clarification required before Stage 7:
1. freeze block header version 2 for Devnet2/Testnet;
2. freeze exact 32-byte big-endian target in the hashed v2 header;
3. keep legacy v1 header byte-for-byte unchanged;
4. freeze the exact retarget-window timestamp endpoints already used by Stage 4 implementation.

Benefits: exact Difficulty v2 representation, deterministic independent implementations, headers-first validation before block bodies, no quantization of target.

Risks: Devnet2/Testnet header/block hashes are intentionally incompatible with v0.1 and therefore require fresh network-specific Genesis. Legacy v0.1 data/migration remains on header v1 and is not rewritten.

## 32. Definition of Done

Полный текущий мастер-план завершён только когда v0.1 остаётся воспроизводимо доступным, все v0.2 acceptance criteria выполнены, Public Testnet прошёл stability window, а draft Mainnet specification создан.

Это не означает Mainnet launch, продажу VDR или exchange listing.

## Appendix A - key defaults

- Devnet v0.2: `valdr-devnet-2`
- Testnet: `valdr-testnet-1`
- Protocol: 2
- Block target: 60 s
- Retarget: 60 blocks / 3,600 s
- Clamp: 0.25x..4x
- MTP: 11 blocks
- Future timestamp: +2 h
- Max block: 1,000,000 bytes
- Max tx: 100,000 bytes
- Testnet min relay: 1 val/byte
- Mempool: 64 MiB / 72 h
- Peers: outbound 8 / inbound 64
- Max P2P payload: 4 MiB
- Testnet seeds: minimum 3
- Prefix: VDR1, network-dependent checksum
- Header: v1 legacy; v2 Devnet2/Testnet
- v2 target: exact 32-byte big-endian in hash preimage / 64 lowercase hex in JSON
- Testnet subsidy: 50 VDR, protocol testing only

## Appendix B - new CLI surface

```text
valdrd migrate --from-v0.1 <data> --network valdr-devnet-1
valdrd start --network devnet2|testnet ...
valdrd verify-db --data <path>
valdr-cli chain info
valdr-cli sync status
valdr-cli fee estimate
valdr-cli address history <VDR1...>
valdr-cli wallet migrate <name-or-address>
valdr-explorer start --node http://127.0.0.1:17332 --listen 127.0.0.1:8080
```

## Appendix C - deliberately deferred beyond v0.2

- final Mainnet halving/emission curve;
- Mainnet coinbase maturity decision;
- signature algorithm replacement;
- compact binary P2P payload after v2;
- advanced RBF/package relay/compact blocks;
- smart contracts, staking, privacy extensions, bridges and token standards.
