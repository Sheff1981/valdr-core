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
