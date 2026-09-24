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

type DesktopState struct {
	Network        string             `json:"network"`
	ChainID        string             `json:"chain_id"`
	MainnetEnabled bool               `json:"mainnet_enabled"`
	Paths          desktopcore.Paths  `json:"paths"`
	NodeRunning    bool               `json:"node_running"`
	NodeStatus     *rpc.StatusResult  `json:"node_status,omitempty"`
	NodeError      string             `json:"node_error,omitempty"`
	Wallets        []wallet.Metadata  `json:"wallets"`
}

type App struct {
	ctx           context.Context
	paths         desktopcore.Paths
	node          *desktopcore.NodeManager
	walletService  *desktopcore.WalletService
	historyService *desktopcore.HistoryService

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
	walletService, err := desktopcore.NewWalletService(
		wallet.NewStore(paths.Wallets),
		rpcClient,
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
		walletService:  walletService,
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = a.node.Stop(ctx)
}

func (a *App) GetState() (DesktopState, error) {
	items, err := wallet.NewStore(a.paths.Wallets).List()
	if err != nil {
		return DesktopState{}, err
	}

	state := DesktopState{
		Network:        config.NetworkTestnetV02,
		ChainID:        "valdr-testnet-1",
		MainnetEnabled: false,
		Paths:          a.paths,
		NodeRunning:    a.node.Running(),
		NodeError:      a.getNodeError(),
		Wallets:        items,
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

	created, err := wallet.NewStore(a.paths.Wallets).
		CreateEncrypted(strings.TrimSpace(name), secret)
	if err != nil {
		return wallet.Metadata{}, err
	}
	return created.Metadata(), nil
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
	passphrase string,
	recipient string,
	amountVDR string,
) (desktopcore.SendPreview, error) {
	amount, err := desktopcore.ParseVDR(amountVDR)
	if err != nil || amount == 0 {
		return desktopcore.SendPreview{}, desktopcore.ErrInvalidVDRAmt
	}
	secret := []byte(passphrase)
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
	passphrase string,
	recipient string,
	amountVDR string,
) (desktopcore.SendResult, error) {
	amount, err := desktopcore.ParseVDR(amountVDR)
	if err != nil || amount == 0 {
		return desktopcore.SendResult{}, desktopcore.ErrInvalidVDRAmt
	}
	secret := []byte(passphrase)
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
