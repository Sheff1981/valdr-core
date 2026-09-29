package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseBenchmarkWindowsShape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raw.txt")
	raw := `VALDR_RANDOMX_REAL_CPU_V1
date_utc=2026-09-29T18:00:00Z
os=Microsoft Windows NT 10.0.26100.0
cpu=Example Weak Laptop CPU
memory_bytes=8589934592
logical_cpus=8

=== threads=1 ===
RandomX benchmark v1.2.3
Performance: 42.125 hashes per second

=== threads=2 ===
Performance: 78.5 hashes per second

=== threads=4 ===
Performance: 140.0 hashes per second

=== threads=8 ===
Performance: 205.75 hashes per second
`
	if err := os.WriteFile(path, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	ev, err := parseBenchmark(path)
	if err != nil {
		t.Fatal(err)
	}
	if ev.CPU != "Example Weak Laptop CPU" || ev.LogicalCPUs != 8 {
		t.Fatalf("metadata=%+v", ev)
	}
	if ev.ThreadRates[1] != "42.125" || ev.ThreadRates[8] != "205.75" {
		t.Fatalf("rates=%v", ev.ThreadRates)
	}
	if ev.Class != "unclassified" || ev.Status != "MEASURED_UNCLASSIFIED" {
		t.Fatalf("unexpected classification: %+v", ev)
	}
}

func TestParseBenchmarkRejectsMissingAllThreads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raw.txt")
	raw := `VALDR_RANDOMX_REAL_CPU_V1
date_utc=2026-09-29T18:00:00Z
os=Windows
cpu=CPU
logical_cpus=4
=== threads=1 ===
Performance: 10 hashes per second
`
	if err := os.WriteFile(path, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := parseBenchmark(path); err == nil {
		t.Fatal("expected missing all-thread measurement error")
	}
}
