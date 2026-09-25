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
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
	desktopcore "github.com/Sheff1981/valdr-core/desktop"
	"github.com/Sheff1981/valdr-core/rpc"
	"github.com/Sheff1981/valdr-core/wallet"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	ErrPrivateKeyExportConfirmation = errors.New(
		"private key export requires exact confirmation",
	)
	ErrDesktopAdvancedModeRequired = errors.New(
		"VALDR Desktop Advanced mode is required",
	)
	ErrDesktopNodeRequired = errors.New(
		"VALDR Desktop local node must be running",
	)
	ErrInvalidReceiveAddress = errors.New(
		"invalid VALDR receive address",
	)
	ErrDesktopPublicNodeDisableFirst = errors.New(
		"disable public-node mode before leaving Advanced mode",
	)
)

const privateKeyExportConfirmation = "EXPORT PRIVATE KEY"

type DesktopState struct {
	Network               string                         `json:"network"`
	ChainID               string                         `json:"chain_id"`
	MainnetEnabled        bool                           `json:"mainnet_enabled"`
	Paths                 desktopcore.Paths              `json:"paths"`
	Preferences           desktopcore.DesktopPreferences `json:"preferences"`
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
	miner          *desktopcore.MinerManager
	walletStore     *wallet.Store
	walletService   *desktopcore.WalletService
	walletSessions  *desktopcore.WalletSessionManager
	historyService  *desktopcore.HistoryService
	preferenceStore *desktopcore.PreferenceStore
	nodeLogs        *desktopcore.LogBuffer

	mu          sync.Mutex
	nodeError   string
	preferences desktopcore.DesktopPreferences
}

func NewApp() (*App, error) {
	paths, err := desktopcore.DefaultPaths(config.NetworkTestnetV02)
	if err != nil {
		return nil, err
	}
	if err := paths.Ensure(); err != nil {
		return nil, err
	}

	preferenceStore := desktopcore.NewPreferenceStore(
		filepath.Join(paths.Root, "desktop-settings.json"),
	)
	preferences, err := preferenceStore.Load()
	if err != nil {
		return nil, err
	}

	nodeLogs := desktopcore.NewLogBuffer(desktopcore.DefaultDesktopLogBytes)
	node, err := desktopcore.NewNodeManager(desktopcore.NodeProcessConfig{
		BinaryPath: strings.TrimSpace(os.Getenv("VALDRD_PATH")),
		Network:    config.NetworkTestnetV02,
		DataDir:    paths.NodeData,
		NodeID:     "valdr-desktop",
		Seeds:            desktopSeedsFromEnv(),
		PublicNode:       preferences.PublicNode,
		AdvertiseAddress: preferences.PublicNodeAdvertiseAddress,
		Stdout:           nodeLogs,
		Stderr:           nodeLogs,
	})
	if err != nil {
		return nil, err
	}
	miner, err := desktopcore.NewMinerManager(desktopcore.MinerProcessConfig{
		BinaryPath:   strings.TrimSpace(os.Getenv("VALDR_MINER_PATH")),
		NodeEndpoint: node.Endpoint(),
		PIDFile:      filepath.Join(paths.Root, "miner.pid"),
		Interval:     time.Second,
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
		time.Duration(preferences.WalletAutoLockMinutes)*time.Minute,
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
		miner:          miner,
		walletStore:    walletStore,
		walletService:  walletService,
		walletSessions:  walletSessions,
		historyService:  historyService,
		preferenceStore: preferenceStore,
		nodeLogs:        nodeLogs,
		preferences:     preferences,
	}, nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if !a.preferencesSnapshot().StartNode {
		return
	}
	if err := a.node.Start(); err != nil {
		a.setNodeError(err.Error())
	}
}

func (a *App) shutdown(context.Context) {
	if a.walletSessions != nil {
		a.walletSessions.LockAll()
	}
	if a.miner != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = a.miner.Stop(ctx)
		cancel()
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
		Preferences:     a.preferencesSnapshot(),
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

	timeout := time.Duration(minutes) * time.Minute
	previous := a.walletSessions.Timeout()
	if err := a.walletSessions.SetTimeout(timeout); err != nil {
		return err
	}

	prefs := a.preferencesSnapshot()
	prefs.WalletAutoLockMinutes = minutes
	store := a.preferenceStore
	if store == nil {
		store = desktopcore.NewPreferenceStore(
			filepath.Join(a.paths.Root, "desktop-settings.json"),
		)
	}
	if err := store.Save(prefs); err != nil {
		_ = a.walletSessions.SetTimeout(previous)
		return err
	}

	a.mu.Lock()
	a.preferenceStore = store
	a.preferences = prefs
	a.mu.Unlock()
	return nil
}

func (a *App) GetReceiveQRCode(address string) (string, error) {
	return desktopcore.AddressQRCodeDataURI(
		strings.TrimSpace(address),
	)
}

func (a *App) CopyReceiveAddress(address string) error {
	address = strings.TrimSpace(address)
	if !valdrcrypto.ValidateAddress(address) {
		return ErrInvalidReceiveAddress
	}
	if a.ctx == nil {
		return errors.New("VALDR Desktop is not started")
	}
	return wailsruntime.ClipboardSetText(a.ctx, address)
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

func (a *App) GetNodeLogs() (string, error) {
	if !a.preferencesSnapshot().Advanced {
		return "", ErrDesktopAdvancedModeRequired
	}
	if a.nodeLogs == nil {
		return "", nil
	}
	return a.nodeLogs.String(), nil
}

func (a *App) GetPeers() ([]rpc.PeerResult, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		2*time.Second,
	)
	defer cancel()

	var peers []rpc.PeerResult
	if err := rpc.NewClient(a.node.Endpoint()).Call(
		ctx,
		rpc.MethodGetPeers,
		nil,
		&peers,
	); err != nil {
		return nil, err
	}
	return peers, nil
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
	if err := a.backupWalletTo(selector, path); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) backupWalletTo(
	selector string,
	destination string,
) error {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return wallet.ErrWalletNotFound
	}
	store := a.walletStore
	if store == nil {
		store = wallet.NewStore(a.paths.Wallets)
	}
	return store.BackupEncrypted(selector, destination)
}

