package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
)

type Evidence struct {
	ID            string `json:"id"`
	Class         string `json:"class"`
	SourceType    string `json:"source_type"`
	UserOwned     bool   `json:"user_owned"`
	CPU           string `json:"cpu"`
	OS            string `json:"os"`
	MemoryBytes   uint64 `json:"memory_bytes"`
	LogicalCPUs   int    `json:"logical_cpus"`
	OneThreadHS   string `json:"one_thread_hs"`
	AllThreadsHS  string `json:"all_threads_hs"`
	RawEvidence   string `json:"raw_evidence"`
}

var requiredClasses = []string{
	"low_power_laptop",
	"mainstream_desktop",
	"higher_performance_desktop",
}

func main() {
	input := flag.String("input", "", "JSON evidence manifest")
	flag.Parse()
	if *input == "" {
		fmt.Fprintln(os.Stderr, "error: -input is required")
		os.Exit(2)
	}

	raw, err := os.ReadFile(*input)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
	var evidence []Evidence
	if err := json.Unmarshal(raw, &evidence); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	report, err := validateEvidence(evidence)
	if err != nil {
		fmt.Fprintln(os.Stderr, "M5_EVIDENCE_INCOMPLETE:", err)
		os.Exit(1)
	}
	fmt.Print(report)
}

func validateEvidence(items []Evidence) (string, error) {
	if len(items) < 3 {
		return "", errors.New("need evidence covering at least three CPU classes")
	}

	required := map[string]bool{}
	for _, c := range requiredClasses {
		required[c] = false
	}
	userOwned := false
	seenIDs := map[string]struct{}{}

	for i, e := range items {
		if e.ID == "" {
			return "", fmt.Errorf("entry %d: id is required", i)
		}
		if _, ok := seenIDs[e.ID]; ok {
			return "", fmt.Errorf("duplicate id %q", e.ID)
		}
		seenIDs[e.ID] = struct{}{}

		if _, ok := required[e.Class]; !ok {
			return "", fmt.Errorf("entry %q: unsupported class %q", e.ID, e.Class)
		}
		required[e.Class] = true

		switch e.SourceType {
		case "user_machine", "independent_reproducible", "controlled_ci_cloud":
		default:
			return "", fmt.Errorf("entry %q: unsupported source_type %q", e.ID, e.SourceType)
		}
		if e.UserOwned {
			if e.SourceType != "user_machine" {
				return "", fmt.Errorf("entry %q: user_owned requires source_type=user_machine", e.ID)
			}
			userOwned = true
		}

		if e.CPU == "" || e.OS == "" || e.LogicalCPUs <= 0 {
			return "", fmt.Errorf("entry %q: cpu, os and logical_cpus are required", e.ID)
		}
		if e.OneThreadHS == "" || e.AllThreadsHS == "" {
			return "", fmt.Errorf("entry %q: 1-thread and all-thread H/s are required", e.ID)
		}
		if e.RawEvidence == "" {
			return "", fmt.Errorf("entry %q: raw_evidence path/reference is required", e.ID)
		}
	}

	for _, c := range requiredClasses {
		if !required[c] {
			return "", fmt.Errorf("missing CPU class %s", c)
		}
	}
	if !userOwned {
		return "", errors.New("at least one real user-owned machine is mandatory")
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Class < items[j].Class })
	out := "M5_EVIDENCE_STRUCTURE=PASS\n"
	out += "user_owned_machine=present\n"
	for _, c := range requiredClasses {
		out += fmt.Sprintf("class_%s=present\n", c)
	}
	out += fmt.Sprintf("evidence_entries=%d\n", len(items))
	out += "note=This validates evidence completeness only; InitialTarget and PowLimit are not frozen.\n"
	return out, nil
}
