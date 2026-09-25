# VALDR release verification

**Baseline:** `docs/VALDR_Master_TZ_v0.2.8.md`  
**Repository:** `Sheff1981/valdr-core`

VALDR Testnet release authenticity is verified with two independent checks:

1. SHA-256 from the canonical `SHA256SUMS` / `release-manifest.json`;
2. GitHub Artifact Attestation backed by Sigstore keyless signing, tied to the VALDR repository, signer workflow and source commit.

Windows Authenticode and Apple Developer ID/notarization are not mandatory Testnet identities. When absent, the release metadata must say so explicitly. A GitHub/Sigstore attestation proves artifact origin and integrity; it is not a Microsoft/Apple endorsement and is not a security audit.

## Online provenance verification

Install a current GitHub CLI, then verify the downloaded file:

```bash
gh attestation verify PATH_TO_DOWNLOADED_FILE \
  --repo Sheff1981/valdr-core \
  --signer-workflow Sheff1981/valdr-core/.github/workflows/valdr-v02-ci.yml \
  --source-digest EXACT_RELEASE_COMMIT
```

For a frozen release, replace `EXACT_RELEASE_COMMIT` with the 40-character commit published in `release-manifest.json`. A successful check must identify the expected repository/workflow and source digest.

## SHA-256 verification

Linux:

```bash
sha256sum -c SHA256SUMS
```

macOS:

```bash
shasum -a 256 PATH_TO_FILE
```

Windows PowerShell:

```powershell
Get-FileHash .\PATH_TO_FILE -Algorithm SHA256
```

Compare the result with the artifact entry in `release-manifest.json` / `SHA256SUMS`.

## Offline/repeatable provenance

The CI release assembly stores the generated Sigstore bundle as:

`VALDR-Desktop-<version>-provenance.sigstore.json`

GitHub CLI can verify against a local bundle. For a fully offline environment, also export the current trusted root with `gh attestation trusted-root` while online and follow GitHub's offline attestation verification flow.

## OS security warnings

Unsigned Windows/macOS Testnet builds can trigger SmartScreen/Smart App Control or Gatekeeper warnings. Do not disable operating-system security globally. Verify SHA-256 and provenance first, then use only the normal per-application override allowed by the operating system if you decide to run the verified Testnet build.

## Failure rule

If SHA-256 does not match, provenance verification fails, the repository/workflow identity is unexpected, or the source commit differs from the published manifest: **do not run the artifact**.
