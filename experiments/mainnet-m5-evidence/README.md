# VALDR Mainnet M5 evidence validator

Status: engineering evidence tooling only. Mainnet remains disabled.

This validator enforces the current v0.3.7 M5 evidence shape without inventing benchmark results.

Required CPU classes:
- `low_power_laptop`
- `mainstream_desktop`
- `higher_performance_desktop`

Allowed evidence source types:
- `user_machine`
- `independent_reproducible`
- `controlled_ci_cloud`

At least one entry must be a real user-owned machine:
- `source_type = user_machine`
- `user_owned = true`

The validator also requires CPU identity, OS, logical CPU count, 1-thread H/s, all-thread H/s and a reference/path to raw benchmark evidence.

Run:

```sh
go run ./experiments/mainnet-m5-evidence -input evidence.json
```

A PASS means only that the evidence set is structurally complete. It does **not** choose or freeze InitialTarget, PowLimit or Mainnet launch difficulty.

Do not fill the example manifest with invented CPU models or measurements.