func (a *App) RestoreWallet(
	passphrase string,
) (wallet.Metadata, error) {
	if a.ctx == nil {
		return wallet.Metadata{}, errors.New("VALDR Desktop is not started")
	}
	if a.walletSessions == nil {
		return wallet.Metadata{}, desktopcore.ErrWalletLocked
	}
	if passphrase == "" {
		return wallet.Metadata{}, wallet.ErrPassphraseRequired
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
	return a.restoreWalletFrom(path, passphrase)
}

func (a *App) restoreWalletFrom(
	path string,
	passphrase string,
) (wallet.Metadata, error) {
	if a.walletSessions == nil {
		return wallet.Metadata{}, desktopcore.ErrWalletLocked
	}
	secret := []byte(passphrase)
	defer clearSecret(secret)
	if len(secret) == 0 {
		return wallet.Metadata{}, wallet.ErrPassphraseRequired
	}

	store := a.walletStore
	if store == nil {
		store = wallet.NewStore(a.paths.Wallets)
	}
	meta, err := store.ImportEncryptedVerified(path, secret)
	if err != nil {
		return wallet.Metadata{}, err
	}
	if _, err := a.walletSessions.Unlock(meta.Address, secret); err != nil {
		return wallet.Metadata{}, err
	}
	return meta, nil
}

func (a *App) SetDesktopPreferences(
	language string,
	theme string,
	startNode bool,
	advanced bool,
) (desktopcore.DesktopPreferences, error) {
	prefs := a.preferencesSnapshot()
	prefs.Language = strings.ToLower(strings.TrimSpace(language))
	prefs.Theme = strings.ToLower(strings.TrimSpace(theme))
	prefs.StartNode = startNode
	prefs.Advanced = advanced

	if !advanced && prefs.PublicNode {
		return desktopcore.DesktopPreferences{}, ErrDesktopPublicNodeDisableFirst
	}
	if !advanced && a.miner != nil && a.miner.Running() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := a.miner.Stop(ctx)
		cancel()
		if err != nil {
			return desktopcore.DesktopPreferences{}, err
		}
	}

	store := a.preferenceStore
	if store == nil {
		store = desktopcore.NewPreferenceStore(
			filepath.Join(a.paths.Root, "desktop-settings.json"),
		)
	}
	if err := store.Save(prefs); err != nil {
		return desktopcore.DesktopPreferences{}, err
	}

	a.mu.Lock()
	a.preferenceStore = store
	a.preferences = prefs
	a.mu.Unlock()
	return prefs, nil
}

