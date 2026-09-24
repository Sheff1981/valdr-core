package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	desktopcore "github.com/Sheff1981/valdr-core/desktop"
	"github.com/Sheff1981/valdr-core/rpc"
	"github.com/Sheff1981/valdr-core/wallet"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var ErrPrivateKeyExportConfirmation = errors.New(
	"private key export requires exact confirmation",
)

const privateKeyExportConfirmation = "EXPORT PRIVATE KEY"

type DesktopState struct {
	Network               string             `json:"network"`
	ChainID               string             `json:"chain_id"`
	MainnetEnabled        bool               `json:"mainnet_enabled"`
	Paths                 desktopcore.Paths  `json:"paths"`
	NodeRunning           bool               `json:"node_running"`
	NodeStatus            *rpc.StatusResult  `json:"node_status,omitempty"`
	NodeError             string             `json:"node_error,omitempty"`
	Wallets               []wallet.Metadata  `json:"wallets"`
	UnlockedWallets       []string           `json:"unlocked_wallets"`
	WalletAutoLockMinutes int                `json:"wallet_auto_lock_minutes"`
}

type App struct {
	ctx            context.Context
	paths          desktopcore.Paths
	node           *desktopcore.NodeManager
	walletStore     *wallet.Store
	walletService   *desktopcore.WalletService
	walletSessions  *desktopcore.WalletSessionManager
	historyService  *desktopcore.HistoryService

	mu        sync.Mutex
	nodeError string
}

func NewApp() (*App, error) {
	paths, err := desktopcore.DefaultPaths(config.NetworkTestnetV02)
	if err != nil {
		return nil, err
	}
	if err := paths.Ensure(); err != nil {
		return nil, err
	}

	node, err := desktopcore.NewNodeManager(desktopcore.NodeProcessConfig{
		BinaryPath: strings.TrimSpace(os.Getenv("VALDRD_PATH")),
		Network:    config.NetworkTestnetV02,
		DataDir:    paths.NodeData,
		NodeID:     "valdr-desktop",
		Seeds:      desktopSeedsFromEnv(),
	})
	if err != nil {
		return nil, err
	}
	rpcClient := rpc.NewClient(node.Endpoint())
	walletStore := wallet.NewStore(paths.Wallets)
	walletService, err := desktopcore.NewWalletService(
		walletStore,
		rpcClient,
	)
	if err != nil {
		return nil, err
	}
	walletSessions, err := desktopcore.NewWalletSessionManager(
		walletStore,
		desktopcore.DefaultWalletAutoLock,
	)
	if err != nil {
		return nil, err
	}
	historyService, err := desktopcore.NewHistoryService(
		rpcClient,
		filepath.Join(
			paths.Root,
			"index",
			paths.Network+".json",
		),
	)
	if err != nil {
		return nil, err
	}

	return &App{
		paths:          paths,
		node:           node,
		walletStore:    walletStore,
		walletService:  walletService,
		walletSessions: walletSessions,
		historyService: historyService,
	}, nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if err := a.node.Start(); err != nil {
		a.setNodeError(err.Error())
	}
}

func (a *App) shutdown(context.Context) {
	if a.walletSessions != nil {
		a.walletSessions.LockAll()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = a.node.Stop(ctx)
}

func (a *App) GetState() (DesktopState, error) {
	store := a.walletStore
	if store == nil {
		store = wallet.NewStore(a.paths.Wallets)
	}
	items, err := store.List()
	if err != nil {
		return DesktopState{}, err
	}

	state := DesktopState{
		Network:         config.NetworkTestnetV02,
		ChainID:         "valdr-testnet-1",
		MainnetEnabled:  false,
		Paths:           a.paths,
		NodeRunning:     a.node.Running(),
		NodeError:       a.getNodeError(),
		Wallets:         items,
		UnlockedWallets: []string{},
	}
	if a.walletSessions != nil {
		state.WalletAutoLockMinutes = int(
			a.walletSessions.Timeout() / time.Minute,
		)
		for _, item := range items {
			if a.walletSessions.IsUnlocked(item.Address) {
				state.UnlockedWallets = append(
					state.UnlockedWallets,
					item.Address,
				)
			}
		}
	}
	if !state.NodeRunning {
		if exitErr := a.node.LastExitError(); exitErr != nil {
			state.NodeError = exitErr.Error()
		}
	}

	if state.NodeRunning {
		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()
		status, statusErr := a.node.Status(ctx)
		if statusErr != nil {
			state.NodeError = statusErr.Error()
		} else {
			state.NodeStatus = &status
			state.NodeError = ""
		}
	}
	return state, nil
}

func (a *App) CreateWallet(
	name string,
	passphrase string,
) (wallet.Metadata, error) {
	secret := []byte(passphrase)
	defer clearSecret(secret)

	store := a.walletStore
	if store == nil {
		store = wallet.NewStore(a.paths.Wallets)
	}
	created, err := store.CreateEncrypted(
		strings.TrimSpace(name),
		secret,
	)
	if err != nil {
		return wallet.Metadata{}, err
	}
	meta := created.Metadata()
	if a.walletSessions != nil {
		if _, err := a.walletSessions.Unlock(meta.Address, secret); err != nil {
			return wallet.Metadata{}, err
		}
	}
	return meta, nil
}

func (a *App) UnlockWallet(
	selector string,
	passphrase string,
) (wallet.Metadata, error) {
	if a.walletSessions == nil {
		return wallet.Metadata{}, desktopcore.ErrWalletLocked
	}
	secret := []byte(passphrase)
	defer clearSecret(secret)
	return a.walletSessions.Unlock(
		strings.TrimSpace(selector),
		secret,
	)
}

func (a *App) LockWallet(selector string) {
	if a.walletSessions == nil {
		return
	}
	a.walletSessions.Lock(strings.TrimSpace(selector))
}

func (a *App) SetWalletAutoLockMinutes(minutes int) error {
	if a.walletSessions == nil {
		return desktopcore.ErrWalletLocked
	}
	return a.walletSessions.SetTimeout(
		time.Duration(minutes) * time.Minute,
	)
}

func (a *App) GetReceiveQRCode(address string) (string, error) {
	return desktopcore.AddressQRCodeDataURI(
		strings.TrimSpace(address),
	)
}

func (a *App) ExportPrivateKey(
	selector string,
	confirmation string,
) (string, error) {
	if strings.TrimSpace(confirmation) != privateKeyExportConfirmation {
		return "", ErrPrivateKeyExportConfirmation
	}
	secret, err := a.walletPassphrase(selector)
	if err != nil {
		return "", err
	}
	defer clearSecret(secret)

	store := a.walletStore
	if store == nil {
		store = wallet.NewStore(a.paths.Wallets)
	}
	exported, err := store.Export(
		strings.TrimSpace(selector),
		secret,
	)
	if err != nil {
		return "", err
	}
	privateKey := exported.PrivateKey
	exported.PrivateKey = ""
	return privateKey, nil
}

func (a *App) GetWalletBalance(
	address string,
) (desktopcore.WalletBalance, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		2*time.Second,
	)
	defer cancel()
	return a.walletService.Balance(ctx, strings.TrimSpace(address))
}

