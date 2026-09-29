package main

import (
	"strings"
	"testing"
)

func validEvidence() []Evidence {
	return []Evidence{
		{
			ID: "laptop-1", Class: "low_power_laptop", SourceType: "user_machine",
			UserOwned: true, CPU: "Example Laptop CPU", OS: "Windows",
			LogicalCPUs: 8, OneThreadHS: "50.1", AllThreadsHS: "250.2",
			RawEvidence: "valdr-randomx-real-cpu.txt",
		},
		{
			ID: "desktop-1", Class: "mainstream_desktop", SourceType: "independent_reproducible",
			CPU: "Example Desktop CPU", OS: "Linux",
			LogicalCPUs: 16, OneThreadHS: "100.0", AllThreadsHS: "800.0",
			RawEvidence: "source://desktop-1",
		},
		{
			ID: "high-1", Class: "higher_performance_desktop", SourceType: "controlled_ci_cloud",
			CPU: "Example High CPU", OS: "Linux",
			LogicalCPUs: 32, OneThreadHS: "150.0", AllThreadsHS: "1500.0",
			RawEvidence: "source://high-1",
		},
	}
}

func TestValidateEvidencePassesRequiredStructure(t *testing.T) {
	report, err := validateEvidence(validEvidence())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(report, "M5_EVIDENCE_STRUCTURE=PASS") {
		t.Fatalf("unexpected report: %s", report)
	}
}

func TestValidateEvidenceRequiresUserOwnedMachine(t *testing.T) {
	e := validEvidence()
	e[0].UserOwned = false
	e[0].SourceType = "independent_reproducible"
	if _, err := validateEvidence(e); err == nil || !strings.Contains(err.Error(), "user-owned") {
		t.Fatalf("error=%v", err)
	}
}

func TestValidateEvidenceRequiresAllThreeClasses(t *testing.T) {
	e := validEvidence()
	e[2].Class = "mainstream_desktop"
	if _, err := validateEvidence(e); err == nil || !strings.Contains(err.Error(), "higher_performance_desktop") {
		t.Fatalf("error=%v", err)
	}
}

func TestValidateEvidenceRejectsFakeUserOwnershipShape(t *testing.T) {
	e := validEvidence()
	e[0].SourceType = "controlled_ci_cloud"
	if _, err := validateEvidence(e); err == nil || !strings.Contains(err.Error(), "user_owned requires") {
		t.Fatalf("error=%v", err)
	}
}
