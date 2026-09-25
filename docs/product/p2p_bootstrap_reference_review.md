# VALDR P2P Bootstrap Reference Review

**Date reviewed:** 25 September 2026  
**Scope:** v0.2.6 peer discovery / zero-budget distributed Testnet architecture

## Official references reviewed

1. Bitcoin Core chain parameters:
   - https://github.com/bitcoin/bitcoin/blob/master/src/kernel/chainparams.h
   - https://github.com/bitcoin/bitcoin/blob/master/src/kernel/chainparams.cpp
2. Bitcoin Core DNS seed operator policy:
   - https://github.com/bitcoin/bitcoin/blob/master/doc/dnsseed-policy.md
3. Litecoin Core chain parameters:
   - https://github.com/litecoin-project/litecoin/blob/master/src/chainparams.cpp

## Observations

- Bitcoin Core network profiles expose both DNS seed hostnames and fixed seed addresses.
- Bitcoin Core documents DNS seeds as bootstrap helpers and explicitly aims to minimize trust in them.
- Litecoin Core main/test networks likewise contain multiple DNS seed hostnames and fixed seed data.
- These networks are not based on one central consensus server: ordinary nodes validate the chain independently.
- However, a brand-new Internet node still needs some initial reachable contact before peer exchange can begin.

## What VALDR adopts

- persistent learned-peer cache/address book;
- peer exchange after connection;
- optional fixed seed addresses;
- optional DNS seed hostnames;
- manual peer/seed overrides;
- repeated outbound maintenance;
- outbound-only Desktop default;
- optional public full-node mode for users/operators;
- bootstrap sources never participate in consensus decisions.

## What VALDR rejects

- mandatory project-owned paid public servers;
- fabricated placeholder seed addresses;
- treating a seed as a trusted chain authority;
- requiring ordinary Desktop users to configure inbound NAT/firewall;
- copying Bitcoin/Litecoin branding, text or incompatible implementation details.

## Zero-budget operational rule

VALDR may ship Testnet with no project-owned server. Real bootstrap endpoints are added only when independently operated users/volunteers provide reachable peers or DNS seeds. Before such a contact exists, initial participants use explicitly shared peer addresses. Once a node learns peers, it persists them locally and can reconnect without relying on the same bootstrap source.

This is a product/P2P operational change, not a consensus change.
