# VALDR M5 benchmark ingest

This tool normalizes the raw `valdr-randomx-real-cpu.txt` output into machine-readable JSON.

It deliberately does **not** auto-assign a CPU performance class. Automatic class guessing from one machine would be evidence fabrication. The output stays:

- `status = MEASURED_UNCLASSIFIED`
- `class = unclassified`

until the benchmark is reviewed against the full M5 evidence set.

Windows package usage is automatic after the benchmark. Manual usage:

```sh
go run ./experiments/mainnet-m5-ingest -input valdr-randomx-real-cpu.txt -output valdr-m5-machine.json
```

The normalized JSON records CPU/OS identity and exact measured 1-thread / 2-thread / 4-thread / all-logical-CPU rates when present.
