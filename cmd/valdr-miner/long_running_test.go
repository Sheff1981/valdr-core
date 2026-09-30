package main

import (
	"os"
	"strings"
	"testing"
)

func TestMinerStartHasNoFixedBlockDeadline(t *testing.T) {
	raw, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	if strings.Contains(source, "mineBlockRPCTimeout") {
		t.Fatal("start miner still has a fixed per-block RPC timeout")
	}
	for _, marker := range []string{
		"signal.NotifyContext(",
		"client.Call(",
		"runCtx,",
		"case <-runCtx.Done():",
	} {
		if !strings.Contains(source, marker) {
			t.Fatalf("long-running miner cancellation contract missing %q", marker)
		}
	}
}
