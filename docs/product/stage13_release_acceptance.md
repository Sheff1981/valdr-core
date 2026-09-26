# VALDR Stage 13 release acceptance evidence

**Status:** DEVELOPMENT EVIDENCE COMPLETE — Testnet2 development packaging/provenance and clean verification are CI-verified; this is not a public release acceptance.  
**Baseline:** `docs/VALDR_Master_TZ_v0.2.13.md`  
**Current evidence commit:** `71934ef8e36d03a74d01fdc81314678d381c3090`  
**CI evidence:** VALDR v0.2 CI run `36265468602` (#463) — SUCCESS on 2026-09-26.

This document records evidence only. CI #457 proves the active Testnet2 development pipeline on one exact source commit, including the completed Stage 12B cross-platform matrix, packaging, clean verification and non-public provenance handoff. It does not satisfy the remaining manual/public-release gates and does not authorize a Mainnet launch.

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
| Testnet identity | manifest binds active network `testnet2`, Chain ID `valdr-testnet-2`, protocol 2/2 | automated development gate green |
| Canonical cross-platform assembly | one CI bundle contains Windows installer/portable, Linux AppImage/deb, macOS ARM64/Intel DMGs, canonical manifest, `SHA256SUMS` and provenance bundle | automated development gate green |
| GitHub/Sigstore keyless provenance | `actions/attest@v4` creates signed provenance using GitHub Actions OIDC/Sigstore for the assembled release subjects | automated development gate green |
| Provenance bundle retention | assembly contains `VALDR-Desktop-<version>-provenance.sigstore.json` | automated development gate green |
| Clean provenance verification | separate clean runner verifies artifacts with `gh attestation verify` constrained to `Sheff1981/valdr-core`, expected workflow, source ref and exact commit | automated development gate green |
| Exact CI-run → release handoff | `valdr-v02-release.yml` is called as a same-commit reusable workflow after clean release verification; CI #457 completed `stage13-release-handoff-development / verify-development-release-handoff` successfully on exact commit `61f6941cee00a19cd187929de2879cf54e43cb6e`; the handoff re-verifies manifest/checksums/provenance and emits a non-public record only | automated development gate green |
| Production fail-closed gate | current `0.2.0-dev` cannot generate a production Testnet manifest; development signing/notarization claims remain invalid in production metadata; mandatory package set is enforced | automated safety gate green |
| Windows Authenticode | unavailable for current project owner; v0.2.8 makes it optional future hardening and requires truthful `unsigned` metadata when absent | not a Testnet release blocker |
| Apple Developer ID / notarization | unavailable for current project owner; v0.2.8 makes it optional future hardening and requires truthful ad-hoc / not-notarized metadata when absent | not a Testnet release blocker |
| Official download page | authoritative source `Sheff1981/valdr-site`; exact Stage 13B integration commit `c72ea74f38d11be27e73d147c42d58cf94ea0c0f` binds Testnet2 / `valdr-testnet-2`, current Core CI evidence, OS suggestion without hiding alternatives, SHA-256/provenance instructions, disk/network warning and fail-closed download behavior | Stage 13B implementation + CI verified; final RC links intentionally pending |
| Official website CI | `Sheff1981/valdr-site` run `36259870463` (#52), attempt 3, completed **SUCCESS** on exact site commit `c72ea74f38d11be27e73d147c42d58cf94ea0c0f`; PHP/static validation, JavaScript syntax, HTTP route/asset/language/header smoke, fake-download guard and private-specification guard all executed and passed | Stage 13B CI gate green |
| Final Testnet release-candidate version | repository still uses `config.Version = "0.2.0-dev"` | pending deliberate freeze |
| Final release clean verification | development path is proven; frozen non-development RC must repeat checksum + provenance verification | pending release candidate |
| Stage 12B automated usability gate | final Stage 12B commit `61f6941cee00a19cd187929de2879cf54e43cb6e`; full cross-platform CI #457 green | automated gate green |
| Stage 12 manual Windows acceptance | Stage 12C remains deferred final validation; no manual GUI PASS is recorded yet | blocking public release |

## 2. Development artifacts proven by CI

Run `36217224372` (#367) proves the current Testnet2 Stage 13 development pipeline and final non-public handoff execute successfully on exact commit `61f6941cee00a19cd187929de2879cf54e43cb6e`. The packaging/provenance artifacts remain development artifacts:

- Windows x64 installer and portable ZIP;
- Linux x64 AppImage and amd64 `.deb`;
- macOS ARM64 and Intel/x64 DMGs;
- canonical `release-manifest.json`;
- canonical `SHA256SUMS`;
- GitHub/Sigstore keyless provenance attestation plus retained Sigstore bundle;
- independent clean-runner checksum and provenance verification.

Current application version remains `0.2.0-dev`. Provenance proves where these development artifacts came from and that their digests match the attested build identity; it does not convert them into an accepted public Testnet release.

## 3. OS-vendor signing boundary

Windows Authenticode and Apple Developer ID/notarization are not available to the current project owner and are no longer mandatory Testnet gates under Master-TZ v0.2.11.

Windows packages must therefore state `unsigned` when Authenticode is absent. macOS packages may use the existing ad-hoc signature for bundle integrity but must state that they are not Developer ID signed and not notarized. Neither state may be described as Microsoft/Apple certification.

If legitimate OS-vendor signing becomes obtainable later, it may be added as hardening without replacing the mandatory SHA-256 + repository/workflow/commit provenance checks.

## Stage 13B website integration evidence

- Site repository: `Sheff1981/valdr-site`.
- Exact site commit: `c72ea74f38d11be27e73d147c42d58cf94ea0c0f`.
- Website CI run: `36259870463` (#52), attempt 3 — **SUCCESS** on 2026-09-26.
- The CI job executed real validation steps; it was not a zero-step runner failure.
- Active release surfaces identify Testnet2 / `valdr-testnet-2`.
- Static validation rejects stale Testnet1 identity on active download/verify/release surfaces.
- Fake executable links remain blocked while `current_release=null` and `public_release_ready=false`.
- Download/verification pages expose the release-verification method and remain fail-closed until verified public RC metadata exists.

**Stage 13B exit gate: PASS.**

## Stage 13C website content evidence

- Site repository: `Sheff1981/valdr-site`.
- Exact site commit: `805acb14df2680c13fc3488c3fc9ced1db9b9c95`.
- Website CI run: `36266129459` (#59) — **SUCCESS**.
- Required public routes are present, including Getting Started and Using VALDR.
- English canonical content and Russian maintained localization are covered by static/HTTP CI checks.
- Active content is pinned to Testnet2 / `valdr-testnet-2`; stale Devnet2/Testnet1 wording is rejected by the site validation contract.
- Existing original diagrams cover wallet/node/P2P/transaction/mining/Explorer behavior.
- The Home page now includes a real VALDR Desktop first-run screenshot captured from Core CI #463 on exact Core commit `71934ef8e36d03a74d01fdc81314678d381c3090`, rather than a fabricated product screenshot.
- French remains deferred until canonical English/Russian content is stable, as required.

**Stage 13C implementation/CI exit gate: PASS.** This does not satisfy Stage 12C manual Windows acceptance and does not freeze a public release candidate.

## 4. Remaining public-release blockers

The remaining mandatory blockers are:

1. close Stage 12 manual Windows GUI acceptance;
2. deliberately freeze a non-development Testnet release-candidate version instead of `0.2.0-dev`;
3. Stage 13B and Stage 13C website gates are green; final RC artifact links remain intentionally disabled until freeze;
4. run the final RC packaging, SHA-256, GitHub/Sigstore provenance and independent clean verification gates;
5. publish only after the accepted artifacts map to the exact frozen commit.

No fake, borrowed or misleading Microsoft/Apple identity may be introduced to satisfy an obsolete checklist item.

## 5. Current gate result and next step

The Stage 13A Testnet2 development package matrix, clean install/launch/uninstall checks, manifest/checksum verification, GitHub/Sigstore provenance verification and exact-run non-public handoff are all CI-green on run `36257506295` (#457), exact commit `61f6941cee00a19cd187929de2879cf54e43cb6e`.

Therefore the **Stage 13A development evidence gate is PASS**. This does not create a public Testnet release: `config.Version` remains `0.2.0-dev`, Stage 12C manual Windows acceptance remains deferred final validation, and the website/public RC gates remain unresolved.

Stage 13B-C website technical/content gates are CI-green. The next automatable hardening work follows Master-TZ v0.2.13; Stage 12C manual Windows acceptance remains a genuine external gate and no Mainnet or public-release claim is authorized by this document.
