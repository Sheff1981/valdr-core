# VALDR Mainnet RandomX prototype

Experimental only. This branch does **not** change Testnet2 consensus.

Purpose:

1. pin upstream RandomX v1.2.3;
2. run upstream self-tests;
3. build a tiny VALDR hash/vector harness against the official RandomX C API;
4. run the official RandomX benchmark in verification mode on Linux/Windows/macOS CI;
5. record deterministic cross-platform hash equality before any Mainnet consensus code is changed.

Upstream source is downloaded at build time from `tevador/RandomX` tag `v1.2.3`.
No RandomX source is vendored into VALDR in this experiment.

## Local Linux/macOS

```sh
cd experiments/randomx-prototype
./build.sh
./build/valdr-randomx-vector
```

## Windows PowerShell

```powershell
cd experiments/randomx-prototype
./build.ps1
./build/Release/valdr-randomx-vector.exe
```

## Candidate mining blob

This prototype uses a temporary fixed binary sample blob only to prove deterministic hashing.
It is **not** the frozen Mainnet block-header serialization.

The final mining blob remains a Mainnet-spec decision.

## Gate

RandomX is not frozen for VALDR Mainnet until:

- upstream self-tests pass;
- VALDR vector output is identical on Linux, Windows and macOS;
- benchmark results are recorded;
- native packaging impact is reviewed;
- DAA + InitialTarget/PowLimit are evaluated from measured hashrate.
