# VALDR Desktop Reference Review

**Date reviewed:** 2026-09-24  
**Master specification:** `docs/VALDR_Master_TZ_v0.2.5.md`  
**Purpose:** mandatory product/reference review before Stage 12 implementation.

This document records product observations only. It does not authorize copying branding, artwork, copyrighted UI text, source code, wallet formats or foreign chain architecture.

## 1. Bitcoin Core

Official reference reviewed:

- https://bitcoincore.org/en/download/

Observed:

- the download page presents the current version first and then explicit platform choices;
- Windows, macOS Intel, macOS ARM and multiple Linux architectures are distinct artifacts;
- SHA-256 binary hashes and hash signatures are first-class release artifacts;
- the page explains bandwidth/storage requirements before users install a full node;
- verification instructions are platform-specific;
- reproducible-build verification is documented as an additional assurance layer.

VALDR adopts:

- explicit Windows/macOS/Linux platform choices;
- architecture labels;
- SHA-256 checksums;
- signed release/checksum manifest;
- visible disk/network synchronization warning;
- release notes/version identity;
- clean verification instructions per OS;
- source-code link and exact git-commit identity in release metadata.

VALDR rejects/defer:

- copying Bitcoin Core branding/UI;
- assuming Bitcoin storage/pruning numbers apply to VALDR;
- adopting Bitcoin consensus or wallet formats;
- requiring command-line verification for ordinary first launch. Verification instructions remain available, while OS code signing/notarization should provide the normal path.

Security/release implication:

VALDR release artifacts must be independently verifiable and traceable to one release commit.

## 2. Monero GUI

Official reference reviewed:

- https://www.getmonero.org/downloads/

Observed:

- Monero explicitly presents a GUI wallet for both beginners and advanced users;
- Simple mode emphasizes minimal user configuration;
- Advanced mode exposes more node/wallet control;
- Windows installer/zip, macOS Intel/ARM and Linux builds are presented separately;
- hash verification is prominently recommended;
- wallet/node concepts are exposed progressively rather than forcing every user into operator-level settings.

VALDR adopts:

- Simple and Advanced product modes;
- Simple mode as the default;
- automatic local-node lifecycle and outbound networking for ordinary users;
- platform-specific desktop packages;
- wallet/send/receive as primary tasks;
- Advanced node/network/mining diagnostics;
- prominent download integrity guidance.

VALDR rejects/defer:

- remote-node-by-default as the normal VALDR trust model;
- Monero-specific privacy protocol, wallet format, pruning semantics or hardware-wallet behavior;
- merchant/fiat conversion features in Stage 12.

Security/release implication:

A beginner-friendly UI must not weaken local key custody, node verification or download authenticity.

## 3. Litecoin Core

Official references reviewed:

- https://litecoin.org/
- https://download.litecoin.org/

Observed:

- Litecoin distributes native Core artifacts for desktop/server operating systems;
- current release directories include Windows setup/zip, macOS DMG/tar and Linux archives;
- signed checksum material accompanies release artifacts.

VALDR adopts:

- native installable desktop artifacts in addition to operator/server binaries;
- a portable Windows artifact in addition to the installer;
- macOS disk-image style distribution;
- Linux package/archive options;
- signed release metadata.

VALDR rejects/defer:

- copying Litecoin branding or Bitcoin-derived UI conventions when they do not fit VALDR;
- adopting Litecoin consensus, wallet or address internals.

Security/release implication:

Desktop packaging and operator binaries must resolve to the same VALDR version/commit and manifest.

## 4. Ethereum ecosystem

Official references reviewed:

- https://ethereum.org/wallets/
- https://ethereum.org/wallets/find-wallet/
- https://ethereum.org/run-a-node
- https://ethereum.org/security

Observed:

- wallet onboarding is organized around user needs and device/platform availability;
- self-custody and responsibility for keys are explained in plain language;
- irreversible-transaction and key/recovery safety education is prominent;
- running a local node is framed as improving privacy, security and reduced reliance on third parties;
- Ethereum's node model is an ecosystem of separate clients/components rather than a single VALDR-style node implementation.

VALDR adopts:

