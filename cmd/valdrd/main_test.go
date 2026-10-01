package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/p2p"
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


func TestJoinHostPortSupportsIPv6(t *testing.T) {
	if got := joinHostPort("127.0.0.1", 17332); got != "127.0.0.1:17332" {
		t.Fatalf("IPv4 address=%q want 127.0.0.1:17332", got)
	}
	if got := joinHostPort("::1", 17332); got != "[::1]:17332" {
		t.Fatalf("IPv6 address=%q want [::1]:17332", got)
	}
}


func TestPrivilegedRPCWriteTimeoutCoversMiningDeadline(t *testing.T) {
	if privilegedRPCWriteTimeout <= 5*time.Minute {
		t.Fatalf(
			"privileged RPC write timeout=%s must exceed valdr-miner mineBlock deadline",
			privilegedRPCWriteTimeout,
		)
	}
}

func TestProbePeerCommandConnectsTestnet2(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	target, err := p2p.NewNode(p2p.NodeConfig{
		NodeID:         "probe-target",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		HeightProvider: func() uint64 { return 42 },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := target.Start(); err != nil {
		t.Fatal(err)
	}
	defer target.Close()

	var out, errOut bytes.Buffer
	code := run([]string{
		"probe-peer",
		"--network", config.NetworkTestnetV029,
		"--address", target.Address(),
		"--timeout", "2s",
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("probe-peer exit=%d stderr=%s", code, errOut.String())
	}

	var got struct {
		Reachable       bool   `json:"reachable"`
		Network         string `json:"network"`
		ChainID         string `json:"chain_id"`
		Address         string `json:"address"`
		PeerNodeID      string `json:"peer_node_id"`
		ProtocolVersion uint32 `json:"protocol_version"`
		PeerHeight      uint64 `json:"peer_height"`
		Inbound         bool   `json:"inbound"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("decode probe output: %v output=%s", err, out.String())
	}
	if !got.Reachable ||
		got.Network != config.NetworkTestnetV029 ||
		got.ChainID != profile.ChainID ||
		got.Address != target.Address() ||
		got.PeerNodeID != "probe-target" ||
		got.ProtocolVersion != uint32(profile.ProtocolMax) ||
		got.PeerHeight != 42 ||
		got.Inbound {
		t.Fatalf("unexpected probe result: %+v", got)
	}
}

func TestProbePeerCommandRequiresAddress(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"probe-peer"}, &out, &errOut); code != 2 {
		t.Fatalf("probe-peer exit=%d want=2 stdout=%s stderr=%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "usage: valdrd probe-peer") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}
