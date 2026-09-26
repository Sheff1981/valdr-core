# VALDR Testnet v0.2.9 consensus recalibration

**Date:** 26 September 2026  
**Source baseline:** Master-TZ v0.2.8 + Windows manual QA of CI #356  
**Status:** implementation evidence for Master-TZ v0.2.9

## Observed problem

The CI #356 Windows Desktop build was exercised on a real machine with the built-in single-thread SHA-256 miner. Observed effective hashrate was approximately 330–379 kH/s. Representative accepted-block solves shown by Desktop included:

- 34,561 hashes / 89.3 ms;
- 98,410 hashes / 273 ms;
- 461,842 hashes / 1.24 s.

One run reached 94 accepted blocks quickly. With the old fixed 50 Testnet VDR subsidy that represented 4,700 Testnet VDR. The main defect is not the nominal reward by itself: the old initial target and 60-block retarget allowed the chain to advance far faster than its 60-second target cadence.

## v0.2.9 active Testnet calibration

Active profile: `testnet2`  
Chain ID: `valdr-testnet-2`  
Target block interval: 60 s  
Fixed Testnet subsidy: 1 Testnet VDR/block  
Retarget interval: 10 blocks  
Measured retarget window: 540 s (10 headers / 9 intervals)  
Clamp: 135..2,160 s  
PoW limit leading-zero bits: 22  
Min-difficulty escape: 10 min

Initial target:

`0000031b5d43afe99ee43470e1337c3642e9d9254926038fdf6d1a2e57aaa21f`

This target is calibrated at approximately 5.4 million SHA-256 trials. The exact integer work value for the frozen target under `floor(2^256/(target+1))` is 5,399,999. At 360 kH/s that is about 15 seconds expected bootstrap time.

If the first window arrives at approximately 15 s/block, the first retarget clamps to one quarter of the target:

`000000c6d750ebfa67b90d1c384cdf0d90ba7649524980e3f7db468b95eaa887`

That corresponds to about 21.6 million expected trials, or approximately 60 seconds at the observed 360 kH/s calibration point.

## Genesis

Timestamp: `1790380800` (2026-09-26 00:00:00 UTC)  
Nonce: `22759786`  
Message: `VALDR genesis block | valdr-testnet-2 | 2026-09-26`  
Hash: `000000a065ed224c03ff3107b2d3a073906415b347600f2e83a8874371f15485`  
Magic: `900d7c51`

The Genesis hash was derived from the current V2 canonical header serializer and is below the frozen initial target.

## Scope boundary

This recalibration is Testnet-only. It does not define Mainnet subsidy, halving, maximum supply, coinbase maturity, launch difficulty or economics. Those remain Stage 15 work.