- plain-language self-custody warnings;
- clear distinction between wallet, address, node and network;
- no support workflow should ever ask for private keys/passphrases;
- transaction confirmation should emphasize irreversibility;
- local-node privacy/security benefits should be explained without overwhelming first-run UX;
- security guidance belongs inside the product/download documentation, not only developer docs.

VALDR rejects:

- Ethereum execution/consensus multi-client architecture as a template for VALDR;
- dapps, browser-wallet model, swaps, bridges, NFTs, staking and exchange integrations in Stage 12;
- cloud/custodial accounts.

Security/release implication:

VALDR Desktop remains a self-custody application controlling a local VALDR wallet and local VALDR node.

## 5. Wails v2 framework review

Official reference reviewed:

- https://v2.wails.io/docs/gettingstarted/installation/

Observed on 2026-09-24:

- Wails v2 documentation identifies v3 as beta;
- v2 supports Windows 10/11 AMD64/ARM64, macOS AMD64/ARM64 and Linux AMD64/ARM64;
- Windows uses WebView2;
- macOS requires native Xcode tooling for development/build;
- Linux requires GTK/WebKit dependencies;
- NSIS is supported as an optional Windows installer dependency.

VALDR decision:

- Stage 12 baseline is Wails v2 stable;
- implementation pin: **Wails v2.12.0**;
- v2.12.0 is used because its module declares Go 1.22 and is compatible with the current VALDR Go 1.23.x toolchain;
- Wails v2.13.0-v2.15.0 currently declare Go 1.25, so adopting them would require a deliberate VALDR toolchain upgrade rather than an incidental GUI dependency change;
- native CI runners are required for real platform packaging/smoke;
- no remote CDN frontend dependencies;
- Stage 13 owns signing/notarization/installers.

## 6. VALDR Desktop product decisions after review

Adopt now:

1. Simple mode default; Advanced mode opt-in.
2. Managed local `valdrd`; do not duplicate consensus.
3. Default outbound-only P2P for Desktop.
4. Encrypted wallet v2 and local signing.
5. Main navigation: Overview, Send, Receive, Transactions, Wallet, Network, Settings; Mining only in Advanced/Testnet.
6. Clear synchronization progress, peer count, network identity and wallet lock status.
7. Platform-native release artifacts with signed/checksummed manifest.
8. No terminal requirement for normal use.
9. No telemetry/ads/cloud account by default.
10. No Mainnet selection until a separately approved Mainnet specification/release exists.

Explicitly not implementing in Stage 12:

- mobile app;
- browser extension;
- remote-node default;
- exchange/buy/sell;
- fiat conversion;
- hardware-wallet integration;
- pruning mode;
- automatic unsigned updates;
- Mainnet.

## 7. First implementation slice

The first Stage 12 code slice must be:

1. outbound-only P2P mode with no inbound listener requirement;
2. public peer gossip must omit non-routable/non-listening Desktop endpoints;
3. desktop application backend package for platform data paths, node lifecycle and status;
4. Wails v2 application shell;
5. first-run Testnet-only UI;
6. CI tests before adding send/receive screens.

No GUI work may bypass existing wallet encryption or localhost RPC boundaries.


## 8. Project-specific Premium Desktop placement

**Date:** 2026-09-25  
**Source:** project-owner supplied VALDR layout screenshot/mockup. This is an original VALDR product direction, not an external product/brand reference.

Adopt:

- keep the primary navigation on the left;
- reserve the far-right edge of the Premium Desktop layout for a compact quick-access affordance;
- place the VALDR coin/download affordance at the lower-right edge, visually separated from wallet/node controls;
- preserve the existing dark/gold VALDR visual identity and locally bundled assets.

Release constraint:

- the download affordance may be visible during Stage 12, but it must not expose nonexistent, unsigned or unverified installers;
- it remains non-operational until Stage 13 produces signed/verifiable release artifacts and the official download target is frozen.


## 9. Owner-approved Premium Desktop visual reference

**Date:** 2026-09-25  
**Source:** project-owner approved VALDR Premium Desktop reference/mockup created for this project.

Implemented product direction:

