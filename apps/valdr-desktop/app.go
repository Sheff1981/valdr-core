package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
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
	ErrDesktopExplorerUnavailable = errors.New(
		"VALDR Explorer is not available on the local Testnet endpoint",
	)
	ErrDesktopNodeDataFirstRunOnly = errors.New(
		"node data directory can only be changed before the first wallet is created",
	)
)

const (
	privateKeyExportConfirmation = "EXPORT PRIVATE KEY"
	localExplorerURL             = "http://127.0.0.1:8080"
)

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
	InitializationReady   bool               `json:"initialization_ready"`
}

type DesktopExplorerStatus struct {
	Available bool   `json:"available"`
	URL       string `json:"url"`
	ChainID   string `json:"chain_id,omitempty"`
	Height    uint64 `json:"height,omitempty"`
	TipHash   string `json:"tip_hash,omitempty"`
	Message   string `json:"message,omitempty"`
}

type explorerHealthResponse struct {
	Status  string `json:"status"`
	ChainID string `json:"chain_id"`
	Height  uint64 `json:"height"`
	TipHash string `json:"tip_hash"`
}

type DesktopDiagnosticsPreferences struct {
	Language                   string `json:"language"`
	Theme                      string `json:"theme"`
	StartNode                  bool   `json:"start_node"`
	Advanced                   bool   `json:"advanced"`
	WalletAutoLockMinutes      int    `json:"wallet_auto_lock_minutes"`
	PublicNode                 bool   `json:"public_node"`
	PublicNodeAdvertiseAddress string `json:"public_node_advertise_address,omitempty"`
	NodeDataDirectory          string `json:"node_data_directory,omitempty"`
}

type DesktopDiagnosticsMining struct {
	Running                bool    `json:"running"`
	AcceptedBlocks         uint64  `json:"accepted_blocks"`
	LastError              string  `json:"last_error,omitempty"`
	Height                 uint64  `json:"height"`
	NextHeight             uint64  `json:"next_height"`
	CurrentTarget          string  `json:"current_target,omitempty"`
	TargetBlockTimeSeconds int64   `json:"target_block_time_seconds"`
	RetargetInterval       uint64  `json:"retarget_interval"`
	BlocksUntilRetarget    uint64  `json:"blocks_until_retarget"`
	HashrateHPS            float64 `json:"hashrate_hps"`
	LastBlockHashrateHPS   float64 `json:"last_block_hashrate_hps"`
	LastBlockHashes        uint64  `json:"last_block_hashes"`
	LastBlockDurationMS    float64 `json:"last_block_duration_ms"`
	TotalHashes            uint64  `json:"total_hashes"`
	TotalMiningDurationMS  float64 `json:"total_mining_duration_ms"`
}

