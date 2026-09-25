package desktop

import (
	"errors"
	"os/exec"
	"slices"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
)

func TestDesktopNodeArgsUseCanonicalOutboundNode(t *testing.T) {
	cfg := NodeProcessConfig{
		Network: config.NetworkTestnetV02,
		DataDir: "/tmp/valdr-desktop-testnet",
		NodeID:  "desktop-test",
		RPCPort: 28332,
		Seeds: []string{
			"seed-a.example:17333",
			"",
			" seed-b.example:17333 ",
		},
	}
	args, err := desktopNodeArgs(cfg)
	if err != nil {
		t.Fatal(err)
	}

	required := [][]string{
		{"start"},
		{"--network", "testnet"},
		{"--data", cfg.DataDir},
		{"--node-id", cfg.NodeID},
		{"--outbound-only"},
		{"--managed-stdin-shutdown"},
		{"--rpc-host", "127.0.0.1"},
		{"--rpc-port", "28332"},
		{"--seed", "seed-a.example:17333"},
		{"--seed", "seed-b.example:17333"},
	}
	for _, sequence := range required {
		if !containsSequence(args, sequence) {
			t.Fatalf("args=%v missing sequence %v", args, sequence)
		}
	}
	if slices.Contains(args, "--advertise-address") ||
		slices.Contains(args, "--p2p-host") {
		t.Fatalf("Desktop node unexpectedly configures inbound P2P: %v", args)
	}
}

func TestDesktopNodeRejectsLegacyAndMainnet(t *testing.T) {
	for _, network := range []string{
		config.NetworkLegacyV01,
		"mainnet",
	} {
		_, err := NewNodeManager(NodeProcessConfig{
			BinaryPath: "/tmp/valdrd",
			Network:    network,
			DataDir:    "/tmp/valdr",
		})
		if !errors.Is(err, ErrDesktopMainnet) {
			t.Fatalf(
				"network=%q error=%v want ErrDesktopMainnet",
				network,
				err,
			)
		}
	}
}

func TestDesktopNodeDefaultsToTestnet(t *testing.T) {
	manager, err := NewNodeManager(NodeProcessConfig{
		BinaryPath: "/tmp/valdrd",
		DataDir:    "/tmp/valdr",
	})
	if err != nil {
		t.Fatal(err)
	}
	if manager.config.Network != config.NetworkTestnetV02 {
		t.Fatalf(
			"network=%q want=%q",
			manager.config.Network,
			config.NetworkTestnetV02,
		)
	}
	if manager.Endpoint() != "http://127.0.0.1:17332" {
		t.Fatalf("endpoint=%q", manager.Endpoint())
	}
}

func containsSequence(values []string, sequence []string) bool {
	if len(sequence) == 0 {
		return true
	}
	for i := 0; i+len(sequence) <= len(values); i++ {
		if slices.Equal(values[i:i+len(sequence)], sequence) {
			return true
		}
	}
	return false
}


func TestDesktopNodeSurfacesUnexpectedExitButNotIntentionalStop(t *testing.T) {
	manager, err := NewNodeManager(NodeProcessConfig{
		BinaryPath: "/tmp/valdrd",
		Network:    config.NetworkTestnetV02,
		DataDir:    "/tmp/valdr-desktop-crash-test",
		NodeID:     "desktop-crash-test",
	})
	if err != nil {
		t.Fatal(err)
	}

	crashed := &exec.Cmd{}
	manager.command = crashed
	manager.recordProcessExit(crashed, errors.New("exit status 2"))
	if manager.Running() {
		t.Fatal("manager still reports running after child exit")
	}
	if !errors.Is(manager.LastExitError(), ErrNodeExitedUnexpectedly) {
		t.Fatalf("last exit error=%v want ErrNodeExitedUnexpectedly", manager.LastExitError())
	}

	manager.lastExit = nil
	stopped := &exec.Cmd{}
	manager.command = stopped
	manager.stopping = true
	manager.recordProcessExit(stopped, errors.New("signal: terminated"))
	if manager.LastExitError() != nil {
		t.Fatalf("intentional stop was surfaced as crash: %v", manager.LastExitError())
	}
}

func TestDesktopNodeStartClearsPreviousCrashState(t *testing.T) {
	manager, err := NewNodeManager(NodeProcessConfig{
		BinaryPath: "/path/that/does/not/exist/valdrd",
		Network:    config.NetworkTestnetV02,
		DataDir:    "/tmp/valdr-desktop-restart-test",
		NodeID:     "desktop-restart-test",
	})
	if err != nil {
		t.Fatal(err)
	}

	manager.lastExit = ErrNodeExitedUnexpectedly
	if err := manager.Start(); err == nil {
		t.Fatal("expected missing binary start to fail")
	}
	if manager.LastExitError() != nil {
		t.Fatalf("restart attempt did not clear previous crash state: %v", manager.LastExitError())
	}
}


func TestDesktopNodeArgsSupportExplicitAdvancedPublicNode(t *testing.T) {
	cfg := NodeProcessConfig{
		Network:          config.NetworkTestnetV02,
		DataDir:          "/tmp/valdr-desktop-public-testnet",
		NodeID:           "desktop-public-test",
		RPCPort:          28332,
		PublicNode:       true,
		AdvertiseAddress: "node.example.com:17333",
	}
	args, err := desktopNodeArgs(cfg)
	if err != nil {
		t.Fatal(err)
	}

	required := [][]string{
		{"--p2p-host", "0.0.0.0"},
		{"--advertise-address", "node.example.com:17333"},
		{"--rpc-host", "127.0.0.1"},
	}
	for _, sequence := range required {
		if !containsSequence(args, sequence) {
			t.Fatalf("args=%v missing sequence %v", args, sequence)
		}
	}
	if slices.Contains(args, "--outbound-only") {
		t.Fatalf("public Desktop node unexpectedly remains outbound-only: %v", args)
	}
}

func TestDesktopPublicNodeAdvertiseAddressValidation(t *testing.T) {
	valid := []string{
		"node.example.com:17333",
		"8.8.8.8:17333",
		"[2606:4700:4700::1111]:17333",
	}
	for _, value := range valid {
		if err := ValidatePublicNodeAdvertiseAddress(value); err != nil {
			t.Fatalf("valid address %q rejected: %v", value, err)
		}
	}

	invalid := []string{
		"",
		"localhost:17333",
		"node.local:17333",
		"127.0.0.1:17333",
		"10.0.0.5:17333",
		"192.168.1.5:17333",
		"node-without-domain:17333",
		"node.valdr.example",
		"node.valdr.example:0",
	}
	for _, value := range invalid {
		if err := ValidatePublicNodeAdvertiseAddress(value); !errors.Is(
			err,
			ErrDesktopPublicNodeAddress,
		) {
			t.Fatalf("invalid address %q error=%v", value, err)
		}
	}
}

func TestDesktopNodePublicModeCanBeReconfiguredForNextStart(t *testing.T) {
	manager, err := NewNodeManager(NodeProcessConfig{
		BinaryPath: "/tmp/valdrd",
		Network:    config.NetworkTestnetV02,
		DataDir:    "/tmp/valdr-public-config",
		NodeID:     "desktop-public-config",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.ConfigurePublicNode(
		true,
		"node.example.com:17333",
	); err != nil {
		t.Fatal(err)
	}
	if !manager.config.PublicNode ||
		manager.config.AdvertiseAddress != "node.example.com:17333" {
		t.Fatalf("unexpected public-node config: %+v", manager.config)
	}
}
