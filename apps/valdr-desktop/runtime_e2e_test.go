package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	desktopcore "github.com/Sheff1981/valdr-core/desktop"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestDesktopRuntimeWalletSendReceiveHistory(t *testing.T) {
	if os.Getenv("VALDR_DESKTOP_RUNTIME_E2E") != "1" {
		t.Skip("set VALDR_DESKTOP_RUNTIME_E2E=1 to run live Desktop runtime integration")
	}
	for _, key := range []string{"VALDRD_PATH", "VALDR_MINER_PATH"} {
		path := strings.TrimSpace(os.Getenv(key))
		if path == "" {
			t.Fatalf("%s is required", key)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s=%q: %v", key, path, err)
		}
	}

	root := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("LOCALAPPDATA", filepath.Join(root, "localappdata"))
	case "darwin":
		t.Setenv("HOME", filepath.Join(root, "home"))
	case "linux":
		t.Setenv("HOME", filepath.Join(root, "home"))
		t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	default:
		t.Fatalf("unsupported runtime E2E platform %s", runtime.GOOS)
	}
	t.Setenv("VALDR_DESKTOP_SEEDS", "")

	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	app.startup(context.Background())
	t.Cleanup(func() {
		_ = app.StopMining()
		app.shutdown(context.Background())
	})

	initial := waitForRuntimeNode(t, app, 30*time.Second)
	if initial.NodeStatus == nil {
		t.Fatal("Desktop node status unavailable")
	}
	if initial.NodeStatus.Network != config.NetworkTestnetV02 ||
		initial.NodeStatus.ChainID != "valdr-testnet-1" {
		t.Fatalf("unexpected Desktop Testnet identity: %+v", initial.NodeStatus)
	}

	sourcePassphrase := "runtime-source-passphrase"
	recipientPassphrase := "runtime-recipient-passphrase"
	source, err := app.CreateWallet("runtime-source", sourcePassphrase)
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := app.CreateWallet("runtime-recipient", recipientPassphrase)
	if err != nil {
		t.Fatal(err)
	}

	qr, err := app.GetReceiveQRCode(recipient.Address)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(qr, "data:image/svg+xml;base64,") {
		t.Fatalf("unexpected receive QR data URI: %q", qr)
	}

	if _, err := app.SetDesktopPreferences(true, true); err != nil {
		t.Fatal(err)
	}

	fundingStart := initial.NodeStatus.Height
	if err := app.StartMining(source.Address); err != nil {
		t.Fatal(err)
	}
	funded := waitForRuntimeHeight(t, app, fundingStart+1, 30*time.Second)
	if err := app.StopMining(); err != nil {
		t.Fatal(err)
	}

	sourceBalance, err := app.GetWalletBalance(source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if sourceBalance.BalanceVal < config.InitialMiningReward {
		t.Fatalf("source balance=%d want at least one block reward=%d", sourceBalance.BalanceVal, config.InitialMiningReward)
	}

	preview, err := app.PreviewSend(
		source.Address,
		recipient.Address,
		"1.00000000",
	)
	if err != nil {
		t.Fatal(err)
	}
	if preview.AmountVal != config.AtomicUnitsPerVDR ||
		preview.FeeVal == 0 ||
		preview.TotalVal != preview.AmountVal+preview.FeeVal {
		t.Fatalf("unexpected Desktop send preview: %+v", preview)
	}

	sent, err := app.SendTransaction(
		source.Address,
		recipient.Address,
		"1.00000000",
	)
	if err != nil {
		t.Fatal(err)
	}
	if sent.TransactionID == "" {
		t.Fatal("Desktop send returned empty transaction id")
	}

	sourcePending, err := app.GetTransactionHistory(source.Address)
	if err != nil {
		t.Fatal(err)
	}
	pendingSent := runtimeHistoryItem(sourcePending, sent.TransactionID)
	if pendingSent == nil ||
		pendingSent.Status != "pending" ||
		pendingSent.Direction != "sent" {
		t.Fatalf("source pending history missing sent transaction %s: %+v", sent.TransactionID, sourcePending)
	}

	recipientPending, err := app.GetTransactionHistory(recipient.Address)
	if err != nil {
		t.Fatal(err)
	}
	pendingReceived := runtimeHistoryItem(recipientPending, sent.TransactionID)
	if pendingReceived == nil ||
		pendingReceived.Status != "pending" ||
		pendingReceived.Direction != "received" ||
		pendingReceived.AmountVal != config.AtomicUnitsPerVDR {
		t.Fatalf("recipient pending history missing receive transaction %s: %+v", sent.TransactionID, recipientPending)
	}

	if err := app.StartMining(source.Address); err != nil {
		t.Fatal(err)
	}
	_ = waitForRuntimeHeight(t, app, funded.NodeStatus.Height+1, 30*time.Second)
	if err := app.StopMining(); err != nil {
		t.Fatal(err)
	}

	sourceConfirmed, err := app.GetTransactionHistory(source.Address)
	if err != nil {
		t.Fatal(err)
	}
	confirmedSent := runtimeHistoryItem(sourceConfirmed, sent.TransactionID)
	if confirmedSent == nil ||
		confirmedSent.Status != "confirmed" ||
		confirmedSent.Direction != "sent" ||
		confirmedSent.Confirmations < 1 {
		t.Fatalf("source confirmed history missing sent transaction %s: %+v", sent.TransactionID, sourceConfirmed)
	}

	recipientConfirmed, err := app.GetTransactionHistory(recipient.Address)
	if err != nil {
		t.Fatal(err)
	}
	confirmedReceived := runtimeHistoryItem(recipientConfirmed, sent.TransactionID)
	if confirmedReceived == nil ||
		confirmedReceived.Status != "confirmed" ||
		confirmedReceived.Direction != "received" ||
		confirmedReceived.AmountVal != config.AtomicUnitsPerVDR ||
		confirmedReceived.Confirmations < 1 {
		t.Fatalf("recipient confirmed history missing receive transaction %s: %+v", sent.TransactionID, recipientConfirmed)
	}

	recipientBalance, err := app.GetWalletBalance(recipient.Address)
	if err != nil {
		t.Fatal(err)
	}
	if recipientBalance.BalanceVal != config.AtomicUnitsPerVDR ||
		recipientBalance.BalanceVDR != "1.00000000" {
		t.Fatalf("unexpected recipient balance after confirmation: %+v", recipientBalance)
	}

	if _, err := app.UnlockWallet(recipient.Address, "wrong-runtime-passphrase"); err == nil {
		t.Fatal("wrong wallet passphrase unexpectedly succeeded")
	}
	if _, err := app.UnlockWallet(recipient.Address, recipientPassphrase); err != nil {
		t.Fatal(err)
	}

	backupPath := filepath.Join(root, "runtime-source.valdr-wallet")
	if err := app.backupWalletTo(source.Address, backupPath); err != nil {
		t.Fatal(err)
	}
	backupRaw, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(backupRaw, []byte(sourcePassphrase)) ||
		bytes.Contains(backupRaw, []byte("private_key")) {
		t.Fatal("encrypted Desktop backup contains plaintext secret material")
	}

	privateKey, err := app.ExportPrivateKey(
		source.Address,
		privateKeyExportConfirmation,
	)
	if err != nil {
		t.Fatal(err)
	}
	if privateKey == "" {
		t.Fatal("private key export returned empty value")
	}
	if bytes.Contains(backupRaw, []byte(privateKey)) {
		t.Fatal("encrypted Desktop backup contains plaintext private key")
	}

	restoreDir := filepath.Join(root, "restored-wallets")
	restoreStore := wallet.NewStore(restoreDir)
	restoreSessions, err := desktopcore.NewWalletSessionManager(
		restoreStore,
		desktopcore.DefaultWalletAutoLock,
	)
	if err != nil {
		t.Fatal(err)
	}
	restoreApp := &App{
		paths: desktopcore.Paths{
			Root:    filepath.Join(root, "restore-root"),
			Wallets: restoreDir,
			Network: config.NetworkTestnetV02,
		},
		walletStore:    restoreStore,
		walletSessions: restoreSessions,
	}

	if _, err := restoreApp.restoreWalletFrom(
		backupPath,
		"wrong-backup-passphrase",
	); !errors.Is(err, wallet.ErrWalletAuthentication) {
		t.Fatalf("wrong backup passphrase error=%v", err)
	}
	items, err := restoreStore.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("wrong-passphrase restore imported wallet: %+v", items)
	}

	restored, err := restoreApp.restoreWalletFrom(
		backupPath,
		sourcePassphrase,
	)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Address != source.Address {
		t.Fatalf("restored address=%s want=%s", restored.Address, source.Address)
	}
	if !restoreSessions.IsUnlocked(source.Address) {
		t.Fatal("restored Desktop wallet is not unlocked")
	}

	assertRuntimeSecretsAbsent(
		t,
		root,
		sourcePassphrase,
		recipientPassphrase,
		privateKey,
	)
	privateKey = ""
}