func (a *App) SetPublicNodeMode(
	enabled bool,
	advertiseAddress string,
) (desktopcore.DesktopPreferences, error) {
	prefs := a.preferencesSnapshot()
	if !prefs.Advanced {
		return desktopcore.DesktopPreferences{}, ErrDesktopAdvancedModeRequired
	}

	advertiseAddress = strings.TrimSpace(advertiseAddress)
	if advertiseAddress != "" {
		if err := desktopcore.ValidatePublicNodeAdvertiseAddress(
			advertiseAddress,
		); err != nil {
			return desktopcore.DesktopPreferences{}, err
		}
	}
	if enabled && advertiseAddress == "" {
		return desktopcore.DesktopPreferences{}, desktopcore.ErrDesktopPublicNodeAddress
	}

	previousEnabled := prefs.PublicNode
	previousAddress := prefs.PublicNodeAdvertiseAddress
	changed := previousEnabled != enabled || previousAddress != advertiseAddress

	if err := a.node.ConfigurePublicNode(enabled, advertiseAddress); err != nil {
		return desktopcore.DesktopPreferences{}, err
	}
	prefs.PublicNode = enabled
	prefs.PublicNodeAdvertiseAddress = advertiseAddress

	store := a.preferenceStore
	if store == nil {
		store = desktopcore.NewPreferenceStore(
			filepath.Join(a.paths.Root, "desktop-settings.json"),
		)
	}
	if err := store.Save(prefs); err != nil {
		_ = a.node.ConfigurePublicNode(previousEnabled, previousAddress)
		return desktopcore.DesktopPreferences{}, err
	}

	a.mu.Lock()
	a.preferenceStore = store
	a.preferences = prefs
	a.mu.Unlock()

	if changed && a.node.Running() {
		if err := a.RestartNode(); err != nil {
			return prefs, err
		}
	}
	return prefs, nil
}

func (a *App) GetMiningState() (desktopcore.MinerStatus, error) {
	if a.miner == nil {
		return desktopcore.MinerStatus{}, desktopcore.ErrDesktopMinerConfig
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return a.miner.Status(ctx)
}

func (a *App) StartMining(rewardAddress string) error {
	if !a.preferencesSnapshot().Advanced {
		return ErrDesktopAdvancedModeRequired
	}
	if a.node == nil || !a.node.Running() {
		return ErrDesktopNodeRequired
	}
	if a.miner == nil {
		return desktopcore.ErrDesktopMinerConfig
	}
	return a.miner.Start(strings.TrimSpace(rewardAddress))
}

func (a *App) StopMining() error {
	if a.miner == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return a.miner.Stop(ctx)
}

func (a *App) StartNode() error {
	if err := a.node.Start(); err != nil {
		a.setNodeError(err.Error())
		return err
	}
	a.setNodeError("")
	return nil
}

func (a *App) RestartNode() error {
	if err := a.StopNode(); err != nil {
		return err
	}
	return a.StartNode()
}

func (a *App) StopNode() error {
	if a.miner != nil && a.miner.Running() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := a.miner.Stop(ctx)
		cancel()
		if err != nil {
			a.setNodeError(err.Error())
			return err
		}
	}
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

func (a *App) preferencesSnapshot() desktopcore.DesktopPreferences {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.preferences.Version != desktopcore.DesktopPreferencesVersion {
		return desktopcore.DefaultDesktopPreferences()
	}
	return a.preferences
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
