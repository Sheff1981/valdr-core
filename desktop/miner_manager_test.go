package desktop

import (
	"errors"
	"os/exec"
	"slices"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/wallet"
)

func TestDesktopMinerArgsUseLocalNodeAndExplicitReward(t *testing.T) {
	rewardWallet, err := wallet.New("desktop-miner-reward")
	if err != nil {
		t.Fatal(err)
	}
	cfg := MinerProcessConfig{
		NodeEndpoint: "http://127.0.0.1:17332",
		PIDFile:      "/tmp/valdr-desktop-miner.pid",
		Interval:     2 * time.Second,
	}
	args, err := desktopMinerArgs(cfg, rewardWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	required := [][]string{
		{"start"},
		{"--node", cfg.NodeEndpoint},
		{"--reward-address", rewardWallet.Address},
		{"--blocks", "0"},
		{"--interval", "2s"},
		{"--pid-file", cfg.PIDFile},
	}
	for _, sequence := range required {
		if !containsSequence(args, sequence) {
			t.Fatalf("args=%v missing sequence %v", args, sequence)
		}
	}
	if slices.Contains(args, "--network") ||
		slices.Contains(args, "--advertise-address") {
		t.Fatalf("Desktop miner unexpectedly controls consensus/network profile: %v", args)
	}
}

func TestDesktopMinerRejectsInvalidRewardAddress(t *testing.T) {
	cfg := MinerProcessConfig{
		NodeEndpoint: "http://127.0.0.1:17332",
		PIDFile:      "/tmp/valdr-desktop-miner.pid",
		Interval:     time.Second,
	}
	if _, err := desktopMinerArgs(cfg, "not-a-valdr-address"); !errors.Is(
		err,
		ErrDesktopMinerRewardAddress,
	) {
		t.Fatalf("error=%v want ErrDesktopMinerRewardAddress", err)
	}
}

func TestDesktopMinerDefaultsIntervalAndSurfacesUnexpectedExit(t *testing.T) {
	manager, err := NewMinerManager(MinerProcessConfig{
		BinaryPath:   "/tmp/valdr-miner",
		NodeEndpoint: "http://127.0.0.1:17332",
		PIDFile:      "/tmp/valdr-desktop-miner.pid",
	})
	if err != nil {
		t.Fatal(err)
	}
	if manager.config.Interval != time.Second {
		t.Fatalf("interval=%v want=1s", manager.config.Interval)
	}

	crashed := &exec.Cmd{}
	manager.command = crashed
	manager.recordProcessExit(crashed, errors.New("exit status 2"))
	if manager.Running() {
		t.Fatal("manager still reports running after child exit")
	}
	if !errors.Is(manager.LastExitError(), ErrDesktopMinerExited) {
		t.Fatalf("last exit error=%v want ErrDesktopMinerExited", manager.LastExitError())
	}

	manager.lastExit = nil
	stopped := &exec.Cmd{}
	manager.command = stopped
	manager.stopping = true
	manager.recordProcessExit(stopped, errors.New("signal: terminated"))
	if manager.LastExitError() != nil {
		t.Fatalf("intentional stop surfaced as crash: %v", manager.LastExitError())
	}
}
