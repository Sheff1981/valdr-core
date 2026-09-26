# VALDR Stage 13 release identity and version contract

**Status:** ACTIVE / STAGE 13 IN PROGRESS — development-only release plumbing  
**Date:** 2026-09-26  
**Baseline:** `docs/VALDR_Master_TZ_v0.2.11.md`  
**Branch:** `valdr-v0.2`

This document defines the release metadata rules used by the active Stage 13 implementation. Stage 12 remains incomplete until its outstanding manual acceptance is closed; public release remains blocked.

## 1. Current repository facts

- application/core version constant: `config.Version = "0.2.0-dev"`;
- active Testnet profile: `testnet2`;
- active Testnet Chain ID: `valdr-testnet-2`;
- Testnet protocol range: min 2 / max 2;
- current Stage 12 Desktop framework: Wails v2.12.0;
- there are currently no git tags in the repository;
- Master-TZ version `v0.2.11` is the specification revision, not automatically the application release version.

Therefore the release pipeline must never derive the application version from the master-spec filename.

## 2. Version source of truth

Before the first Stage 13 release candidate, VALDR must have exactly one application release version source that is consumed by:

- `valdrd version`;
- `valdr-miner version`;
- `valdr-cli version`;
- Desktop package metadata;
- Windows installer metadata;
- macOS bundle metadata;
- Linux package metadata;
- release manifest;
- release artifact filenames.

The existing `config.Version` remains the logical source unless implementation proves that a generated build-time value is required.

A release job must fail if package/manifest versions disagree with the binary-reported version.

## 3. Specification version is separate

The following are separate identities:

- Master-TZ revision: currently `v0.2.11`;
- product/application version: currently development value `0.2.0-dev`;
- P2P protocol version: Testnet protocol 2;
- Chain ID: `valdr-testnet-2`;
- git commit SHA: exact source snapshot.

No release UI or manifest may collapse these into one ambiguous "version" field.

## 4. Planned release-candidate naming

The first cryptographically provenance-attested public Testnet build should use an explicit pre-release/Testnet application version rather than silently publishing `0.2.0-dev`.

Recommended pattern for implementation review:

`0.2.0-testnet.N`

and git tag:

`v0.2.0-testnet.N`

where `N` increases for externally distributed Testnet builds.

This naming is a release-process proposal only. It is not yet written into binaries and does not require a Master-TZ revision unless the project chooses a different release/versioning policy that changes normative release behavior.

## 5. Artifact filename contract

Planned canonical shape:

`VALDR-Desktop-<version>-<os>-<arch>.<ext>`

Examples of format only:

- `VALDR-Desktop-0.2.0-testnet.1-windows-x64-setup.exe`
- `VALDR-Desktop-0.2.0-testnet.1-macos-arm64.dmg`
- `VALDR-Desktop-0.2.0-testnet.1-macos-x64.dmg`
- `VALDR-Desktop-0.2.0-testnet.1-linux-x64.AppImage`
- `VALDR-Desktop-0.2.0-testnet.1-linux-amd64.deb`

Do not put mutable labels such as `latest` into the canonical artifact filename.

## 6. Release manifest identity

Every manifest must bind:

- product version;
- exact 40-character git commit;
- network = `testnet2`;
- Chain ID = `valdr-testnet-2`;
- protocol min/max = 2/2;
- release UTC timestamp;
- artifact filename;
- OS/architecture;
- byte size;
- SHA-256;
- truthful OS-vendor signing status;
- truthful notarization status where relevant;
- provenance method and requirement;
- minimum supported OS.

The release manifest and package artifacts must be covered by the GitHub/Sigstore keyless provenance attestation required by Master-TZ v0.2.11. Development manifests remain development-only even when provenance-attested and cannot satisfy §23 until the release-candidate version and remaining acceptance gates are frozen/green.

## 7. Minimum OS planning baseline

Based on the pinned Wails v2 line and current native CI targets, the initial Stage 13 compatibility baseline should not claim broader support than:

- Windows 10/11 x64;
- macOS 10.13+ Intel/AMD64;
- macOS 11.0+ ARM64;
- Linux x64 on the package/runtime baseline proven by release CI.

Linux minimum distribution/glibc compatibility is not yet proven and must be measured by the AppImage/.deb clean-environment jobs before publication.

These values are provisional release metadata until Stage 13 clean-machine tests make them evidence-backed.

## 8. Windows WebView2 policy to resolve in implementation

Wails Desktop requires Microsoft WebView2 on Windows.

Stage 13 must explicitly select and test a Wails WebView2 installer strategy rather than relying on an undocumented default. The decision must optimize normal-user installation reliability while avoiding a hidden dependency or unverified third-party download path.

Whichever strategy is selected must be recorded in the installer evidence and tested on a clean Windows environment where WebView2 availability is known.

## 9. Release provenance identity

Master-TZ v0.2.11 removes inaccessible Microsoft/Apple identities from mandatory Testnet acceptance.

The mandatory release identity is now the public repository/workflow identity:

- repository: `Sheff1981/valdr-core`;
- workflow: `.github/workflows/valdr-v02-ci.yml` for development provenance until a dedicated frozen release workflow is introduced;
- exact source commit SHA;
- GitHub Actions OIDC identity;
- Sigstore public-good short-lived signing certificate/transparency record.

No long-lived private signing key is required for this keyless path. Windows Authenticode and Apple Developer ID/notarization remain optional future hardening only and must never be faked or borrowed.

## 10. Current implementation gate

Master-TZ v0.2.11 authorizes CI-safe Stage 13 development plumbing and keyless provenance attestation while Stage 12 manual Windows QA remains pending.

Until Stage 12 is accepted:

- keep `config.Version = "0.2.0-dev"`;
- do not create a public Testnet release tag;
- do not claim Microsoft Authenticode or Apple Developer ID/notarization unless those platform identities are actually present;
- do not publish release assets as an accepted Testnet release;
- do not start Stage 14.

The deterministic manifest/checksum generator must therefore require an explicit development mode while `config.Version` is a development version. A production Testnet manifest must fail closed until a non-development release-candidate version is deliberately frozen.

## 11. Implemented Stage 13 metadata plumbing

The repository provides `cmd/valdr-release-manifest` as the canonical manifest/checksum generator.

It derives application version from `config.Version`, Testnet identity from the compiled `testnet2` network profile, computes artifact byte sizes and SHA-256 hashes from the actual files, sorts artifacts deterministically, binds the exact git commit and explicit UTC release timestamp, rejects unsafe/duplicate/missing artifacts, and labels current dry-runs as development-only.

The manifest declares `provenance_method = github-sigstore-keyless` and `provenance_required = true`. The CI release-assembly job generates the actual attestation after the manifest/checksums exist, then a separate clean job verifies that attestation against the expected repository, workflow, ref and source commit.


## 12. Current Testnet2 R0 evidence

The active development identity above is CI-verified on exact source commit `ed9b1425847f4e391a678cca3cf87d3acf0d866c` by VALDR v0.2 CI run `36217224372` (#367), completed successfully on 2026-09-26. The same run assembled the cross-platform development artifacts, verified SHA-256/manifest consistency and GitHub/Sigstore provenance from a clean environment, and completed the non-public same-commit handoff.

This is development evidence only. Application version remains `0.2.0-dev`; no public Testnet RC or Mainnet release is implied.
