# VALDR RandomX real-CPU benchmark pack

Purpose: collect representative real hardware evidence before freezing Mainnet `InitialDifficulty`, `InitialTarget` and `PowLimit`.

The benchmark does not change Testnet2 or Mainnet consensus.

## Windows

Place `randomx-benchmark.exe` and `benchmark-real-cpu.ps1` in one folder, then run:

```powershell
powershell -ExecutionPolicy Bypass -File .\benchmark-real-cpu.ps1
```

Result: `valdr-randomx-real-cpu.txt`.

## Linux/macOS

```sh
chmod +x benchmark-real-cpu.sh randomx-benchmark
./benchmark-real-cpu.sh ./randomx-benchmark
```

Result: `valdr-randomx-real-cpu.txt`.

## Minimum evidence before Mainnet freeze

At least five independent computers:
1. ordinary laptop;
2. ordinary desktop;
3. modern midrange desktop;
4. higher-end desktop;
5. one additional independent machine.

Collect CPU model, RAM, OS, 1/2/4-thread H/s where available, all-logical-CPU H/s and any allocation/JIT failures.

Do not use GitHub-hosted runner H/s as final launch calibration.
