# RandomX native adapter candidate

Status: **candidate / Mainnet disabled**

This package is the native node-side RandomX adapter used by the Mainnet consensus candidate.

Design:
- official upstream RandomX v1.2.3;
- cgo boundary is isolated in this package;
- validator uses RandomX cache/light mode rather than the multi-GiB mining dataset;
- one VM is protected by a mutex;
- cache is reinitialized only when the RandomX seed key changes;
- no network profile calls this package unless a future approved Mainnet integration explicitly wires it in.

Build requirements:
- `CGO_ENABLED=1`;
- build tag `randomx_native`;
- RandomX headers under `third_party/randomx/include`;
- platform library under `third_party/randomx/lib`.

Normal Testnet2/default builds use the non-native stub and therefore do not acquire a hidden RandomX build dependency.

The CI workflow builds official RandomX v1.2.3 from source on each supported OS, stages only the header/library into the expected build location, runs upstream RandomX tests, then runs native adapter and consensus golden-vector tests.

This is deliberately not yet an enabled Mainnet network profile.