- the coin-style ornate **V** is the canonical Premium Desktop visual mark; an inverted-A-like mark is rejected;
- the same V/coin identity is used in the sidebar, splash, hero watermark, download affordance and browser/app surface where supported;
- Dashboard uses the project-owned black/gold mountain-and-raven visual language;
- layout preserves the real VALDR wallet/node data and controls instead of replacing the application with a static mockup;
- no fiat conversion is fabricated: Stage 12 continues to show actual Testnet/VALDR data only;
- Mainnet remains disabled;
- the Downloads affordance remains non-operational until Stage 13 signed/verifiable packages exist.

This is a visual/product change only. It does not modify consensus, P2P, wallet format, RPC trust boundaries or storage.


## 10. Exact owner-supplied raster reference assets

**Date:** 2026-09-25

For the Premium Desktop dashboard, the project owner explicitly requested that the supplied reference artwork be used directly rather than re-drawn or re-generated.

Implementation rule:

- sidebar VALDR branding uses a direct crop of the owner-supplied reference;
- header/mountain/raven artwork uses a direct crop of the owner-supplied reference;
- the dashboard keeps live VALDR wallet/node data and controls over those local assets;
- no remote runtime content is introduced;
- no consensus, wallet, P2P, RPC or storage behavior changes.


## 11. Settings interaction and reference rendering fixes

**Date:** 2026-09-25

Owner QA identified two Stage 12 Desktop defects:

- the owner-reference title/header was being enlarged as a raster background and appeared visibly blurred;
- Settings exposed language/theme controls as disabled placeholders, so the controls looked broken.

Implemented:

- keep the owner-supplied reference artwork, but cover the baked raster title area and render the live VALDR title copy sharply above it;
- Language now supports English and Russian and persists locally;
- Theme now switches immediately between Premium/reference and Classic dark and persists locally;
- Start-node and Advanced switches persist immediately on change;
- Network is displayed as a locked Testnet-only value rather than a non-functional dropdown because Mainnet remains disabled by the active Master-TZ.

No consensus, P2P, wallet, RPC or storage behavior is changed.


## 12. Continuous raven dashboard composition

**Date:** 2026-09-25  
**Source:** owner-approved dashboard screenshot supplied in project chat.

Implemented visual rule:

- keep the dark Premium Desktop palette;
- use a single continuous raven/background composition spanning the top header and Overview hero, instead of rendering the same raven asset twice;
- mask baked/ghost text from the raster reference with live dark UI surfaces and render the actual Desktop labels above them;
- keep Send/Receive/Mining inside the live hero flow so all three controls remain fully visible;
- keep live wallet/node/network data and existing Stage 12 behavior.

This remains a UI-only change; consensus, wallet format, RPC, P2P and storage are unchanged.


## 13. Clean Premium Desktop visual reset

**Date:** 2026-09-25

After owner QA, the accumulated experimental CSS overrides were removed from the active Desktop build. The active implementation now uses the stable pre-reference stylesheet as the base plus one consolidated Premium reference layer.

Key points:

- one raven/reference background layer only;
- no duplicated reference-art element inside the hero;
- live VALDR title and controls remain HTML, not baked screenshot UI;
- Send/Receive/Mining stay in normal document flow and cannot be clipped by the left edge;
- Settings functionality from the prior Stage 12 fix is preserved;
- dark Premium theme remains the default; Classic dark remains optional.

No consensus, wallet format, RPC, P2P or storage behavior changed.

## 14. Final single Premium background and canonical Windows icon

**Date:** 2026-09-25

Owner QA froze the Premium Desktop visual implementation to one bundled decorative image.

Implemented rule:

- Dashboard/Overview uses one locally bundled `valdr-premium-background.jpg` as its only raster background layer;
- the live VALDR title, balance, node state, network state, cards and Send/Receive/Mining controls remain HTML/UI above the artwork;
- duplicate raven/reference layers, raster sidebar branding and old reference-background usage are removed from the active Premium layer;
- the canonical application icon is the gold VALDR **V**, rendered from `frontend/public/valdr-app-icon.svg` into `build/appicon.png`;
- the inverted-A-like icon is explicitly rejected;
- the Downloads affordance uses the same canonical V/emblem and remains disabled until the Stage 13 signed-release gate.

This is a UI/branding fix only. Consensus, PoW, UTXO, P2P, wallet format, storage and RPC security boundaries are unchanged. Mainnet remains disabled.

## 15. Advanced public-node control

