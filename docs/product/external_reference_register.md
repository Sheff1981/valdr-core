# External implementation references for VALDR

This register records external systems used only for architecture/security comparison. It is not a dependency manifest.

| Project | Exact version/tag | Purpose | Code import policy |
| --- | --- | --- | --- |
| Bitcoin Core | v31.0 / commit `6574cb40869b96b9ffc79c19dc8f4e467d60f321` | node decomposition, consensus boundary, wallet/node separation, P2P, policy, RPC, GUI, testing, release engineering | reference only; no identity/consensus constants copied |
| Litecoin Core | pending exact source tag | compare mature UTXO/PoW network parameters, mining and release choices | reference only |
| Dogecoin Core | pending exact source tag | compare issuance, AuxPoW/mining boundary and P2P evolution | reference only |
| Bitcoin Cash Node | pending exact source tag | compare UTXO policy, mempool/block validation and operational tooling | reference only |

Rules:

- pin every reviewed source to an exact tag/commit;
- record source archive/package SHA-256 when a local artifact is supplied;
- distinguish source-derived observation from VALDR design decisions;
- never treat external defaults as VALDR defaults;
- no external chain identity, Genesis, address prefix, P2P magic or monetary constants enter VALDR without an explicit Master-TZ revision.
