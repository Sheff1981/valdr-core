package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
	desktopcore "github.com/Sheff1981/valdr-core/desktop"
	"github.com/Sheff1981/valdr-core/wallet"
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
	store := wallet.NewStore(paths.Wallets)
	sessions, err := desktopcore.NewWalletSessionManager(
		store,
		desktopcore.DefaultWalletAutoLock,
	)
	if err != nil {
		t.Fatal(err)
	}
	app := &App{
		paths:          paths,
		node:           node,
		walletStore:    store,
		walletSessions: sessions,
	}

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
	if len(state.UnlockedWallets) != 1 ||
		state.UnlockedWallets[0] != meta.Address {
		t.Fatalf("new Desktop wallet was not unlocked: %+v", state)
	}
	if state.WalletAutoLockMinutes != 15 {
		t.Fatalf("auto-lock minutes=%d want=15", state.WalletAutoLockMinutes)
	}

	app.LockWallet(meta.Address)
	state, err = app.GetState()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.UnlockedWallets) != 0 {
		t.Fatalf("locked wallet still reported unlocked: %+v", state)
	}
	if _, err := app.UnlockWallet(meta.Address, "wrong-passphrase"); !errors.Is(
		err,
		wallet.ErrWalletAuthentication,
	) {
		t.Fatalf("wrong passphrase error=%v", err)
	}
	if _, err := app.UnlockWallet(
		meta.Address,
		"desktop-test-passphrase",
	); err != nil {
		t.Fatal(err)
	}
	if err := app.SetWalletAutoLockMinutes(5); err != nil {
		t.Fatal(err)
	}
	state, err = app.GetState()
	if err != nil {
		t.Fatal(err)
	}
	if state.WalletAutoLockMinutes != 5 ||
		len(state.UnlockedWallets) != 1 {
		t.Fatalf("unexpected unlocked security state: %+v", state)
	}

	if _, err := app.ExportPrivateKey(
		meta.Address,
		"yes",
	); !errors.Is(err, ErrPrivateKeyExportConfirmation) {
		t.Fatalf("unsafe export confirmation error=%v", err)
	}
	privateKey, err := app.ExportPrivateKey(
		meta.Address,
		privateKeyExportConfirmation,
	)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := valdrcrypto.DecodePrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	address, err := valdrcrypto.AddressFromPublicKey(&decoded.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if address != meta.Address {
		t.Fatalf("exported key address=%s want=%s", address, meta.Address)
	}

	app.LockWallet(meta.Address)
	if _, err := app.ExportPrivateKey(
		meta.Address,
		privateKeyExportConfirmation,
	); !errors.Is(err, desktopcore.ErrWalletLocked) {
		t.Fatalf("locked export error=%v", err)
	}
}
