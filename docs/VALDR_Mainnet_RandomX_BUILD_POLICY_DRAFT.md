# VALDR Mainnet RandomX dependency/build policy — DRAFT

Status: **M3 candidate evidence; Mainnet disabled**

Upstream dependency:
- project: tevador/RandomX
- release label: v1.2.3
- exact upstream commit: `12f2c2ffe2108d6cf54c391fee33c8bc3646cdab`
- source URL: official upstream repository only

Rules:
1. Consensus/CI builds MUST resolve RandomX to the exact commit above, not only to a mutable tag name.
2. A different RandomX commit/version is a consensus-sensitive dependency change and requires a new Master-TZ revision before Mainnet freeze.
3. Official upstream RandomX tests must pass before VALDR native integration tests run.
4. VALDR golden vectors must match on Windows, Linux and macOS.
5. Node validation uses the isolated cgo adapter in `core/consensus/randomxnative`.
6. Default/Testnet2 builds remain independent of native RandomX through the non-native stub.
7. Mainnet-capable native builds require `CGO_ENABLED=1` and build tag `randomx_native`.
8. Validator mode uses RandomX cache/light mode; full-memory dataset mode is reserved for mining/benchmark paths unless a later approved specification changes this.
9. Build artifacts must map to an exact VALDR commit and exact RandomX upstream commit.
10. No runtime download or replacement of the RandomX consensus engine is allowed.

Evidence already present on the experiment branch:
- official RandomX vector verified;
- VALDR RandomX vector identical across supported CI operating systems;
- native adapter CI verified;
- persisted candidate end-to-end validation CI verified.

This document records the dependency/build-policy candidate only. It does not freeze M3 and does not enable Mainnet.