type DesktopDiagnosticsReport struct {
	GeneratedAtUTC      string                        `json:"generated_at_utc"`
	Version             string                        `json:"version"`
	Network             string                        `json:"network"`
	ChainID             string                        `json:"chain_id"`
	MainnetEnabled      bool                          `json:"mainnet_enabled"`
	NodeRunning         bool                          `json:"node_running"`
	NodeError           string                        `json:"node_error,omitempty"`
	NodeStatus          *rpc.StatusResult             `json:"node_status,omitempty"`
	Storage             desktopcore.StorageDiagnostics `json:"storage"`
	Preferences         DesktopDiagnosticsPreferences `json:"preferences"`
	WalletCount         int                           `json:"wallet_count"`
	UnlockedWalletCount int                           `json:"unlocked_wallet_count"`
	Mining              *DesktopDiagnosticsMining     `json:"mining,omitempty"`
	MiningStatusError   string                        `json:"mining_status_error,omitempty"`
	NodeLogs            string                        `json:"node_logs,omitempty"`
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

	preferenceStore := desktopcore.NewPreferenceStore(
		filepath.Join(paths.Root, "desktop-settings.json"),
	)
	preferences, err := preferenceStore.Load()
	if err != nil {
		return nil, err
	}
	if preferences.NodeDataDirectory != "" {
		nodeData, err := desktopcore.NormalizeNodeDataDirectory(
			preferences.NodeDataDirectory,
		)
		if err != nil {
			return nil, err
		}
		paths.NodeData = nodeData
	}
	if err := paths.Ensure(); err != nil {
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
	paths := a.pathsSnapshot()
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
		Paths:           paths,
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
	state.InitializationReady = desktopInitializationReady(
		len(state.Wallets),
		state.NodeRunning,
		state.NodeStatus,
	)
	return state, nil
}

func desktopInitializationReady(
	walletCount int,
	nodeRunning bool,
	status *rpc.StatusResult,
) bool {
	return walletCount > 0 && nodeRunning && status != nil
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

func (a *App) ChooseFirstRunNodeDataDirectory() (string, error) {
	if a.ctx == nil {
		return "", errors.New("VALDR Desktop is not started")
	}
	current := a.pathsSnapshot().NodeData
	selected, err := wailsruntime.OpenDirectoryDialog(
		a.ctx,
		wailsruntime.OpenDialogOptions{
			Title:            "Choose VALDR node data directory",
			DefaultDirectory: current,
		},
	)
	if err != nil || strings.TrimSpace(selected) == "" {
		return current, err
	}
	return a.setFirstRunNodeDataDirectory(selected)
}

func (a *App) setFirstRunNodeDataDirectory(path string) (string, error) {
	currentPaths := a.pathsSnapshot()
	store := a.walletStore
	if store == nil {
		store = wallet.NewStore(currentPaths.Wallets)
	}
	wallets, err := store.List()
	if err != nil {
		return "", err
	}
	if len(wallets) != 0 {
		return "", ErrDesktopNodeDataFirstRunOnly
	}

	clean, err := desktopcore.PrepareNodeDataDirectory(path)
	if err != nil {
		return "", err
	}
	if clean == currentPaths.NodeData {
		return clean, nil
	}
	if a.node == nil {
		return "", errors.New("VALDR Desktop node manager is unavailable")
	}

	wasRunning := a.node.Running()
	if wasRunning {
		if err := a.StopNode(); err != nil {
			return "", err
		}
	}
	rollback := func() {
		_ = a.node.ConfigureDataDir(currentPaths.NodeData)
		if wasRunning {
			_ = a.StartNode()
		}
	}

	if err := a.node.ConfigureDataDir(clean); err != nil {
		rollback()
		return "", err
	}

	prefs := a.preferencesSnapshot()
	prefs.NodeDataDirectory = clean
	preferenceStore := a.preferenceStore
	if preferenceStore == nil {
		preferenceStore = desktopcore.NewPreferenceStore(
			filepath.Join(currentPaths.Root, "desktop-settings.json"),
		)
	}
	if err := preferenceStore.Save(prefs); err != nil {
		rollback()
		return "", err
	}

	a.mu.Lock()
	a.paths.NodeData = clean
	a.preferenceStore = preferenceStore
	a.preferences = prefs
	a.mu.Unlock()

	if wasRunning || prefs.StartNode {
		if err := a.StartNode(); err != nil {
			return clean, err
		}
	}
	return clean, nil
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

func (a *App) GetStorageDiagnostics() (desktopcore.StorageDiagnostics, error) {
	if !a.preferencesSnapshot().Advanced {
		return desktopcore.StorageDiagnostics{}, ErrDesktopAdvancedModeRequired
	}
	return desktopcore.InspectStorage(
		a.pathsSnapshot().NodeData,
		desktopcore.DefaultStorageDiagnosticsMaxEntries,
	), nil
}

func (a *App) GetExplorerStatus() (DesktopExplorerStatus, error) {
	if !a.preferencesSnapshot().Advanced {
		return DesktopExplorerStatus{}, ErrDesktopAdvancedModeRequired
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	return probeDesktopExplorer(ctx, localExplorerURL)
}

func (a *App) OpenExplorer() error {
	if !a.preferencesSnapshot().Advanced {
		return ErrDesktopAdvancedModeRequired
	}
	status, err := a.GetExplorerStatus()
	if err != nil {
		return err
	}
	if !status.Available {
		return ErrDesktopExplorerUnavailable
	}
	if a.ctx == nil {
		return errors.New("VALDR Desktop is not started")
	}
	wailsruntime.BrowserOpenURL(a.ctx, localExplorerURL+"/")
	return nil
}

func probeDesktopExplorer(
	ctx context.Context,
	baseURL string,
) (DesktopExplorerStatus, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	status := DesktopExplorerStatus{URL: baseURL}

	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return status, ErrDesktopExplorerUnavailable
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return status, ErrDesktopExplorerUnavailable
	}
	if !strings.EqualFold(host, "localhost") {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return status, ErrDesktopExplorerUnavailable
		}
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		baseURL+"/healthz",
		nil,
	)
	if err != nil {
		return status, err
	}
	client := &http.Client{
		Timeout: 1500 * time.Millisecond,
		CheckRedirect: func(
			_ *http.Request,
			_ []*http.Request,
		) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Do(request)
	if err != nil {
		status.Message = "Local Explorer is not running."
		return status, nil
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		status.Message = fmt.Sprintf(
			"Local Explorer health returned HTTP %d.",
			response.StatusCode,
		)
		return status, nil
	}

	var health explorerHealthResponse
	if err := json.NewDecoder(
		io.LimitReader(response.Body, 64*1024),
	).Decode(&health); err != nil {
		status.Message = "Local Explorer health response is invalid."
		return status, nil
	}
	if health.Status != "ok" ||
		health.ChainID != "valdr-testnet-1" {
		status.Message = "Local Explorer is not connected to VALDR Testnet."
		return status, nil
	}

	status.Available = true
	status.ChainID = health.ChainID
	status.Height = health.Height
	status.TipHash = health.TipHash
	status.Message = "Local VALDR Explorer is ready."
	return status, nil
}

func (a *App) ExportDiagnostics() (string, error) {
	if !a.preferencesSnapshot().Advanced {
		return "", ErrDesktopAdvancedModeRequired
	}
	if a.ctx == nil {
		return "", errors.New("VALDR Desktop is not started")
	}

	report, err := a.desktopDiagnosticsJSON()
	if err != nil {
		return "", err
	}

	path, err := wailsruntime.SaveFileDialog(
		a.ctx,
		wailsruntime.SaveDialogOptions{
			Title: "Export VALDR diagnostics",
			DefaultFilename: "VALDR-diagnostics-" +
				time.Now().UTC().Format("20060102-150405") + ".json",
			Filters: []wailsruntime.FileFilter{{
				DisplayName: "JSON diagnostics (*.json)",
				Pattern:     "*.json",
			}},
		},
	)
	if err != nil || path == "" {
		return path, err
	}
	if filepath.Ext(path) == "" {
		path += ".json"
	}
	if err := writeDesktopDiagnostics(path, report); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) desktopDiagnosticsJSON() ([]byte, error) {
	if !a.preferencesSnapshot().Advanced {
		return nil, ErrDesktopAdvancedModeRequired
	}

	state, err := a.GetState()
	if err != nil {
		return nil, err
	}
	prefs := state.Preferences
	report := DesktopDiagnosticsReport{
		GeneratedAtUTC: time.Now().UTC().Format(time.RFC3339),
		Version:        config.Version,
		Network:        state.Network,
		ChainID:        state.ChainID,
		MainnetEnabled: state.MainnetEnabled,
		NodeRunning:    state.NodeRunning,
		NodeError:      state.NodeError,
		NodeStatus:     state.NodeStatus,
		Storage: desktopcore.InspectStorage(
			state.Paths.NodeData,
			desktopcore.DefaultStorageDiagnosticsMaxEntries,
		),
		Preferences: DesktopDiagnosticsPreferences{
			Language:                   prefs.Language,
			Theme:                      prefs.Theme,
			StartNode:                  prefs.StartNode,
			Advanced:                   prefs.Advanced,
			WalletAutoLockMinutes:      prefs.WalletAutoLockMinutes,
			PublicNode:                 prefs.PublicNode,
			PublicNodeAdvertiseAddress: prefs.PublicNodeAdvertiseAddress,
			NodeDataDirectory:          prefs.NodeDataDirectory,
		},
		WalletCount:         len(state.Wallets),
		UnlockedWalletCount: len(state.UnlockedWallets),
	}
	if a.nodeLogs != nil {
		report.NodeLogs = a.nodeLogs.String()
	}
	if a.miner != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		mining, miningErr := a.miner.Status(ctx)
		cancel()
		report.Mining = &DesktopDiagnosticsMining{
			Running:                mining.Running,
			AcceptedBlocks:         mining.AcceptedBlocks,
			LastError:              mining.LastError,
			Height:                 mining.Height,
			NextHeight:             mining.NextHeight,
			CurrentTarget:          mining.CurrentTarget,
			TargetBlockTimeSeconds: mining.TargetBlockTimeSeconds,
			RetargetInterval:       mining.RetargetInterval,
			BlocksUntilRetarget:    mining.BlocksUntilRetarget,
			HashrateHPS:            mining.HashrateHPS,
			LastBlockHashrateHPS:   mining.LastBlockHashrateHPS,
			LastBlockHashes:        mining.LastBlockHashes,
			LastBlockDurationMS:    mining.LastBlockDurationMS,
			TotalHashes:            mining.TotalHashes,
			TotalMiningDurationMS:  mining.TotalMiningDurationMS,
		}
		if miningErr != nil {
			report.MiningStatusError = miningErr.Error()
		}
	}

	return json.MarshalIndent(report, "", "  ")
}

func writeDesktopDiagnostics(path string, report []byte) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("diagnostics path is required")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".valdr-diagnostics-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}
	if err := tmp.Chmod(0o600); err != nil {
		cleanup()
		return err
	}
	if _, err := tmp.Write(report); err != nil {
		cleanup()
		return err
	}
	if _, err := tmp.Write([]byte("\n")); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Chmod(path, 0o600)
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

func (a *App) pathsSnapshot() desktopcore.Paths {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.paths
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