**Date:** 2026-09-25

Stage 12 Network/Node now implements the Master-TZ Advanced public-node control.

Behavior:

- default Desktop mode remains outbound-only;
- public-node mode is available only while Advanced mode is enabled;
- enabling it requires an explicit advertised `host:port` that is not loopback, private, link-local or a local-only hostname;
- the managed node listens for P2P on local interfaces only after the user explicitly applies the change;
- applying the mode performs a controlled managed-node restart when the node is already running;
- VALDR does not attempt automatic router/NAT/firewall changes;
- node RPC remains bound to `127.0.0.1`;
- Mainnet remains unavailable.

This implements an existing Stage 12 requirement and does not change consensus, PoW, UTXO, wallet format, storage or RPC trust boundaries.

## 16. Bounded local storage diagnostics

**Date:** 2026-09-25

Stage 12 Network/Node now exposes read-only local storage state in Advanced mode.

The diagnostics report:

- the existing node-data path;
- whether the directory is accessible;
- bounded file count;
- bounded aggregate file size;
- whether the scan reached its safety entry limit.

The scan runs only when the Advanced Network view requests it; it is not added to the normal five-second Desktop state loop. It does not modify BadgerDB, consensus state, wallet files or RPC.

## 17. Advanced diagnostics export

**Date:** 2026-09-25

Stage 12 Settings now provides an explicit Advanced-mode diagnostics export.

The user-selected local JSON report contains only operational data needed for support/debugging:

- Testnet/network identity and current node status;
- bounded storage diagnostics;
- non-secret Desktop preferences;
- wallet counts only, never wallet file contents;
- miner counters/timing/hashrate fields without the reward address;
- the bounded in-memory node log buffer after secret-line redaction.

Private keys, wallet passphrases, encrypted wallet payloads and automatic telemetry/upload are excluded. The diagnostics file is written locally with private permissions where the platform supports POSIX modes.

## 18. Advanced local Explorer link

**Date:** 2026-09-25

Stage 12 Network/Node now exposes the optional Explorer link required by the Advanced-mode plan.

Rules:

- VALDR Desktop does not embed or silently start the Explorer service;
- the separate Stage 10 read-only Explorer remains its own process;
- Desktop probes only the fixed local endpoint `http://127.0.0.1:8080/healthz`;
- redirects and non-loopback endpoints are rejected;
- the Open Explorer action is enabled only when the local service reports `status=ok` and Chain ID `valdr-testnet-1`;
- opening the Explorer uses the user's default browser;
- no wallet secrets or privileged RPC are exposed to Explorer.

This preserves the Stage 10 separation and the Desktop localhost RPC security boundary.

## 19. First-run initialization readiness gate

**Date:** 2026-09-25

Stage 12 first-run now enforces the Master-TZ initialization gate instead of dismissing onboarding immediately after wallet creation or restore.

Behavior:

- first-run explicitly states that local disk space and outbound network access are required;
- creating or restoring the encrypted wallet is only one part of initialization;
- the wallet setup controls are hidden once a wallet exists, preventing accidental duplicate first-run creation;
- the main application is entered only when an encrypted wallet exists and the managed local Testnet node is actually reachable through its localhost RPC status check;
- if the node is stopped or failed, first-run remains visible and exposes the controlled start/restart action;
- synchronization/connection status remains visible while initialization is pending.

This does not change consensus, wallet format, Testnet parameters, Mainnet state, P2P defaults or RPC trust boundaries. Custom data-directory selection remains a separate Stage 12 task because safe post-creation migration is intentionally not implemented.

## 20. Bitcoin Core first-run data-directory reference

**Date:** 2026-09-25  
**Reference:** Bitcoin Core `src/qt/intro.cpp`, `src/qt/forms/intro.ui`, and `doc/files.md`.

Bitcoin Core's first-run flow was reviewed before implementing the VALDR data-directory step. The useful product pattern is:

- present the OS default data directory;
- allow an explicit custom directory before normal startup continues;
- create/check the selected directory before committing it;
- persist the custom choice;
- explain that blockchain storage grows and initial synchronization is resource-intensive;
- keep chain-specific data separated under the selected storage location.

