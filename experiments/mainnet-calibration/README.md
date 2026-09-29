# VALDR Mainnet M5 calibration tool

Status: **engineering utility only; Mainnet disabled**

This tool independently converts measured RandomX hashrate into an exact 256-bit InitialTarget candidate.

It does **not** choose the final hashrate class, safety margin, PowLimit or launch difficulty.

Formula:

- measured hashrate = H hashes/second;
- target interval = T seconds;
- explicit conservative multiplier = N/D;
- InitialWork = floor(H * T * N / D);
- InitialTarget = floor(2^256 / InitialWork) - 1.

All arithmetic is exact rational/integer arithmetic. No floating-point calculations are used.

Example:

```sh
go run ./experiments/mainnet-calibration -hashrate-hs 100 -seconds 600 -safety-num 9 -safety-den 10
```

Before M5 can be frozen, v0.3.7 still requires:
1. at least one benchmark from a real user-owned computer;
2. evidence covering low-power laptop, mainstream desktop and higher-performance desktop classes;
3. an explicit conservative reference/safety policy;
4. an exact 256-bit InitialTarget;
5. an exact PowLimit;
6. independent reproduction plus DAA simulations.

The tool deliberately leaves PowLimit unresolved because the current Master-TZ has not yet specified the policy relating PowLimit to InitialTarget.
