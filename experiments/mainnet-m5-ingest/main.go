package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type MachineEvidence struct {
	Format       string            `json:"format"`
	DateUTC      string            `json:"date_utc"`
	OS           string            `json:"os"`
	CPU          string            `json:"cpu"`
	MemoryBytes  uint64            `json:"memory_bytes,omitempty"`
	MemoryKiB    uint64            `json:"memory_kib,omitempty"`
	LogicalCPUs  int               `json:"logical_cpus"`
	ThreadRates  map[int]string    `json:"thread_hashrate_hs"`
	SourceFile   string            `json:"source_file"`
	Status       string            `json:"status"`
	Class        string            `json:"class"`
	SourceType   string            `json:"source_type"`
	UserOwned    bool              `json:"user_owned"`
}

var (
	threadRE = regexp.MustCompile(`^=== threads=([0-9]+) ===$`)
	rateRE   = regexp.MustCompile(`^Performance:\s*([0-9]+(?:\.[0-9]+)?)\s+hashes per second\s*$`)
)

func main() {
	in := flag.String("input", "valdr-randomx-real-cpu.txt", "raw benchmark output")
	out := flag.String("output", "valdr-m5-machine.json", "normalized machine evidence JSON")
	flag.Parse()

	ev, err := parseBenchmark(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
	raw, err := json.MarshalIndent(ev, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(*out, raw, 0644); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
	fmt.Printf("Saved: %s\n", *out)
}

func parseBenchmark(path string) (*MachineEvidence, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ev := &MachineEvidence{
		ThreadRates: make(map[int]string),
		SourceFile:  path,
		Status:      "MEASURED_UNCLASSIFIED",
		Class:       "unclassified",
		SourceType:  "user_machine",
		UserOwned:   true,
	}

	s := bufio.NewScanner(f)
	currentThread := 0
	first := true

	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if first {
			first = false
			if line != "VALDR_RANDOMX_REAL_CPU_V1" {
				return nil, errors.New("unexpected benchmark format header")
			}
			ev.Format = line
			continue
		}
		if m := threadRE.FindStringSubmatch(line); m != nil {
			n, _ := strconv.Atoi(m[1])
			currentThread = n
			continue
		}
		if m := rateRE.FindStringSubmatch(line); m != nil {
			if currentThread <= 0 {
				return nil, errors.New("performance line found before thread section")
			}
			ev.ThreadRates[currentThread] = m[1]
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			switch k {
			case "date_utc":
				ev.DateUTC = v
			case "os":
				ev.OS = v
			case "cpu":
				ev.CPU = v
			case "memory_bytes":
				ev.MemoryBytes, _ = strconv.ParseUint(v, 10, 64)
			case "memory_kib":
				ev.MemoryKiB, _ = strconv.ParseUint(v, 10, 64)
			case "logical_cpus":
				ev.LogicalCPUs, _ = strconv.Atoi(v)
			}
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if ev.Format == "" || ev.CPU == "" || ev.OS == "" || ev.LogicalCPUs <= 0 {
		return nil, errors.New("benchmark metadata is incomplete")
	}
	if _, ok := ev.ThreadRates[1]; !ok {
		return nil, errors.New("missing 1-thread measurement")
	}
	if _, ok := ev.ThreadRates[ev.LogicalCPUs]; !ok {
		return nil, errors.New("missing all-logical-CPU measurement")
	}
	return ev, nil
}
