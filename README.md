# VALDR Core

VALDR is a standalone cryptocurrency and blockchain project. The active v0.1 implementation follows `VALDR_Master_TZ_v0.1_14_days.pdf`.

## Protocol identity

- Network / coin: **VALDR**
- Ticker: **VDR**
- Smallest unit: **val**
- `1 VDR = 100,000,000 val`
- Primary implementation language: **Go**

## Development status

Current active milestone: **Day 1 - project skeleton**.

Implemented in Day 1:

- Go module and project directory structure;
- skeleton entry points for `valdrd`, `valdr-cli`, and `valdr-miner`;
- project identity constants;
- package skeletons for the modules named by the master specification;
- compliance audit against the previous experimental repository state;
- basic metadata unit test.

Not implemented yet: blockchain/Genesis, Proof of Work, wallet cryptography, transactions, UTXO engine, mining rewards, P2P synchronization, persistent blockchain storage, RPC behavior, and the final three-node devnet scenario.

## Build and test

```bash
go build ./...
go test ./...
```

## Project layout

```text
cmd/
  valdrd/
  valdr-cli/
  valdr-miner/
core/
  block/
  blockchain/
  transaction/
  utxo/
  consensus/
crypto/
wallet/
p2p/
mining/
storage/
rpc/
config/
tests/
docs/
scripts/
go.mod
```

## Archived Bitcoin Core experiment

The previous Bitcoin Core 31.1 / C++ experiment is not the active v0.1 architecture. It is preserved unchanged on branch `archive/bitcoin-core-experiment-0.5.0` at commit `bd32a30c698056ac601a6553e74169724a56e6ac` for later review. Existing legacy files still present on this branch are historical artifacts and are not evidence that the corresponding Go v0.1 stage has been implemented.

See `docs/COMPLIANCE_AUDIT.md` for the recorded divergences.
