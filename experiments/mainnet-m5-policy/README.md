# VALDR Mainnet M5 policy candidate calculator

Status: **candidate engineering tool only — no M5 freeze**

The tool turns an explicitly selected reference RandomX hashrate into an exact InitialTarget and a candidate PowLimit relation.

Default reference:
- real user-owned low-power laptop all-thread result: `418.723 H/s`;
- target interval: `600 s`;
- safety multiplier: `1/1`;
- PowLimit work: `InitialWork / 4`.

Default candidate output:
- InitialWork = `251233`;
- InitialTarget = `000042c78dcd2f09f6673e99417b7b7d0dbb0ddcf732f964dc73fce1c6b50768`;
- PowLimitWork = `62808`;
- PowLimitTarget = `00010b1e7ce2d1034b183a2be3b26504779a86ab0ed6197fb90be6d3c07b200c`.

These values are **not frozen**. The 4x PowLimit relation is a candidate chosen for simulation, not a Master-TZ decision.

All calculations use exact `big.Rat` / `big.Int` arithmetic. No binary floating-point is used.

Run:

```sh
go run ./experiments/mainnet-m5-policy -hashrate-hs 418.723 -seconds 600 -safety-num 1 -safety-den 1 -powlimit-factor 4
```

Before freeze:
1. compare 2x / 4x / 8x PowLimit candidates;
2. run DAA simulations with the exact candidate values;
3. independently reproduce 256-bit outputs;
4. only then record the selected values in a new Master-TZ revision.
