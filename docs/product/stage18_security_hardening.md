# VALDR Testnet2 Stage 18 security hardening checklist

**Master baseline:** `docs/VALDR_Master_TZ_v0.2.13.md`  
**Scope:** Testnet2 technical hardening pulled forward before Stage 14.  
**Audit status:** **NOT AUDITED.** No independent external security audit/report has been performed.

This checklist maps the Stage 18 requirements to repository tests and CI gates. A green CI run on the exact commit containing this file is required before the Stage 18 automated gate may be recorded as PASS.

| Requirement | Evidence / gate | Status |
| --- | --- | --- |
| Parser/decoder fuzzing | `p2p/fuzz_v2_test.go`, RPC and wallet fuzz targets; CI step **Parser and decoder fuzz gate** | automated |
| DB crash/restart/corruption recovery | `storage/crash_atomicity_v02_test.go`, `storage/corruption_v02_test.go` | automated |
| Deep/repeated reorg | `core/blockchain/reorg_deep_v02_test.go`, runtime reorg/restart gates | automated |
| P2P flood/oversize/checksum/malformed | P2P protection/runtime tests plus v2 frame fuzzing and per-message payload limits | automated |
| RPC exposure | `rpc/security_v02_test.go`; localhost Desktop RPC contract | automated |
| Wallet tamper/wrong-password/migration | `wallet/security_v02_test.go`, `wallet/wallet_v2_test.go`, backup/import verification | automated |
| Desktop lifecycle | cross-platform runtime E2E covers first-run, node crash detection/restart, wallet backup/restore and mining lifecycle | automated |
| Desktop duplicate managed child protection | `TestDesktopNodeRejectsDuplicateManagedStart`, `TestDesktopMinerRejectsDuplicateManagedStart` | automated |
| Release upgrade/rollback | `scripts/stage18-release-upgrade-rollback.sh` performs a Linux package-manager upgrade/downgrade cycle and verifies VALDR user data survives both directions and uninstall | automated Linux package contract |
| Dependency audit | `govulncheck` for Core/Desktop plus frontend `npm audit --audit-level=high` | automated |
| Static analysis | `go vet ./...` plus Desktop backend vet | automated |
| Race tests | `go test -race ./...` | automated |
| Deterministic/reproducible build analysis | CI rebuilds `valdrd` and `valdr-miner` twice with `-trimpath -buildvcs=false` and byte-compares them | practical deterministic Core check |
| External review | Perform only if a competent independent reviewer becomes available; no review is claimed today | not performed / conditional |

## Reproducibility boundary

The deterministic check is deliberately limited to canonical Go node/miner binaries built twice in the same clean CI environment and toolchain. NSIS, DMG and AppImage containers may embed tool- or filesystem-generated metadata, so VALDR does **not** claim that final cross-platform packages are bit-for-bit reproducible. Release authenticity continues to rely on exact commit binding, SHA-256, the canonical manifest and GitHub/Sigstore provenance.

## Upgrade/rollback boundary

The Stage 18 Linux test creates two **CI-only synthetic package revisions** from the exact candidate package. It validates package replacement/downgrade mechanics and preservation of the real VALDR Linux user-data location. The synthetic revisions are never release artifacts and are never published. A future real RC-to-RC migration may add additional compatibility coverage if persistent schemas change.

## Security gate interpretation

A green active CI run demonstrates that the automated hardening checklist above passes for that exact commit. It does not prove absence of vulnerabilities and must not be described as an external audit.

Stage 12C manual Windows acceptance, Stage 13D RC freeze and Stage 14A independent multi-machine Testnet validation remain separate gates.
