# VALDR v0.1 compliance audit

Baseline reviewed: `main` at `bd32a30c698056ac601a6553e74169724a56e6ac`.
Master specification: `VALDR_Master_TZ_v0.1_14_days.pdf`, dated 23 September 2026.

## Material divergences found

| Area | Existing experimental line | Master specification v0.1 | Disposition |
| --- | --- | --- | --- |
| Implementation | Bitcoin Core 31.1 / C++ overlay | Own VALDR implementation in Go | Go implementation is the active v0.1 line. |
| Ticker | `VLD` | `VDR` | Use `VDR` in v0.1. |
| Stage | `0.5.0 PUBLIC TESTNET` | 14-day devnet MVP v0.1 | Return active development to devnet v0.1. |
| PoW hash | SHA-256d | SHA-256 for MVP | Implement the master specification when PoW is reached. |
| Block target | 600 seconds | 60 seconds | Use 60 seconds for devnet v0.1. |
| Address naming | `tvld` / VLD-oriented parameters | Working prefix `VDR` | Use VDR-oriented naming in v0.1. |
| DAA | ASERTI3-2d candidate | Simplified difficulty adjustment allowed for MVP | Do not carry ASERT into v0.1 unless the master specification is revised. |
| Build path | Fetch and patch upstream Bitcoin Core | `go build ./...` from a clean clone | Go build is the readiness gate. |

## Preservation of the experiment

The previous Bitcoin Core/C++ experiment is preserved unchanged on branch:

`archive/bitcoin-core-experiment-0.5.0`

Archive point:

`bd32a30c698056ac601a6553e74169724a56e6ac`

No experimental result is treated as proof that the Go-based VALDR v0.1 stages are implemented or tested.

## Current implementation boundary

Day 1 only: project skeleton, module metadata, binary entry-point skeletons, package skeletons, documentation, and a metadata test. Blockchain, Genesis, PoW, wallet, transactions, UTXO, P2P, mining, storage implementation, and RPC behavior are not implemented yet.