func (a *App) GetTransactionHistory(
	address string,
) ([]desktopcore.TransactionHistoryItem, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()
	return a.historyService.History(
		ctx,
		strings.TrimSpace(address),
		desktopcore.DefaultHistoryLimit,
	)
}

func (a *App) PreviewSend(
	selector string,
	recipient string,
	amountVDR string,
) (desktopcore.SendPreview, error) {
	amount, err := desktopcore.ParseVDR(amountVDR)
	if err != nil || amount == 0 {
		return desktopcore.SendPreview{}, desktopcore.ErrInvalidVDRAmt
	}
	secret, err := a.walletPassphrase(selector)
	if err != nil {
		return desktopcore.SendPreview{}, err
	}
	defer clearSecret(secret)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()
	return a.walletService.PreviewSend(
		ctx,
		strings.TrimSpace(selector),
		secret,
		strings.TrimSpace(recipient),
		amount,
	)
}

func (a *App) SendTransaction(
	selector string,
	recipient string,
	amountVDR string,
) (desktopcore.SendResult, error) {
	amount, err := desktopcore.ParseVDR(amountVDR)
	if err != nil || amount == 0 {
		return desktopcore.SendResult{}, desktopcore.ErrInvalidVDRAmt
	}
	secret, err := a.walletPassphrase(selector)
	if err != nil {
		return desktopcore.SendResult{}, err
	}
	defer clearSecret(secret)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		8*time.Second,
	)
	defer cancel()
	return a.walletService.Send(
		ctx,
		strings.TrimSpace(selector),
		secret,
		strings.TrimSpace(recipient),
		amount,
	)
}

func (a *App) BackupWallet(selector string) (string, error) {
	if a.ctx == nil {
		return "", errors.New("VALDR Desktop is not started")
	}
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return "", wallet.ErrWalletNotFound
	}
	path, err := wailsruntime.SaveFileDialog(
		a.ctx,
		wailsruntime.SaveDialogOptions{
			Title:           "Back up encrypted VALDR wallet",
			DefaultFilename: "VALDR-" + selector + ".valdr-wallet",
			Filters: []wailsruntime.FileFilter{{
				DisplayName: "VALDR encrypted wallet (*.valdr-wallet)",
				Pattern:     "*.valdr-wallet",
			}},
		},
	)
	if err != nil || path == "" {
		return path, err
	}
	if filepath.Ext(path) == "" {
		path += ".valdr-wallet"
	}
	if err := wallet.NewStore(a.paths.Wallets).
		BackupEncrypted(selector, path); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) RestoreWallet() (wallet.Metadata, error) {
	if a.ctx == nil {
		return wallet.Metadata{}, errors.New("VALDR Desktop is not started")
	}
	path, err := wailsruntime.OpenFileDialog(
		a.ctx,
		wailsruntime.OpenDialogOptions{
			Title: "Restore encrypted VALDR wallet",
			Filters: []wailsruntime.FileFilter{{
				DisplayName: "VALDR encrypted wallet",
				Pattern:     "*.valdr-wallet;*.json",
			}},
		},
	)
	if err != nil || path == "" {
		return wallet.Metadata{}, err
	}
	return wallet.NewStore(a.paths.Wallets).ImportEncrypted(path)
}

func (a *App) StartNode() error {
	if err := a.node.Start(); err != nil {
		a.setNodeError(err.Error())
		return err
	}
	a.setNodeError("")
	return nil
}

func (a *App) StopNode() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.node.Stop(ctx); err != nil {
		a.setNodeError(err.Error())
		return err
	}
	a.setNodeError("")
	return nil
}

func (a *App) walletPassphrase(selector string) ([]byte, error) {
	if a.walletSessions == nil {
		return nil, desktopcore.ErrWalletLocked
	}
	return a.walletSessions.Passphrase(strings.TrimSpace(selector))
}

func (a *App) setNodeError(value string) {
	a.mu.Lock()
	a.nodeError = value
	a.mu.Unlock()
}

func (a *App) getNodeError() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.nodeError
}

func desktopSeedsFromEnv() []string {
	raw := strings.TrimSpace(os.Getenv("VALDR_DESKTOP_SEEDS"))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func clearSecret(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
