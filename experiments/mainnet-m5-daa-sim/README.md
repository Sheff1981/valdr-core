# VALDR Mainnet M5 DAA simulation

Status: **engineering candidate only; M5 remains open until exact values are independently reproduced and frozen in a new Master-TZ.**

This simulation reuses the production-candidate `consensus.MainnetNextTargetLWMA` implementation and the exact M5 InitialTarget candidate derived from the real user-owned i5-8265U measurement.

It compares PowLimit policies that are 2x, 4x and 8x easier than InitialTarget in work terms over 120 deterministic blocks for:
- stable hashrate;
- 2x / 10x / 100x increase;
- 2x / 10x / 100x decrease.

The mining model uses the exact expected solve time `work / hashrate`, rounded to the nearest whole second. It is intentionally deterministic so every platform must reproduce the same matrix.

## Selection criterion

Use the **least permissive** PowLimit that still leaves adjustment headroom after a sustained 2x hashrate loss.

Observed candidate behavior:
- 2x PowLimit reaches the floor under a 2x loss, leaving no further DAA easing headroom.
- 4x PowLimit does not reach the floor under a 2x loss and converges near the LWMA equilibrium.
- 8x also passes that condition, but permits another 2x weakening of minimum work versus 4x.

Therefore **4x is the current policy candidate**, subject to independent reproduction and final freeze.

Under a sustained 10x hashrate loss, the deterministic last-30-block averages are:
- 2x: 3000 s;
- 4x: 1500 s;
- 8x: 750 s.

This trade-off is recorded explicitly rather than silently choosing the easiest recovery limit.
