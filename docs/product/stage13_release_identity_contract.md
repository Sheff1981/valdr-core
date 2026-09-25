# VALDR Stage 13 release identity and version contract

**Status:** DESIGN / PRE-STAGE-13  
**Date:** 2026-09-25  
**Baseline:** `docs/VALDR_Master_TZ_v0.2.5.md`  
**Branch:** `valdr-v0.2`

This document defines release metadata rules before installer implementation begins. It does not start Stage 13 and does not mark Stage 12 complete.

## 1. Current repository facts

- application/core version constant: `config.Version = "0.2.0-dev"`;
- active Testnet profile: `testnet`;
- active Testnet Chain ID: `valdr-testnet-1`;
- Testnet protocol range: min 2 / max 2;
- current Stage 12 Desktop framework: Wails v2.12.0;
- there are currently no git tags in the repository;
- Master-TZ version `v0.2.5` is the specification revision, not automatically the application release version.

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

- Master-TZ revision: currently `v0.2.5`;
- product/application version: currently development value `0.2.0-dev`;
- P2P protocol version: Testnet protocol 2;
- Chain ID: `valdr-testnet-1`;
- git commit SHA: exact source snapshot.

No release UI or manifest may collapse these into one ambiguous "version" field.

## 4. Planned release-candidate naming

The first signed public Testnet build should use an explicit pre-release/Testnet application version rather than silently publishing `0.2.0-dev`.

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
- network = `testnet`;
- Chain ID = `valdr-testnet-1`;
- protocol min/max = 2/2;
- release UTC timestamp;
- artifact filename;
- OS/architecture;
- byte size;
- SHA-256;
- signing status;
- notarization status where relevant;
- minimum supported OS.

The release manifest itself must be signed for a production Testnet release. An unsigned dry-run manifest must be labelled development-only and cannot satisfy Master-TZ §23.

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

## 9. Signing identities not yet available in repository

No signing credentials or public release-signing identity are stored in the repository, which is correct.

Before Stage 13 can satisfy the signed-release gate, the project still needs:

- Windows code-signing identity/certificate custody;
- Apple Developer ID/notarization credentials;
- release-manifest signing key/identity and published verification material.

These are external release prerequisites, not code defects. They must never be replaced with fake/test secrets in production release metadata.

## 10. Gate before implementation

Do not change `config.Version`, create release tags or publish release assets while Stage 12 manual Windows QA is pending.

After Stage 12 is accepted, the first Stage 13 implementation task is to freeze the chosen application release-candidate version and make package metadata derive from that single source.
