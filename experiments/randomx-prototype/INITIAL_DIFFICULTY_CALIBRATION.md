# VALDR Mainnet InitialDifficulty / PowLimit calibration

Status: **engineering candidate — not frozen**

Measured CI evidence from RandomX full-memory mining mode, 1 thread:

- macOS ARM64 runner: **68.2954 H/s**
- Ubuntu runner: **541.106 H/s**
- Windows runner: **387.989 H/s**

These are **CI runner measurements only**. They are useful to prove the benchmark path, but they are not representative enough to freeze Mainnet launch difficulty.

All three platforms produced the same VALDR prototype RandomX hash:

`180cec43033dbed2d5ec9e80f4a5fcb8ceba67ce4054304e2fc3c394aa766493`

## Target relation

For a uniform 256-bit PoW hash and a 600-second target:

`expected_work ≈ network_hashrate × 600`

`target = floor(2^256 / expected_work) - 1`

Candidate reference table:

| Network H/s | Expected hashes/block | Candidate target |
|---:|---:|---|
| 50 | 30,000 | `00022f3d9397b043b874df5e583356270c6cae376eba812918b66895a3f9fe15` |
| 100 | 60,000 | `0001179ec9cbd821dc3a6faf2c19ab138636571bb75d40948c5b344ad1fcff0a` |
| 250 | 150,000 | `00006fd91d84bcda58175fdfab3d77a168e2893e4958803b6b57ae8453fecc69` |
| 500 | 300,000 | `000037ec8ec25e6d2c0bafefd59ebbd0b471449f24ac401db5abd74229ff6634` |
| 1,000 | 600,000 | `00001bf647612f369605d7f7eacf5de85a38a24f9256200edad5eba114ffb319` |
| 5,000 | 3,000,000 | `00000597a7e03ca484679197fbc312c8120b53a983aad3362bc462536a998a37` |
| 10,000 | 6,000,000 | `000002cbd3f01e524233c8cbfde189640905a9d4c1d5699b15e23129b54cc51b` |
| 50,000 | 30,000,000 | `0000008f2a633943a6d72828cc604ead9b67885dc05de1ebd12d3d085775c104` |
| 100,000 | 60,000,000 | `0000004795319ca1d36b941466302756cdb3c42ee02ef0f5e8969e842bbae081` |

## Current engineering decision

Do **not** freeze InitialTarget from GitHub-hosted runner numbers.

Before Mainnet Genesis:

1. benchmark the actual VALDR miner on at least 5 representative user CPUs;
2. include at least one ordinary laptop, one mainstream desktop CPU and one higher-end desktop CPU;
3. benchmark both 1-thread and recommended automatic mining configuration;
4. estimate conservative expected launch network hashrate;
5. select InitialTarget for approximately 600 seconds at that expected launch hashrate;
6. choose PowLimit separately as an emergency lower-difficulty floor;
7. simulate DAA response from InitialTarget to PowLimit under 2×, 10× and 100× hashrate loss;
8. freeze exact 256-bit values only after rehearsal evidence.

## PowLimit policy candidate

PowLimit must be easier than InitialTarget, but not a development/test target.

Current candidate policy:

- size PowLimit so a **single ordinary supported CPU** can still advance the chain after a major hashrate departure;
- do not enable a Testnet-style automatic minimum-difficulty block;
- DAA may move toward PowLimit normally, but never beyond it.

The numerical PowLimit remains unresolved until representative CPU benchmarks exist.

Helper:

```sh
python3 experiments/randomx-prototype/target_calculator.py
```
