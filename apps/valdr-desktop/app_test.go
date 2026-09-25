package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	persisted, err := desktopcore.NewPreferenceStore(
		filepath.Join(root, "desktop-settings.json"),
	).Load()
	if err != nil {
		t.Fatal(err)
	}
	if persisted.WalletAutoLockMinutes != 5 {
		t.Fatalf("persisted auto-lock=%d want=5", persisted.WalletAutoLockMinutes)
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


func TestCopyReceiveAddressRejectsInvalidAddress(t *testing.T) {
	app := &App{}
	if err := app.CopyReceiveAddress("not-a-valdr-address"); !errors.Is(
		err,
		ErrInvalidReceiveAddress,
	) {
		t.Fatalf("error=%v want ErrInvalidReceiveAddress", err)
	}
}


func TestDesktopNodeLogsRequireAdvancedModeAndRedactSecrets(t *testing.T) {
	prefs := desktopcore.DefaultDesktopPreferences()
	logs := desktopcore.NewLogBuffer(1024)
	app := &App{
		preferences: prefs,
		nodeLogs:    logs,
	}

	if _, err := app.GetNodeLogs(); !errors.Is(
		err,
		ErrDesktopAdvancedModeRequired,
	) {
		t.Fatalf("GetNodeLogs error=%v want ErrDesktopAdvancedModeRequired", err)
	}

	prefs.Advanced = true
	app.preferences = prefs
	if _, err := logs.Write([]byte(
		"2026/09/25 [NODE] started\nprivate_key=do-not-display\n",
	)); err != nil {
		t.Fatal(err)
	}

	got, err := app.GetNodeLogs()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "[NODE] started") {
		t.Fatalf("node log output missing safe line: %q", got)
	}
	if strings.Contains(got, "do-not-display") {
		t.Fatalf("node log output leaked secret: %q", got)
	}
	if !strings.Contains(got, "[REDACTED sensitive log line]") {
		t.Fatalf("node log output missing redaction marker: %q", got)
	}
}


func TestDesktopPublicNodeModeRequiresAdvancedAndPersists(t *testing.T) {
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
		NodeID:     "desktop-public-node-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	store := desktopcore.NewPreferenceStore(
		filepath.Join(root, "desktop-settings.json"),
	)
	prefs := desktopcore.DefaultDesktopPreferences()
	app := &App{
		paths:          paths,
		node:           node,
		preferenceStore: store,
		preferences:     prefs,
	}

	if _, err := app.SetPublicNodeMode(
		true,
		"node.valdr.example:17333",
	); !errors.Is(err, ErrDesktopAdvancedModeRequired) {
		t.Fatalf("SetPublicNodeMode error=%v want ErrDesktopAdvancedModeRequired", err)
	}

	prefs.Advanced = true
	app.preferences = prefs
	got, err := app.SetPublicNodeMode(
		true,
		"node.valdr.example:17333",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !got.PublicNode ||
		got.PublicNodeAdvertiseAddress != "node.valdr.example:17333" {
		t.Fatalf("unexpected public-node preferences: %+v", got)
	}

	reloaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.PublicNode ||
		reloaded.PublicNodeAdvertiseAddress != "node.valdr.example:17333" {
		t.Fatalf("public-node preferences not persisted: %+v", reloaded)
	}
	if node.Endpoint() != "http://127.0.0.1:17332" {
		t.Fatalf("public-node mode changed RPC boundary: %s", node.Endpoint())
	}
}
