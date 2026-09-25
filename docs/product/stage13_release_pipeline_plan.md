# VALDR Stage 13 release/installer design plan

**Status:** IN PROGRESS — development-only release plumbing; public release blocked.  
**Baseline:** `docs/VALDR_Master_TZ_v0.2.8.md` §§19 and 23.  
**Branch:** `valdr-v0.2`  
**Stage 12 state:** implementation/automated acceptance green; Windows manual GUI acceptance is still pending.

Master-TZ v0.2.8 authorizes CI-safe Stage 13 implementation and keyless release provenance before Stage 12 manual acceptance closes. Public release tags and accepted release publication remain blocked until Stage 12 is green. Microsoft/Apple vendor signatures are optional hardening, not release blockers.

## 1. Release invariants

Every published Desktop artifact must be traceable to one immutable git commit and one exact Testnet release version.

A release must publish, at minimum:

- Windows x64 installer;
- macOS ARM64 package;
- macOS Intel/AMD64 package, or a separately approved universal package;
- Linux x64 AppImage;
- Linux x64 .deb;
- SHA-256 checksums;
- machine-readable release manifest;
- GitHub/Sigstore keyless provenance attestation covering the release artifacts, manifest and checksums;
- release notes and source-code link.

Mainnet must remain unavailable. During Testnet, release/download UX must not contain buy/sell/exchange calls to action.

## 2. Planned repository layout

The implementation pass should prefer the following minimal additions:

- `.github/workflows/valdr-v02-release.yml` — tag/manual release pipeline;
- `packaging/linux/valdr-desktop.desktop` — Linux desktop entry;
- `packaging/linux/valdr-desktop.svg` or generated PNG copy — package icon;
- `packaging/linux/debian/` — minimal Debian package metadata only when required by the chosen packaging command;
- `cmd/valdr-release-manifest/` — deterministic manifest/checksum generation;
- `scripts/release-verify.sh` / PowerShell equivalent only if needed for clean-environment verification;
- `docs/product/stage13_release_acceptance.md` — evidence matrix for §23.

Do not create duplicate build systems. Existing Wails Desktop build configuration remains authoritative for the application binary.

## 3. Windows packaging design

Target: Windows x64.

Planned flow:

1. build the Wails Desktop application;
2. build/bundle `valdrd.exe` and `valdr-miner.exe`;
3. create the Windows installer with the Wails/NSIS packaging path supported by the pinned Wails version;
4. record `unsigned` truthfully when Authenticode is unavailable; if a legitimate Authenticode identity becomes available later, signing is additive hardening;
5. generate/verify repository-bound provenance for the final artifact;
6. install on a clean Windows runner/profile;
7. launch Desktop without a terminal;
8. verify uninstall removes only VALDR-owned program files while preserving user blockchain/wallet data and unrelated files.

No Windows signing credential is required for Testnet acceptance under v0.2.8. If one is introduced later, it must be injected only through protected CI/external signing and never committed.

## 4. macOS packaging design

Targets: ARM64 and Intel/AMD64 unless a universal artifact is separately approved.

Planned flow:

1. build the Wails `.app`;
2. bundle `valdrd` and `valdr-miner` inside the application bundle;
3. apply the current ad-hoc bundle signature needed for development bundle integrity, while labelling it truthfully as not Developer ID;
4. package the application into a distributable macOS image/package;
5. generate/verify repository-bound provenance for the final artifact;
6. perform a clean mount/launch smoke on the matching architecture;
7. document the normal per-application macOS security override path required for an unsigned/unnotarized Testnet build without instructing users to disable Gatekeeper globally.

Developer ID signing/notarization remains optional future hardening if legitimately obtainable and is not a Stage 13 blocker under v0.2.8.

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
- truthful OS-vendor signing status;
- truthful notarization status where applicable;
- provenance method/requirement;
- release UTC date;
- minimum supported OS.

The manifest generator must be deterministic over a fixed artifact directory. CI must fail if an artifact listed in the manifest is missing or if its size/hash differs.

## 7. Provenance/attestation handling

Master-TZ v0.2.8 fixes the mandatory Testnet release-authenticity mechanism as GitHub Artifact Attestations backed by Sigstore keyless OIDC signing for the public repository.

Stage 13 implementation must:

- attest the canonical package artifacts plus release manifest/checksums after they are assembled;
- bind verification to the expected VALDR repository/workflow and exact source commit for a frozen release;
- retain the generated Sigstore bundle with the release assembly when practical;
- verify provenance in a separate clean job;
- publish verification instructions;
- keep development builds explicitly development-only even when attested.

The keyless path uses short-lived workflow identity and does not require a long-lived private signing key in repository secrets. OS-vendor code signing remains optional future hardening.

## 8. Clean-install acceptance matrix

The release pipeline must prove:

| Platform | Required clean test |
| --- | --- |
| Windows x64 | install → launch → first-run visible → close → uninstall |
| macOS ARM64 | mount/integrity check → launch smoke → provenance verification; unsigned/unnotarized state disclosed |
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
- SHA-256 plus GitHub/Sigstore provenance verification;
- source-code link;
- verification instructions;
- Testnet status;
- system requirements and disk/sync warning.

No placeholder public domain or unofficial mirror should be introduced.

## 10. Release identity prerequisite

The pre-implementation metadata contract is defined in `docs/product/stage13_release_identity_contract.md`.

A concrete release candidate must not use the current development string `0.2.0-dev`. The application version, package metadata, artifact names and release manifest must agree before any public Testnet release is published. Master-spec revision `v0.2.8` remains separate from the application version.

## 11. Stage 13 execution order

1. keep the outstanding Stage 12 manual acceptance as a blocking public-release gate;
2. release-version contract and tag policy — design frozen;
3. deterministic manifest/checksum generator — implementation introduced, CI acceptance required;
4. Windows installer + clean install/uninstall test;
5. Linux AppImage + .deb + clean tests;
6. GitHub/Sigstore keyless provenance attestation for the canonical release assembly;
7. clean provenance verification bound to repository/workflow/commit;
8. release workflow mapping every asset to the exact git commit — development handoff workflow implemented; public publication path intentionally disabled while Stage 12 is open;
9. official download-page integration after its authoritative repository/deployment target is identified;
10. second-clean-environment verification of the frozen public release candidate;
11. close §23 evidence matrix.

Each step must leave the repository buildable/testable. No public Testnet release and no Stage 14 rollout begins until the required Stage 12 and Stage 13 gates are both closed.


## 12. Development release handoff workflow

`.github/workflows/valdr-v02-release.yml` is a manual, non-public handoff verifier for Stage 13 development evidence.

It accepts only:

- the numeric ID of a completed successful `VALDR v0.2 CI` run on `valdr-v0.2`;
- the exact 40-character commit expected from that run.

It then downloads only the canonical Stage 13 assembly from that exact run, rechecks SHA-256 and manifest identity, verifies GitHub/Sigstore provenance against the expected repository/workflow/ref/commit, and emits a small `release-handoff.json` with `publication_allowed=false` and `stage14_allowed=false`.

The workflow has no release/tag/write permission and cannot publish a GitHub Release. This preserves the v0.2.8 rule that Stage 12 manual Windows acceptance remains a public-release blocker while proving the exact source-run → commit → artifacts handoff boundary.
