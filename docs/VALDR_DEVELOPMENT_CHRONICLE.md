# VALDR Development Chronicle

This file is the durable factual project chronicle for later public documentation, including website/community/VK materials. It records verified engineering milestones and major design decisions without marketing inflation.

## Project identity

VALDR (VDR) is a native cryptocurrency with its own blockchain, not a token on Ethereum, BNB Chain, Solana or another external chain.

Current active development network: Testnet2.
Current Chain ID: `valdr-testnet-2`.
Address prefix: `VDR1`.
Consensus family: Proof of Work + UTXO.
Target block interval: 60 seconds.

## How the project evolved

### Native chain baseline

VALDR was built as a standalone blockchain with its own node, wallet, miner, transaction model, UTXO state, persistent blockchain storage, RPC/CLI and Desktop application.

The project deliberately follows an implementation-first engineering sequence:

specification -> implementation -> build -> automated tests -> real distributed validation -> evidence -> release decision.

### Testnet2 hardening

Before Mainnet, development was kept on Testnet2 specifically to expose real operational faults safely.

This proved important. Local wallet-to-wallet transfers, mempool admission, mining and confirmation already worked on one Desktop installation, but real multi-machine testing exposed that independent machines could remain at `Peers 0` and build divergent local chain histories.

The project therefore stopped treating a successful local wallet/miner demonstration as proof of a working cryptocurrency network.

### Bitcoin-class network model adopted as a project law

On 30 September 2026 the project formalized the network invariant:

download official client -> launch -> automatic network discovery -> connect to live peers -> verify/synchronize the chain -> wallet/mining/relay available.

Ordinary users must not enter peer IP addresses, ports, seeds, node IDs or RPC endpoints.

There is no fixed protocol topology such as "3 nodes", "4 nodes" or "10 nodes". Bootstrap nodes are discovery helpers only. After first contact, peers learn other peers, persist them and reconnect automatically.

If every node is offline, the live P2P network is temporarily unavailable, but blockchain state remains on participant disks. When compatible nodes return, they resume the same network identity and reconcile by cumulative chainwork.

Mining is not required for P2P connectivity or transaction relay. Mining is required to create new confirmed blocks.

### P2P defects found during real Windows testing

Real Windows Testnet2 testing found several issues that automated local tests had not fully exposed:

- machines could show `Peers 0`;
- the initial bootstrap route was a single operational dependency;
- public-node configuration was too visible to ordinary users;
- the Windows listener needed explicit IPv4 behavior for IPv4 public endpoints;
- a public bootstrap node could attempt to dial its own advertised seed address.

The corresponding engineering direction is:

- explicit IPv4 P2P listener behavior where an IPv4 endpoint is configured;
- self-bootstrap avoidance;
- automatic bootstrap, peer exchange, peer cache and reconnect;
- redundant real bootstrap/discovery routes before broad public testing;
- no manual networking controls in the ordinary Desktop user flow.

### Desktop UX rule

The ordinary VALDR Desktop network screen is informational and diagnostic, not a network-configuration wizard.

It should show useful status such as:

- node online/offline;
- connected peers;
- synchronization progress;
- mempool state;
- chain tip/chainwork;
- automatic discovery/reconnect state;
- diagnostics and logs when needed.

Manual advertised-address/public-node configuration remains operator tooling, not normal user onboarding.

## Current release gate

A Windows client is not considered a completed Bitcoin-Core-class VALDR client until real independent machines pass this sequence:

1. install/upgrade without losing wallet or node data;
2. automatically connect with no manual peer IP;
3. converge on the same height, tip and chainwork;
4. send while mining is off;
5. receiver sees the same transaction as pending / 0 confirmations;
6. enable mining on only one machine;
7. both machines see the same transaction confirmed in the same block;
8. restart both;
9. peers reconnect and chain/history remain intact;
10. conflict/reorg/recovery behavior remains correct.

## Mainnet rule

Mainnet work may be prepared in code, but Mainnet must not be represented as launched or production-ready until Testnet distributed validation is complete and the Mainnet-specific specification is approved.

## Status language

Use these terms precisely:

- planned: specified, not implemented;
- implemented: code exists;
- CI-verified: automated gates passed;
- manually verified: human-visible acceptance passed;
- distributed-session verified: independent-machine network sessions passed.

This chronicle should be extended as significant verified milestones occur.
