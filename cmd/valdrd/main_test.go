package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"version"}, &out, &errOut); code != 0 {
		t.Fatalf("version exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "VALDR") || !strings.Contains(out.String(), "0.2.0-dev") {
		t.Fatalf("unexpected version output: %s", out.String())
	}
}

func TestInitCommandDefaultsToTestnet2(t *testing.T) {
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
	if !strings.Contains(out.String(), "valdr-testnet-2") ||
		!strings.Contains(out.String(), "\"network\": \"testnet2\"") {
		t.Fatalf("init output missing active Testnet2 identity: %s", out.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "chain-v2")); err != nil {
		t.Fatalf("v0.2 Badger directory missing: %v", err)
	}
}


func TestStartRejectsNonLoopbackRPCBind(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{
		"start",
		"--data", filepath.Join(t.TempDir(), "node"),
		"--rpc-host", "0.0.0.0",
	}, &out, &errOut)
	if code != 2 {
		t.Fatalf("start exit=%d want=2 stdout=%s stderr=%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "privileged RPC must bind to localhost/loopback unless --rpc-allow-non-loopback is explicitly set") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestLoopbackRPCHostValidation(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "::1", "localhost", " LOCALHOST "} {
		if !isLoopbackRPCHost(host) {
			t.Fatalf("loopback host %q rejected", host)
		}
	}
	for _, host := range []string{"0.0.0.0", "::", "192.0.2.10", "example.org", ""} {
		if isLoopbackRPCHost(host) {
			t.Fatalf("non-loopback host %q accepted", host)
		}
		if rpcBindAllowed(host, false) {
			t.Fatalf("non-loopback host %q allowed without explicit opt-in", host)
		}
		if host != "" && !rpcBindAllowed(host, true) {
			t.Fatalf("non-loopback host %q rejected with explicit opt-in", host)
		}
	}
}
