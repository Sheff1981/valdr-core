package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	desktopcore "github.com/Sheff1981/valdr-core/desktop"
)

func TestDesktopAppCreatesEncryptedWalletOnly(t *testing.T) {
	root := t.TempDir()
	paths := desktopcore.Paths{
		Root:     root,
		NodeData: filepath.Join(root, "node", "testnet"),
		Wallets:  filepath.Join(root, "wallets"),
		Logs:     filepath.Join(root, "logs"),
		Network:  config.NetworkTestnetV02,
	}
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}

	node, err := desktopcore.NewNodeManager(desktopcore.NodeProcessConfig{
		BinaryPath: filepath.Join(root, "valdrd-not-started"),
		Network:    config.NetworkTestnetV02,
		DataDir:    paths.NodeData,
		NodeID:     "desktop-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	app := &App{paths: paths, node: node}

	meta, err := app.CreateWallet(
		"desktop-wallet",
		"desktop-test-passphrase",
	)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Address == "" {
		t.Fatal("Desktop wallet has empty address")
	}

	raw, err := os.ReadFile(
		filepath.Join(paths.Wallets, meta.Address+".json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("private_key")) {
		t.Fatal("Desktop wallet file contains private_key field")
	}
	if !bytes.Contains(raw, []byte(`"version": 2`)) {
		t.Fatalf("Desktop wallet is not v2 encrypted: %s", raw)
	}

	state, err := app.GetState()
	if err != nil {
		t.Fatal(err)
	}
	if state.MainnetEnabled {
		t.Fatal("Mainnet unexpectedly enabled in Stage 12 Desktop")
	}
	if state.Network != config.NetworkTestnetV02 ||
		state.ChainID != "valdr-testnet-1" {
		t.Fatalf("unexpected Desktop network state: %+v", state)
	}
	if len(state.Wallets) != 1 ||
		state.Wallets[0].Address != meta.Address {
		t.Fatalf("unexpected Desktop wallets: %+v", state.Wallets)
	}
}
