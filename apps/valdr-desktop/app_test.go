package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
	desktopcore "github.com/Sheff1981/valdr-core/desktop"
	"github.com/Sheff1981/valdr-core/rpc"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestDesktopAppCreatesEncryptedWalletOnly(t *testing.T) {
	root := t.TempDir()
	paths := desktopcore.Paths{
		Root:     root,
		NodeData: filepath.Join(root, "node", "testnet2"),
		Wallets:  filepath.Join(root, "wallets"),
		Logs:     filepath.Join(root, "logs"),
		Network:  config.NetworkTestnetV029,
	}
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}

	node, err := desktopcore.NewNodeManager(desktopcore.NodeProcessConfig{
		BinaryPath: filepath.Join(root, "valdrd-not-started"),
		Network:    config.NetworkTestnetV029,
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
	if state.Network != config.NetworkTestnetV029 ||
		state.ChainID != "valdr-testnet-2" {
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
		NodeData: filepath.Join(root, "node", "testnet2"),
		Wallets:  filepath.Join(root, "wallets"),
		Logs:     filepath.Join(root, "logs"),
		Network:  config.NetworkTestnetV029,
	}
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}

	node, err := desktopcore.NewNodeManager(desktopcore.NodeProcessConfig{
		BinaryPath: filepath.Join(root, "valdrd-not-started"),
		Network:    config.NetworkTestnetV029,
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
		"node.example.com:17333",
	); !errors.Is(err, ErrDesktopAdvancedModeRequired) {
		t.Fatalf("SetPublicNodeMode error=%v want ErrDesktopAdvancedModeRequired", err)
	}

	prefs.Advanced = true
	app.preferences = prefs
	got, err := app.SetPublicNodeMode(
		true,
		"node.example.com:17333",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !got.PublicNode ||
		got.PublicNodeAdvertiseAddress != "node.example.com:17333" {
		t.Fatalf("unexpected public-node preferences: %+v", got)
	}

	reloaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.PublicNode ||
		reloaded.PublicNodeAdvertiseAddress != "node.example.com:17333" {
		t.Fatalf("public-node preferences not persisted: %+v", reloaded)
	}
	if node.Endpoint() != "http://127.0.0.1:17332" {
		t.Fatalf("public-node mode changed RPC boundary: %s", node.Endpoint())
	}
}


func TestDesktopStorageDiagnosticsRequireAdvancedMode(t *testing.T) {
	root := t.TempDir()
	nodeData := filepath.Join(root, "node", "testnet2")
	if err := os.MkdirAll(nodeData, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(nodeData, "sample.dat"),
		[]byte("VALDR"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	prefs := desktopcore.DefaultDesktopPreferences()
	app := &App{
		paths: desktopcore.Paths{
			Root:     root,
			NodeData: nodeData,
			Network:  config.NetworkTestnetV029,
		},
		preferences: prefs,
	}
	if _, err := app.GetStorageDiagnostics(); !errors.Is(
		err,
		ErrDesktopAdvancedModeRequired,
	) {
		t.Fatalf("GetStorageDiagnostics error=%v want Advanced required", err)
	}

	prefs.Advanced = true
	app.preferences = prefs
	got, err := app.GetStorageDiagnostics()
	if err != nil {
		t.Fatal(err)
	}
	if !got.Ready || got.FileCount != 1 || got.SizeBytes != 5 {
		t.Fatalf("unexpected storage diagnostics: %+v", got)
	}
}


func TestDesktopDiagnosticsRequireAdvancedAndExcludeSecrets(t *testing.T) {
	root := t.TempDir()
	paths := desktopcore.Paths{
		Root:     root,
		NodeData: filepath.Join(root, "node", "testnet2"),
		Wallets:  filepath.Join(root, "wallets"),
		Logs:     filepath.Join(root, "logs"),
		Network:  config.NetworkTestnetV029,
	}
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	node, err := desktopcore.NewNodeManager(desktopcore.NodeProcessConfig{
		BinaryPath: filepath.Join(root, "valdrd-not-started"),
		Network:    config.NetworkTestnetV029,
		DataDir:    paths.NodeData,
		NodeID:     "desktop-diagnostics-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	logs := desktopcore.NewLogBuffer(4096)
	_, _ = logs.Write([]byte(
		"[NODE] diagnostic line\nprivate_key=diagnostic-secret\npassword=another-secret\n",
	))

	prefs := desktopcore.DefaultDesktopPreferences()
	app := &App{
		paths:       paths,
		node:        node,
		walletStore: wallet.NewStore(paths.Wallets),
		nodeLogs:    logs,
		preferences: prefs,
	}

	if _, err := app.desktopDiagnosticsJSON(); !errors.Is(
		err,
		ErrDesktopAdvancedModeRequired,
	) {
		t.Fatalf("desktopDiagnosticsJSON error=%v want Advanced required", err)
	}

	prefs.Advanced = true
	app.preferences = prefs
	raw, err := app.desktopDiagnosticsJSON()
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, secret := range []string{
		"diagnostic-secret",
		"another-secret",
		"private_key=",
		"password=",
	} {
		if strings.Contains(text, secret) {
			t.Fatalf("diagnostics leaked secret marker %q: %s", secret, text)
		}
	}
	if !strings.Contains(text, "[REDACTED sensitive log line]") {
		t.Fatalf("diagnostics missing redaction marker: %s", text)
	}
	if strings.Contains(text, "private_key") ||
		strings.Contains(text, "passphrase") {
		t.Fatalf("diagnostics schema unexpectedly includes wallet secret fields: %s", text)
	}
	if !strings.Contains(text, `"network": "testnet2"`) ||
		!strings.Contains(text, `"chain_id": "valdr-testnet-2"`) ||
		!strings.Contains(text, `"mainnet_enabled": false`) {
		t.Fatalf("diagnostics missing Testnet identity: %s", text)
	}
}

func TestWriteDesktopDiagnosticsUsesPrivateFileMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "diagnostics.json")
	if err := writeDesktopDiagnostics(path, []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("diagnostics mode=%#o want=0600", info.Mode().Perm())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "{\"ok\":true}\n" {
		t.Fatalf("unexpected diagnostics file: %q", raw)
	}
}


func TestDesktopExplorerStatusRequiresAdvancedMode(t *testing.T) {
	prefs := desktopcore.DefaultDesktopPreferences()
	app := &App{preferences: prefs}
	if _, err := app.GetExplorerStatus(); !errors.Is(
		err,
		ErrDesktopAdvancedModeRequired,
	) {
		t.Fatalf("GetExplorerStatus error=%v want Advanced required", err)
	}
}

func TestProbeDesktopExplorerAcceptsLocalMatchingTestnet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"status":"ok","chain_id":"valdr-testnet-2","height":42,"tip_hash":"abc"}`,
		))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := probeDesktopExplorer(ctx, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Available ||
		got.ChainID != "valdr-testnet-2" ||
		got.Height != 42 ||
		got.TipHash != "abc" {
		t.Fatalf("unexpected Explorer status: %+v", got)
	}
}

func TestProbeDesktopExplorerRejectsWrongChainAndRemoteEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"status":"ok","chain_id":"valdr-devnet-2","height":7,"tip_hash":"def"}`,
		))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := probeDesktopExplorer(ctx, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got.Available {
		t.Fatalf("wrong-chain Explorer unexpectedly accepted: %+v", got)
	}

	if _, err := probeDesktopExplorer(
		context.Background(),
		"http://example.com:8080",
	); !errors.Is(err, ErrDesktopExplorerUnavailable) {
		t.Fatalf("remote Explorer endpoint error=%v want unavailable", err)
	}
}


func TestDesktopInitializationReadyRequiresWalletAndHealthyNode(t *testing.T) {
	status := &rpc.StatusResult{}
	if desktopInitializationReady(0, true, status) {
		t.Fatal("initialization ready without wallet")
	}
	if desktopInitializationReady(1, false, status) {
		t.Fatal("initialization ready while node is stopped")
	}
	if desktopInitializationReady(1, true, nil) {
		t.Fatal("initialization ready without healthy node status")
	}
	if !desktopInitializationReady(1, true, status) {
		t.Fatal("initialization not ready with wallet and healthy node")
	}
}


func TestDesktopFirstRunNodeDataDirectoryPersistsBeforeWalletOnly(t *testing.T) {
	root := t.TempDir()
	paths := desktopcore.Paths{
		Root:     root,
		NodeData: filepath.Join(root, "node", "testnet2"),
		Wallets:  filepath.Join(root, "wallets"),
		Logs:     filepath.Join(root, "logs"),
		Network:  config.NetworkTestnetV029,
	}
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	node, err := desktopcore.NewNodeManager(desktopcore.NodeProcessConfig{
		BinaryPath: filepath.Join(root, "valdrd-not-started"),
		Network:    config.NetworkTestnetV029,
		DataDir:    paths.NodeData,
		NodeID:     "desktop-data-dir-test",
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
	prefStore := desktopcore.NewPreferenceStore(
		filepath.Join(root, "desktop-settings.json"),
	)
	prefs := desktopcore.DefaultDesktopPreferences()
	prefs.StartNode = false
	app := &App{
		paths:           paths,
		node:            node,
		walletStore:     store,
		walletSessions:  sessions,
		preferenceStore: prefStore,
		preferences:     prefs,
	}

	custom := filepath.Join(root, "custom-chain-data")
	got, err := app.setFirstRunNodeDataDirectory(custom)
	if err != nil {
		t.Fatal(err)
	}
	want, err := desktopcore.NormalizeNodeDataDirectory(custom)
	if err != nil {
		t.Fatal(err)
	}
	if got != want || app.pathsSnapshot().NodeData != want {
		t.Fatalf("node data got=%q state=%q want=%q", got, app.pathsSnapshot().NodeData, want)
	}
	info, err := os.Stat(want)
	if err != nil || !info.IsDir() {
		t.Fatalf("custom node data directory not created: info=%v err=%v", info, err)
	}
	reloaded, err := prefStore.Load()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.NodeDataDirectory != want {
		t.Fatalf("persisted node data=%q want=%q", reloaded.NodeDataDirectory, want)
	}

	if _, err := app.CreateWallet("first-run-wallet", "first-run-passphrase"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.setFirstRunNodeDataDirectory(
		filepath.Join(root, "second-chain-data"),
	); !errors.Is(err, ErrDesktopNodeDataFirstRunOnly) {
		t.Fatalf("post-wallet data-dir error=%v want first-run-only", err)
	}
}

func TestNormalizeNodeDataDirectoryRejectsRootAndRelativePath(t *testing.T) {
	if _, err := desktopcore.NormalizeNodeDataDirectory("relative/path"); !errors.Is(
		err,
		desktopcore.ErrDesktopPath,
	) {
		t.Fatalf("relative path error=%v want ErrDesktopPath", err)
	}
	root := filepath.VolumeName(t.TempDir()) + string(os.PathSeparator)
	if _, err := desktopcore.NormalizeNodeDataDirectory(root); !errors.Is(
		err,
		desktopcore.ErrDesktopPath,
	) {
		t.Fatalf("root path error=%v want ErrDesktopPath", err)
	}
}


func TestDesktopFrontendDoesNotOfferMainnet(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("frontend", "src", "main.ts"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)

	for _, forbidden := range []string{
		`<option value="mainnet"`,
		`value="mainnet"`,
		`data-network="mainnet"`,
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("Desktop frontend exposes Mainnet selector/control %q", forbidden)
		}
	}
	for _, required := range []string{
		"Testnet · valdr-testnet-2",
		"Mainnet is disabled in this build",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("Desktop frontend missing locked Testnet/Mainnet-disabled marker %q", required)
		}
	}
}


func TestDesktopFrontendSecretPresentationIsEphemeral(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("frontend", "src", "main.ts"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	for _, required := range []string{
		`if (view !== "wallet") clearPrivateKeyExport();`,
		`document.addEventListener("visibilitychange"`,
		`if (document.hidden) {`,
		`clearPrivateKeyExport();`,
		`finally {`,
		`document.getElementById("unlock-wallet-passphrase")`,
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("Desktop frontend missing secret-lifecycle guard %q", required)
		}
	}
}


func TestDesktopFrontendStage12ScreenContract(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("frontend", "src", "main.ts"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)

	required := map[string][]string{
		"overview": {
			`id="overview-balance"`,
			`id="overview-wallet-label"`,
			`id="network-name"`,
			`id="height"`,
			`id="sync-progress"`,
			`id="peers"`,
			`id="overview-latest-direction"`,
			`id="overview-latest-meta"`,
		},
		"send": {
			`id="send-recipient"`,
			`id="send-amount"`,
			`id="preview-fee"`,
			`id="preview-total"`,
			`id="send-confirm-dialog"`,
			"transactions are irreversible",
		},
		"receive": {
			`id="receive-address"`,
			`id="copy-address"`,
			`id="receive-qr"`,
			"Generated locally · no web service",
		},
		"transactions": {
			`id="history-list"`,
			`id="transaction-detail-dialog"`,
			`id="transaction-detail-status"`,
			`id="transaction-detail-txid"`,
			`id="transaction-detail-confirmations"`,
		},
		"wallet": {
			`id="unlock-wallet-form"`,
			`id="lock-wallet"`,
			`id="backup-wallet"`,
			`id="restore-wallet-form"`,
			`id="show-export-warning"`,
			`id="confirm-private-key-export"`,
			"High risk:",
		},
		"network": {
			`id="detail-sync"`,
			`id="detail-blocks-remaining"`,
			`id="detail-last-block-time"`,
			`id="detail-sync-eta"`,
			`id="detail-peer-count"`,
			`id="detail-tip"`,
			`id="detail-chainwork"`,
			`id="detail-data"`,
			`id="detail-storage-state"`,
			`id="public-node-enabled"`,
		},
		"mining": {
			`id="start-mining"`,
			`id="stop-mining"`,
			`id="mining-reward-address"`,
			`id="mining-accepted"`,
			`id="mining-hashrate"`,
			`id="mining-last-hashrate"`,
			"Mining never starts automatically.",
		},
		"settings": {
			`id="settings-language"`,
			`id="settings-theme"`,
			`id="settings-start-node"`,
			`id="settings-advanced"`,
			`id="settings-node-data"`,
			`id="export-diagnostics"`,
			"Testnet · valdr-testnet-2",
			"Mainnet is disabled in this build",
		},
	}
	for screen, markers := range required {
		for _, marker := range markers {
			if !strings.Contains(source, marker) {
				t.Fatalf("Stage 12 %s UI missing required marker %q", screen, marker)
			}
		}
	}

	for _, advancedNav := range []string{
		`class="nav-item advanced-only hidden" data-view="mining"`,
		`class="nav-item advanced-only hidden" data-view="network"`,
	} {
		if !strings.Contains(source, advancedNav) {
			t.Fatalf("Advanced-only Desktop navigation contract missing %q", advancedNav)
		}
	}
}

func TestDesktopFrontendStage12B1SyncDetailContract(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("frontend", "src", "main.ts"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	for _, marker := range []string{
		`id="sync-warning"`,
		"Synchronization is incomplete.",
		"best_known_height - status.height",
		`"detail-blocks-remaining"`,
		`"detail-last-block-time"`,
		`"detail-sync-eta"`,
		`status.peer_count === 0 ? "Unknown" : "Calculating…"`,
	} {
		if !strings.Contains(source, marker) {
			t.Fatalf("Stage 12B.1 sync detail missing %q", marker)
		}
	}
}


func TestDesktopFrontendStage12B2BalanceSemanticsContract(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("frontend", "src", "main.ts"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	for _, marker := range []string{
		"SPENDABLE BALANCE",
		`id="overview-pending"`,
		`id="overview-total"`,
		"balance.spendable_vdr",
		"balance.pending_vdr",
		"balance.total_vdr",
	} {
		if !strings.Contains(source, marker) {
			t.Fatalf("Stage 12B.2 balance semantics missing %q", marker)
		}
	}
	if strings.Contains(source, "Immature mining reward") {
		t.Fatal("Stage 12B.2 must not expose immature balance before a maturity rule exists")
	}
}