VALDR adopts the same first-run principle without copying Bitcoin Core's Qt code or storage architecture. In VALDR Desktop, the selectable path is specifically the managed Testnet node-data directory. Encrypted wallet files remain in VALDR's platform application-data wallet directory, so changing the blockchain location does not move private wallet material.

The selected node directory is accepted only before the first wallet is created. Desktop validates an absolute non-root directory, verifies it is creatable/writable, persists it, safely stops the managed node if needed, reconfigures the node, and restarts it when startup policy requires. Post-wallet data migration remains deliberately unavailable until a separate migration workflow is designed.

This implements Master-TZ §16.3 step 2 and §16.5 "data directory where safe"; consensus, PoW, wallet format, Testnet identity and Mainnet state are unchanged.

## 21. First-run node startup ordering

**Date:** 2026-09-25

The first-run lifecycle was aligned with Master-TZ §16.3:

- a fresh Desktop launch no longer starts the managed node before an encrypted wallet exists;
- the user can choose the node-data directory and create/restore the encrypted wallet first;
- after successful wallet setup, the managed Testnet node starts automatically when the saved startup preference permits it;
- if automatic startup is disabled or startup fails, first-run remains visible and exposes the controlled start/restart action only after a wallet exists;
- an existing installation that already has a wallet retains normal automatic node startup on Desktop launch;
- changing the first-run node-data directory no longer starts the node merely because the default startup preference is enabled.

Native clean-launch CI now verifies that the GUI process stays alive on a fresh profile while localhost RPC remains closed before wallet setup. Cross-platform runtime E2E verifies that wallet creation then starts the managed node and that the rest of the wallet/send/receive/history/crash-recovery path still works.

This is a Stage 12 lifecycle correction to match the existing Master-TZ. It does not change consensus, P2P protocol, storage format, wallet format, Testnet identity or Mainnet state.

## 22. Mining telemetry stream fix

**Date:** 2026-09-25

The Stage 12 hashrate display defect was traced to the managed miner stdout parser.

`valdr-miner` writes each accepted-block result as indented multi-line JSON. The Desktop `MinerManager` previously scanned stdout line-by-line and attempted to decode each line as a complete `MineBlockResult`. Real miner output therefore mined blocks successfully but never populated the Desktop accepted-block, hash-count, duration or hashrate counters.

The manager now uses a streaming JSON decoder over the actual child-process stdout and preserves the optional stdout mirror through an `io.TeeReader`. The unit test now uses the same multi-line JSON shape emitted by `valdr-miner`, and the live cross-platform Desktop runtime test requires positive accepted-block, average-hashrate, last-block-hashrate and hash-count telemetry after mining.

This is a Desktop telemetry/parser correction only. Consensus PoW, target calculation, block validation and miner RPC behavior are unchanged.

## 23. Ephemeral wallet-secret presentation

**Date:** 2026-09-25

A final Stage 12 wallet-UI security pass reduced the time sensitive material can remain visible in the WebView:

- revealed private-key text is cleared when the user leaves the Wallet screen;
- revealed private-key text and an open send-confirmation dialog are cleared when the Desktop document becomes hidden;
- first-run wallet passphrase/confirmation fields are cleared after a create attempt whether it succeeds or fails;
- normal wallet create and unlock passphrase fields are likewise cleared after the backend attempt;
- a frontend source-contract test guards these secret-lifecycle hooks.

This does not change wallet encryption, key derivation, consensus, RPC or P2P behavior. It is Desktop presentation hardening within the existing Stage 12 security model and does not require a Master-TZ revision.

## 24. Stage 13 official release/download reference review

**Date:** 2026-09-25  
**Scope:** release/download UX, installer verification, platform packaging and node-onboarding language.

Official references reviewed:

- Bitcoin Core: https://bitcoincore.org/en/download/
- Monero GUI: https://www.getmonero.org/downloads/
- Litecoin: https://litecoin.org/ and official download host
- Ethereum wallet/node education: https://ethereum.org/wallets/ and https://ethereum.org/run-a-node
- Wails v2 official documentation: NSIS installer, code-signing guidance, CLI/platform support.

### Bitcoin Core observations

Useful patterns:

