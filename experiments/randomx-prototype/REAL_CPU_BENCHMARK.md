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

The project owner is not required to provide five physical computers.

Required evidence:

1. **at least one real user-owned computer** running the VALDR RandomX benchmark pack;
2. benchmark evidence covering **at least three CPU performance classes**:
   - low/ordinary laptop class;
   - mainstream desktop class;
   - higher-performance desktop class;
3. CI evidence on supported operating systems remains useful for build/determinism verification, but is not used alone to freeze launch difficulty.

The three CPU classes may be covered by a combination of:
- the user's real machine;
- independent reproducible benchmark measurements;
- controlled CI/cloud hardware where CPU identity and run conditions are recorded.

Collect CPU model, RAM, OS, 1/2/4-thread H/s where available, all-logical-CPU H/s and any allocation/JIT failures.

Final InitialDifficulty must be conservative and must be validated by DAA simulations before Mainnet Genesis.
