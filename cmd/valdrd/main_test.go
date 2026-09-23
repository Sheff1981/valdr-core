package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
)

func TestVersionCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"version"}, &out, &errOut); code != 0 {
		t.Fatalf("version exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "VALDR") || !strings.Contains(out.String(), config.Version) {
		t.Fatalf("unexpected version output: %s", out.String())
	}
}

func TestInitCommand(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "node1")
	var out, errOut bytes.Buffer
	if code := run([]string{"init", "--data", dir}, &out, &errOut); code != 0 {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}

	info, err := os.Stat(filepath.Join(dir, "node.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("node.json permissions = %o, want no group/other access", info.Mode().Perm())
	}
	if !strings.Contains(out.String(), "valdr-devnet-1") {
		t.Fatalf("init output missing chain id: %s", out.String())
	}
}

func TestUniqueAddressesPreservesFirstOccurrence(t *testing.T) {
	got := uniqueAddresses([]string{
		"127.0.0.1:7333",
		"127.0.0.1:7433",
		"127.0.0.1:7333",
	})
	want := []string{"127.0.0.1:7333", "127.0.0.1:7433"}
	if len(got) != len(want) {
		t.Fatalf("unique addresses len=%d want=%d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unique addresses[%d]=%q want=%q", i, got[i], want[i])
		}
	}
}