- lead with the current version and direct OS-specific downloads;
- keep alternative platforms visible;
- publish SHA-256 checksums and signatures adjacent to downloads;
- provide platform-specific verification instructions;
- explicitly explain storage/bandwidth cost before users install a full node;
- link source and version history separately from the primary download.

VALDR adopts the presentation and verification principles, not Bitcoin branding, build system or consensus architecture.

### Monero GUI observations

Useful patterns:

- clearly distinguish a beginner-oriented mode from Advanced controls;
- present Windows installer/portable, Linux and separate macOS Intel/ARM artifacts explicitly;
- show release notes beside the current version;
- make download-hash verification prominent and explain why it matters;
- keep GUI wallet positioning understandable to non-technical users.

VALDR already uses Simple/Advanced behavior inside one Desktop application and keeps its own managed local-node architecture rather than Monero's remote-node/simple-mode model.

### Litecoin observations

Useful patterns:

- distinguish a full-node "Core/Advanced" product from simpler wallet choices;
- show the recommended platform artifact first while retaining other platforms;
- keep signed release artifacts and source code independently discoverable.

VALDR adopts the concise platform-selection concept but must not expose buy/exchange CTAs during Testnet.

### Ethereum observations

Useful education patterns:

- explain that a wallet controls keys while a node independently verifies blockchain state;
- explicitly distinguish public/shared-node trust from operating one's own node;
- explain node storage, uptime and maintenance in user terms;
- keep transaction irreversibility and key custody responsibilities visible.

VALDR adopts the educational separation between wallet/key custody and node verification. It does not adopt Ethereum's multi-client, account, smart-contract, staking or remote-provider architecture.

### Wails v2 observations

The current VALDR Desktop build is pinned to Wails v2.12.0. Official Wails v2 documentation confirms:

- Windows NSIS installer generation is supported with `wails build -nsis`;
- installer metadata comes from the Wails application Info configuration;
- Windows builds depend on WebView2 and Wails exposes explicit runtime strategies;
- Wails v2 supports Windows 10/11 AMD64, macOS AMD64 release targets from 10.13+, macOS ARM64 from 11.0+, and Linux AMD64;
- signing/notarization remains a platform-native release concern and must use protected credentials.

Stage 13 should not upgrade Wails merely to implement packaging. Keep v2.12.0 pinned unless a concrete packaging blocker is demonstrated and separately reviewed.

### Release implications for VALDR

Adopt:

- prominent Testnet release/version identity;
- OS/architecture-specific downloads;
- file size, SHA-256 and signed manifest next to each artifact;
- platform verification instructions;
- explicit disk/synchronization warning;
- source link;
- clean distinction between wallet security and local full-node operation;
- Windows/macOS/Linux platform metadata generated from release artifacts rather than copied manually.

Reject:

- exchange/buy/sell CTAs during Testnet;
- unsigned silent updates;
- unofficial primary mirrors;
- remote wallet/node dependencies that bypass the managed local VALDR node;
- copying third-party branding, layouts or text.

No consensus, P2P, wallet-format or storage changes follow from this review.



## 25. v0.2.8 release provenance review

**Date:** 2026-09-25  
**Scope:** replace unavailable OS-vendor signing gates with independently verifiable Testnet release provenance.

Official references re-reviewed:

- Bitcoin Core: https://bitcoincore.org/en/download/
- Monero GUI: https://www.getmonero.org/downloads/
- Litecoin: https://litecoin.org/ and https://download.litecoin.org/
- Ethereum: https://ethereum.org/wallets/ and https://ethereum.org/run-a-node
- GitHub Artifact Attestations: https://docs.github.com/en/actions/concepts/security/artifact-attestations
- GitHub attestation usage/verification: https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations
- Sigstore keyless signing model: https://docs.sigstore.dev/cosign/signing/overview/

### Observations

Bitcoin Core and Monero both make cryptographic download verification a first-class release concern by publishing SHA-256 data and signatures for the checksum material. Litecoin's official download host likewise publishes SHA256SUMS/SHA256SUMS.asc alongside platform binaries. Ethereum's node guidance reinforces the product principle that users should verify rather than blindly trust third-party infrastructure.

