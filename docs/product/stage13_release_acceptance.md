# VALDR Stage 13 release acceptance evidence

**Status:** IN PROGRESS — development packaging + keyless provenance evidence green; public Testnet release acceptance not yet satisfied.  
**Baseline:** `docs/VALDR_Master_TZ_v0.2.8.md` §23  
**Implementation evidence commit:** `f4b24b89ea28782abb902fe2055897e5f85b8244`  
**CI evidence:** VALDR v0.2 CI run `36145374543` — SUCCESS on 2026-09-25.

This document records evidence only. It does not upgrade `0.2.0-dev` artifacts into a public Testnet release and does not authorize Stage 14.

## 1. Current evidence matrix

| Gate | Current evidence | Status |
| --- | --- | --- |
| Windows clean installer/uninstaller | NSIS development installer builds on Windows x64; silent install launches Desktop; bundled `valdrd.exe` and `valdr-miner.exe` are present; uninstall removes only VALDR-owned files and preserves user data/unrelated files | automated development gate green |
| Windows portable archive | canonical versioned ZIP contains Desktop, node and miner; bundled node version executes | automated development gate green |
| Linux AppImage | versioned AppImage builds on Ubuntu 24.04 runner, passes integrity/launch smoke, contains managed node/miner and fresh first-run does not unexpectedly start the node before wallet setup | automated development gate green |
| Linux .deb | installs with package manager, launches, contains Desktop/node/miner, uninstalls program files and preserves user data marker | automated development gate green |
| macOS ARM64 package | versioned DMG is built from native ARM64 Wails `.app`, ad-hoc signed for bundle integrity, verified, mounted read-only and launched | automated development gate green |
| macOS Intel/AMD64 package | versioned DMG is built on native Intel runner, ad-hoc signed for bundle integrity, verified, mounted read-only and launched | automated development gate green |
| SHA-256 / artifact size | canonical manifest recomputes SHA-256 and byte size from real artifacts; `SHA256SUMS` verifies | automated development gate green |
| Exact commit binding | manifest binds exact 40-character git commit plus deterministic Testnet identity | automated development gate green |
| Testnet identity | manifest binds network `testnet`, Chain ID `valdr-testnet-1`, protocol 2/2 | automated development gate green |
| Canonical cross-platform assembly | one CI bundle contains Windows installer/portable, Linux AppImage/deb, macOS ARM64/Intel DMGs, canonical manifest, `SHA256SUMS` and provenance bundle | automated development gate green |
| GitHub/Sigstore keyless provenance | `actions/attest@v4` creates signed provenance using GitHub Actions OIDC/Sigstore for the assembled release subjects | automated development gate green |
| Provenance bundle retention | assembly contains `VALDR-Desktop-<version>-provenance.sigstore.json` | automated development gate green |
| Clean provenance verification | separate clean runner verifies artifacts with `gh attestation verify` constrained to `Sheff1981/valdr-core`, expected workflow, source ref and exact commit | automated development gate green |
| Exact CI-run → release handoff | manual `valdr-v02-release.yml` accepts a successful source CI run ID + exact commit, re-verifies manifest/checksums/provenance and emits a non-public handoff record; it has no release publication permission | implemented; execution evidence pending manual dispatch |
| Production fail-closed gate | current `0.2.0-dev` cannot generate a production Testnet manifest; development signing/notarization claims remain invalid in production metadata; mandatory package set is enforced | automated safety gate green |
| Windows Authenticode | unavailable for current project owner; v0.2.8 makes it optional future hardening and requires truthful `unsigned` metadata when absent | not a Testnet release blocker |
| Apple Developer ID / notarization | unavailable for current project owner; v0.2.8 makes it optional future hardening and requires truthful ad-hoc / not-notarized metadata when absent | not a Testnet release blocker |
| Official download page | authoritative source identified as `Sheff1981/valdr-site`; development-state Download/Verify/Releases UX now consumes current provenance policy and verified core development identity while executable links remain disabled | implementation present; final RC links pending |
| Official website CI | website runs `36145756061`, `36146021905` and `36146351810` terminated before any step with `runner_id=0`; changing the runner label did not resolve allocation | infrastructure blocked / no test evidence |
| Final Testnet release-candidate version | repository still uses `config.Version = "0.2.0-dev"` | pending deliberate freeze |
| Final release clean verification | development path is proven; frozen non-development RC must repeat checksum + provenance verification | pending release candidate |
| Stage 12 manual Windows acceptance | automated acceptance is green, but outstanding Windows manual GUI acceptance is not recorded as closed | blocking public release |

## 2. Development artifacts proven by CI

Run `36145374543` proves the current Stage 13 development pipeline produces:

- Windows x64 installer and portable ZIP;
- Linux x64 AppImage and amd64 `.deb`;
- macOS ARM64 and Intel/x64 DMGs;
- canonical `release-manifest.json`;
- canonical `SHA256SUMS`;
- GitHub/Sigstore keyless provenance attestation plus retained Sigstore bundle;
- independent clean-runner checksum and provenance verification.

Current application version remains `0.2.0-dev`. Provenance proves where these development artifacts came from and that their digests match the attested build identity; it does not convert them into an accepted public Testnet release.

## 3. OS-vendor signing boundary

Windows Authenticode and Apple Developer ID/notarization are not available to the current project owner and are no longer mandatory Testnet gates under Master-TZ v0.2.8.

Windows packages must therefore state `unsigned` when Authenticode is absent. macOS packages may use the existing ad-hoc signature for bundle integrity but must state that they are not Developer ID signed and not notarized. Neither state may be described as Microsoft/Apple certification.

If legitimate OS-vendor signing becomes obtainable later, it may be added as hardening without replacing the mandatory SHA-256 + repository/workflow/commit provenance checks.

## 4. Remaining public-release blockers

The remaining mandatory blockers are:

1. close Stage 12 manual Windows GUI acceptance;
2. deliberately freeze a non-development Testnet release-candidate version instead of `0.2.0-dev`;
3. restore an executing website CI runner and validate the implemented download/verification integration; final RC artifact links remain intentionally disabled until freeze;
4. run the final RC packaging, SHA-256, GitHub/Sigstore provenance and independent clean verification gates;
5. publish only after the accepted artifacts map to the exact frozen commit.

No fake, borrowed or misleading Microsoft/Apple identity may be introduced to satisfy an obsolete checklist item.

## 5. Next implementation step

Continue Stage 13 only. Do not start Stage 14.

The software-only authenticity path and authoritative website integration are implemented. Core CI is green; website CI is currently an infrastructure-only zero-step runner failure, so the website change is not yet CI-verified. The next safe work is the manual Windows Stage 12 acceptance package/checklist plus release-handoff execution when available; no public RC freeze or Stage 14 begins before Stage 12 closes.
