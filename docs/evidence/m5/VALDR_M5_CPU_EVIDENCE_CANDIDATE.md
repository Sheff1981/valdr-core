# VALDR Mainnet M5 CPU evidence candidate — 2026-09-29

Status: **candidate evidence set; M5 not frozen**

This file records the M5 CPU-performance evidence used for InitialTarget / PowLimit calibration under Master-TZ v0.3.7-DRAFT.

## Evidence set

| Class | Source | CPU | Logical CPUs | 1-thread H/s | all-thread H/s |
|---|---|---|---:|---:|---:|
| low_power_laptop | real user-owned Windows machine | Intel Core i5-8265U | 8 | 304.515 | 418.723 |
| mainstream_desktop | controlled CI/cloud performance proxy | Apple M1 (Virtual) | 3 | 145.113 | 613.962 |
| higher_performance_desktop | controlled CI/cloud | AMD EPYC 9V74 | 4 | 461.635 | 1576.72 |
| higher_performance_desktop reproduction | controlled CI/cloud | AMD EPYC 9V74 | 4 | 457.458 | 1504.99 |

The class labels describe **performance classes for M5 calibration**. The controlled runners are not claimed to be literal end-user desktop chassis. v0.3.7 explicitly permits controlled CI/cloud hardware with recorded CPU identity to cover required CPU performance classes.

## Provenance

- Real user-owned benchmark: 2026-09-29T19:13:38Z, RandomX v1.2.3, full-memory mode.
- Controlled run: GitHub Actions run `36620080973`, commit `8832574b87e0fa941c5f4036a4a553cc67f75c83`.
- macOS job: `109582933042`, artifact `11057023893`.
- Windows job: `109582932715`, artifact `11057992634`.
- Linux reproduction job: `109582932969`, artifact `11058252512`.

All benchmark outputs reported the same calculated/reference RandomX result for the measured nonce set.

## Interpretation boundary

This evidence set satisfies the **shape** required to proceed with M5 calibration:
- at least one real user-owned machine;
- low/ordinary, mainstream and higher-performance CPU classes;
- CPU identity and run conditions recorded.

It does **not** freeze InitialTarget, PowLimit, Genesis or Mainnet launch.