func waitForRuntimeNode(
	t *testing.T,
	app *App,
	timeout time.Duration,
) DesktopState {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last DesktopState
	for time.Now().Before(deadline) {
		state, err := app.GetState()
		if err == nil {
			last = state
			if state.NodeRunning && state.NodeStatus != nil {
				return state
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("Desktop node did not become ready within %s: %+v", timeout, last)
	return DesktopState{}
}

func waitForRuntimeHeight(
	t *testing.T,
	app *App,
	minimum uint64,
	timeout time.Duration,
) DesktopState {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last DesktopState
	for time.Now().Before(deadline) {
		state, err := app.GetState()
		if err == nil {
			last = state
			if state.NodeStatus != nil && state.NodeStatus.Height >= minimum {
				return state
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	mining, _ := app.GetMiningState()
	t.Fatalf(
		"Desktop node did not reach height %d within %s: state=%+v mining=%+v",
		minimum,
		timeout,
		last,
		mining,
	)
	return DesktopState{}
}

func runtimeHistoryItem(
	items []desktopcore.TransactionHistoryItem,
	transactionID string,
) *desktopcore.TransactionHistoryItem {
	for i := range items {
		if items[i].TransactionID == transactionID {
			return &items[i]
		}
	}
	return nil
}

func assertRuntimeSecretsAbsent(
	t *testing.T,
	root string,
	secrets ...string,
) {
	t.Helper()
	err := filepath.WalkDir(root, func(
		path string,
		entry os.DirEntry,
		err error,
	) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, secret := range secrets {
			if secret != "" && bytes.Contains(raw, []byte(secret)) {
				t.Fatalf("plaintext secret material persisted in %s", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