GitHub Artifact Attestations provide signed build-provenance claims for binaries. For a public repository, GitHub uses the Sigstore Public Good instance; the provenance records repository/workflow/commit context and is written to a public transparency log. Verification can be performed with `gh attestation verify` and can constrain the expected repository, signer workflow and source commit. GitHub explicitly states that an attestation is provenance/integrity evidence, not a guarantee that the artifact is secure.

Sigstore's keyless model uses short-lived certificates bound to OIDC workflow identity and transparency logging. This avoids placing a long-lived private release key in the repository or CI secret store.

### VALDR decision

Adopt for Testnet:

- canonical SHA-256 checksums and release manifest;
- GitHub/Sigstore keyless provenance attestation for release artifacts;
- verification policy pinned to `Sheff1981/valdr-core`, the VALDR release workflow and the frozen source commit;
- downloadable attestation bundle where practical for offline/repeatable verification;
- explicit disclosure that Windows/macOS artifacts may be unsigned by the OS vendor.

Keep optional only:

- Windows Authenticode;
- Apple Developer ID signing/notarization.

Reject:

- fake/self-asserted claims of Microsoft or Apple certification;
- borrowed third-party code-signing identities;
- disabling SmartScreen, Smart App Control or Gatekeeper globally merely to run VALDR;
- describing GitHub/Sigstore provenance as a security audit or platform endorsement.

This changes release-distribution security policy only. It does not modify VALDR consensus, P2P, transaction, wallet-encryption, storage or Mainnet behavior.


## 26. Stage 13 download-page platform suggestion pass

**Date:** 2026-09-25  
**Scope:** implement the remaining Master-TZ §19.1 platform-selection and full-node warning behavior in the authoritative website repository.

The existing official-reference review already established two relevant patterns: OS-specific downloads should suggest the user's platform without hiding alternatives, and a full-node download should explain storage/bandwidth cost before installation.

Implementation in `Sheff1981/valdr-site` commit `571a3fcaa23e3bd1b7a84cc927c0054a1a44a745` therefore:

- detects Windows/macOS/Linux client platform locally in the browser;
- highlights only the matching platform card as "recommended" while keeping all three platform cards visible;
- does not redirect or auto-download based on OS detection;
- adds an explicit local-node synchronization/disk-growth warning and reminds the user that the node-data directory can be chosen during first run;
- keeps executable download buttons fail-closed until verified public release metadata exists.

This adopts the already-reviewed Bitcoin Core/Monero/Litecoin download-selection and full-node education patterns without copying branding, layout or third-party code. No consensus, wallet, node, P2P, storage or Mainnet behavior changes.

## 27. Bitcoin Core v31.1 full-screen UX audit

**Date:** 2026-09-26  
**Master decision:** `docs/VALDR_Master_TZ_v0.2.10.md`  
**Reference:** Bitcoin Core v31.1 source, especially `src/qt`, `src/wallet`, `src/net*`, `doc/files.md` and release/download documentation.

Reviewed screen groups:

- first-run/data directory and synchronization;
- wallet creation/encryption/passphrase flows;
- Overview balances and recent transactions;
- Send, fee controls and Coin Control;
- Receive/payment-request UI;
- transaction filters/details/export;
- address book;
- Sign/Verify Message;
- PSBT operations;
- Node Information, RPC Console, Network Traffic and Peers;
- Options, privacy/mask-values and general Desktop behavior.

Adopt into VALDR v0.2.10:

- richer truthful sync detail;
- semantically correct balance states where supported;
- transaction search/filter/date range and CSV export;
- local destination address book;
- privacy amount masking;
- richer Advanced node/peer diagnostics;
- exact version/build/source identity.

Explicitly defer:

- RBF/abandon transaction;
- Coin Control/custom change;
- multiple-recipient send;
- payment URI;
- Sign/Verify Message;
- PSBT;
- hardware/external signer;
- watch-only/blank wallets;
- pruning;
- Tor/I2P/CJDNS;
- embedded RPC console;
- peer bans;
- automatic port mapping;
- system tray;
- network traffic graph.

Reasoning:

VALDR adopts mature product patterns that improve safety and diagnosability without expanding consensus or replacing the existing Go/Wails architecture. Bitcoin-specific features are deferred until VALDR has a concrete protocol/product need.

Security implication:

all new fields must come from real runtime/read models; no UI may fabricate chain state, balances, peer metrics, release identity or synchronization progress.

