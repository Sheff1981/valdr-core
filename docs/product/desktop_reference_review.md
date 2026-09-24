# VALDR Desktop Reference Review

**Date reviewed:** 2026-09-24  
**Master specification:** `docs/VALDR_Master_TZ_v0.2.3.md`  
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
- pin the actual Wails v2 release in Go module/build metadata;
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
