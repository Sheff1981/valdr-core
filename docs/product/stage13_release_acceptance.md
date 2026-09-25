# VALDR Stage 13 release acceptance evidence

**Status:** IN PROGRESS — development packaging evidence green; production release acceptance not yet satisfied.  
**Baseline:** `docs/VALDR_Master_TZ_v0.2.7.md` §23  
**Implementation evidence commit:** `3a2e07cbca5b8b9391fae2ef58a96a87c22fa10f`  
**CI evidence:** VALDR v0.2 CI run `36133995158` — SUCCESS on 2026-09-25.

This document records evidence only. It does not upgrade development artifacts into a production Testnet release and does not authorize Stage 14.

## 1. Current evidence matrix

| Gate | Current evidence | Status |
| --- | --- | --- |
| Windows clean installer/uninstaller | NSIS development installer builds on Windows x64; silent install launches Desktop; bundled `valdrd.exe` and `valdr-miner.exe` are present; uninstall removes program files and preserves user data marker | automated development gate green |
| Windows portable archive | canonical versioned ZIP contains Desktop, node and miner; bundled node version executes | automated development gate green |
| Linux AppImage | versioned AppImage builds on Ubuntu 24.04 runner, passes integrity/launch smoke, contains managed node/miner and fresh first-run does not unexpectedly start the node before wallet setup | automated development gate green |
| Linux .deb | installs with package manager, launches, contains Desktop/node/miner, uninstalls program files and preserves user data marker | automated development gate green |
| macOS ARM64 package | versioned DMG is built from the native ARM64 Wails `.app`, ad-hoc signed for development, verified, mounted read-only and launched from the mounted image | automated development gate green |
| macOS Intel/AMD64 package | versioned DMG is built on native Intel runner, ad-hoc signed for development, verified, mounted read-only and launched from the mounted image | automated development gate green |
| SHA-256 / artifact size | `cmd/valdr-release-manifest` recomputes SHA-256 and byte size from real artifacts; CI verifies manifest values against files | automated development gate green |
| Exact commit binding | development manifests bind the exact 40-character `GITHUB_SHA` and deterministic Testnet identity | automated development gate green |
| Testnet identity | manifest binds network `testnet`, Chain ID `valdr-testnet-1`, protocol 2/2 | automated development gate green |
| Production Windows signing | Authenticode identity/certificate not configured | blocked by external signing identity |
| Production macOS signing | Developer ID identity not configured | blocked by external Apple identity |
| macOS notarization | notarization credentials/service flow not configured | blocked by external Apple identity |
| Signed release/checksum manifest | signing technology/key custody identity has not been approved | blocked by release-signing decision |
| Official download page | authoritative site is separate from this repository; accepted release assets do not exist yet | pending |
| Second independent clean-environment verification | development CI uses clean hosted runners, but production signed artifacts have not yet been verified by a second independent environment | pending production candidate |
| Stage 12 manual acceptance | automated acceptance is green, but outstanding Windows manual GUI acceptance is not recorded as closed | blocking public release |

## 2. Development artifacts proven by CI

Current Stage 13 development packaging produces versioned artifacts using the application version reported by the bundled node:

- Windows x64 installer and portable ZIP;
- Linux x64 AppImage;
- Linux amd64 `.deb`;
- macOS ARM64 DMG;
- macOS Intel/x64 DMG;
- per-platform development `release-manifest.json`.

Current application version remains `0.2.0-dev`. These artifacts are intentionally development-only and must not be published as an accepted public Testnet release.

## 3. macOS development package boundary

The macOS CI currently uses ad-hoc signing only so that bundle integrity can be exercised during development.

This is not Developer ID signing and is not notarization. Stage 13 §23 remains open until final candidate artifacts are signed with the real Developer ID identity, notarized by Apple and verified on the required target architectures.

## 4. Release blockers that must not be faked

Production acceptance still requires real external identities/material:

1. Windows Authenticode code-signing identity/certificate custody;
2. Apple Developer ID and notarization credentials;
3. approved release-manifest signing identity/key custody and public verification material;
4. Stage 12 manual Windows GUI acceptance closure;
5. final release-candidate version replacing `0.2.0-dev`;
6. official download-page integration against the accepted release metadata;
7. second-clean-environment verification of the final signed artifacts.

No test/fake credentials may be substituted for these production gates.

## 5. Next implementation step

Continue Stage 13 only. Do not start Stage 14.

The next code pass should prepare the release workflow around the already-proven package builders while failing closed when required production signing identities are absent. It must not create a public release tag or claim signing/notarization until Stage 12 manual acceptance and the external signing prerequisites are explicitly satisfied.
