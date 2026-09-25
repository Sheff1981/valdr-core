# VALDR Stage 13 release/installer design plan

**Status:** PLANNED ONLY — implementation has not started.  
**Baseline:** `docs/VALDR_Master_TZ_v0.2.5.md` §§19 and 23.  
**Branch:** `valdr-v0.2`  
**Stage 12 state:** implementation/automated acceptance green; Windows manual GUI acceptance is still pending.

This plan may be prepared before Stage 12 is frozen, but no Stage 13 packaging/release implementation should be merged until the Stage 12 manual acceptance gate is closed.

## 1. Release invariants

Every published Desktop artifact must be traceable to one immutable git commit and one exact Testnet release version.

A release must publish, at minimum:

- Windows x64 installer;
- macOS ARM64 signed/notarized package;
- macOS Intel/AMD64 signed/notarized package, or a separately approved universal package;
- Linux x64 AppImage;
- Linux x64 .deb;
- SHA-256 checksums;
- machine-readable release manifest;
- detached signature for the release/checksum manifest;
- release notes and source-code link.

Mainnet must remain unavailable. During Testnet, release/download UX must not contain buy/sell/exchange calls to action.

## 2. Planned repository layout

The implementation pass should prefer the following minimal additions:

- `.github/workflows/valdr-v02-release.yml` — tag/manual release pipeline;
- `packaging/linux/valdr-desktop.desktop` — Linux desktop entry;
- `packaging/linux/valdr-desktop.svg` or generated PNG copy — package icon;
- `packaging/linux/debian/` — minimal Debian package metadata only when required by the chosen packaging command;
- `scripts/release-manifest.go` or a small Go command under `cmd/` — deterministic manifest/checksum generation;
- `scripts/release-verify.sh` / PowerShell equivalent only if needed for clean-environment verification;
- `docs/product/stage13_release_acceptance.md` — evidence matrix for §23.

Do not create duplicate build systems. Existing Wails Desktop build configuration remains authoritative for the application binary.

## 3. Windows packaging design

Target: Windows x64.

Planned flow:

1. build the Wails Desktop application;
2. build/bundle `valdrd.exe` and `valdr-miner.exe`;
3. create the Windows installer with the Wails/NSIS packaging path supported by the pinned Wails version;
4. when an Authenticode identity is available, sign the application binaries and final installer;
5. verify publisher identity;
6. install on a clean Windows runner/profile;
7. launch Desktop without a terminal;
8. verify uninstall removes program files while preserving user blockchain/wallet data unless the uninstaller explicitly asks otherwise.

Signing credentials must be injected only through protected CI secrets or an external signing service and must never be committed.

## 4. macOS packaging design

Targets: ARM64 and Intel/AMD64 unless a universal artifact is separately approved.

Planned flow:

1. build the Wails `.app`;
2. bundle `valdrd` and `valdr-miner` inside the application bundle;
3. sign nested executables first, then the application bundle with Developer ID;
4. enable hardened runtime where compatible;
5. package the signed application into a distributable macOS image/package;
6. notarize with Apple's notarization service;
7. staple notarization;
8. verify with `codesign` and Gatekeeper tooling;
9. perform a clean-install launch smoke on the matching architecture.

The current ad-hoc CI signature is development evidence only; it does not satisfy Stage 13 Developer ID/notarization acceptance.

## 5. Linux packaging design

Target: Linux x64.

Two mandatory artifacts:

- AppImage;
- Debian `.deb`.

Both packages must include:

- VALDR Desktop;
- managed `valdrd`;
- managed `valdr-miner`;
- desktop icon/menu metadata;
- package version and architecture metadata;
- no private keys, passwords, wallet files, or pre-created chain database.

Clean-environment tests must install/run each package independently. The `.deb` uninstall path must not silently delete user data.

## 6. Release manifest contract

The machine-readable manifest must contain at least:

- VALDR version;
- git commit SHA;
- network;
- Chain ID;
- protocol version;
- artifact filename;
- OS;
- architecture;
- artifact SHA-256;
- artifact byte size;
- signing status;
- notarization status where applicable;
- release UTC date;
- minimum supported OS.

The manifest generator must be deterministic over a fixed artifact directory. CI must fail if an artifact listed in the manifest is missing or if its size/hash differs.

## 7. Signature/key handling

The Master-TZ requires a signed release/checksum manifest but does not prescribe the signature technology.

Therefore Stage 13 implementation must:

- keep the signing backend isolated behind one release step;
- publish the corresponding verification material/instructions;
- never print private signing material to CI logs;
- never store signing keys in the repository, Docker image, application bundle, installer, or ordinary artifacts;
- support an unsigned development dry-run, but such a run must be explicitly marked non-release and must not satisfy §23.

The concrete manifest-signature mechanism should be fixed when the release signing identity/key custody model is approved.

## 8. Clean-install acceptance matrix

The release pipeline must prove:

| Platform | Required clean test |
| --- | --- |
| Windows x64 | install → launch → first-run visible → close → uninstall |
| macOS ARM64 | install/mount → Gatekeeper verification → launch → first-run visible |
| macOS Intel/AMD64 | same on Intel runner |
| Linux AppImage x64 | executable launch on clean runner/image |
| Linux .deb x64 | install with package manager → launch → uninstall |

For every platform, the release test must also verify that the bundled node/miner binaries are present and executable and that Mainnet remains unavailable.

## 9. Download-page contract

The current `valdr-core` repository contains no implemented official download-page surface.

Before implementing website integration, identify the authoritative website repository/deployment target. The download UI must consume release metadata rather than duplicating filenames/hashes by hand.

Required UX from the Master-TZ:

- current version;
- OS suggestion without hiding other platforms;
- explicit Windows/macOS/Linux choices;
- architecture and file size;
- release notes;
- SHA-256;
- signed manifest;
- source-code link;
- verification instructions;
- Testnet status;
- system requirements and disk/sync warning.

No placeholder public domain or unofficial mirror should be introduced.

## 10. Release identity prerequisite

The pre-implementation metadata contract is defined in `docs/product/stage13_release_identity_contract.md`.

A concrete release candidate must not use the current development string `0.2.0-dev`. The application version, package metadata, artifact names and release manifest must agree before any signed release is published. Master-spec revision `v0.2.5` remains separate from the application version.

## 11. Stage 13 execution order after Stage 12 freeze

1. freeze Stage 12 accepted commit;
2. create release-version contract and tag policy;
3. implement deterministic manifest/checksum generator;
4. implement Windows installer + clean install/uninstall test;
5. implement Linux AppImage + .deb + clean tests;
6. implement macOS Developer ID/notarization pipeline;
7. add signing backend for manifest/checksums;
8. add release workflow that maps every asset to the exact git commit;
9. integrate official download page;
10. perform a second-clean-environment verification;
11. close §23 evidence matrix.

Each step must leave the repository buildable/testable. No Stage 14 public/private Testnet rollout begins until Stage 13 acceptance is recorded.
