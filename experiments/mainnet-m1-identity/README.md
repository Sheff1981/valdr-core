# VALDR Mainnet M1 identity candidate

This directory contains **non-activating engineering scaffolding** for M1.

It does not add a production Mainnet network profile and cannot enable Mainnet.

Candidate values currently exercised here:

- Chain ID: `valdr-mainnet-1`
- P2P magic: first four SHA-256 bytes of the Chain ID = `12795b10`
- P2P port: `27333`
- RPC port: `27332`
- protocol version: `2`
- data-directory network component: `mainnet`

Deliberately unresolved:

- Mainnet address prefix;
- fixed/DNS bootstrap contacts;
- final bootstrap/discovery policy.

Those items stay open until the active Testnet2 real-network P2P behavior is validated and can be carried into Mainnet without speculative rework.

Safety properties checked by tests:

- no Chain ID collision with existing VALDR profiles;
- no P2P magic collision with existing VALDR profiles;
- no P2P/RPC port collision with existing VALDR profiles;
- Mainnet profile remains rejected by `config.ResolveNetworkProfile("mainnet")`;
- candidate data-directory identity is separated from existing network profile names.

Status: engineering candidate only; not frozen consensus/network identity.
