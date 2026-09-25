package desktop

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/rpc"
)

var (
	ErrNodeAlreadyRunning    = errors.New("VALDR Desktop node is already running")
	ErrNodeExitedUnexpectedly = errors.New("VALDR Desktop node exited unexpectedly")
	ErrDesktopMainnet        = errors.New("VALDR Desktop Mainnet is not enabled")
	ErrDesktopPublicNodeAddress = errors.New("invalid VALDR Desktop public-node advertise address")
)

type NodeProcessConfig struct {
	BinaryPath       string
	Network          string
	DataDir          string
	NodeID           string
	RPCPort          uint16
	Seeds            []string
	PublicNode       bool
	AdvertiseAddress string
	Stdout           io.Writer
	Stderr           io.Writer
}

type NodeManager struct {
	mu       sync.Mutex
	config   NodeProcessConfig
	command  *exec.Cmd
	stdin    io.WriteCloser
	waitCh   chan error
	stopping bool
	lastExit error
}

func NewNodeManager(cfg NodeProcessConfig) (*NodeManager, error) {
	if cfg.Network == "" {
		cfg.Network = config.NetworkTestnetV029
	}
	if cfg.Network != config.NetworkTestnetV029 &&
		cfg.Network != config.NetworkDevnetV02 {
		return nil, ErrDesktopMainnet
	}
	if _, err := config.ResolveNetworkProfile(cfg.Network); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.DataDir) == "" {
		return nil, errors.New("desktop node data directory is required")
	}
	if strings.TrimSpace(cfg.NodeID) == "" {
		cfg.NodeID = "valdr-desktop"
	}
	if cfg.RPCPort == 0 {
		profile, _ := config.ResolveNetworkProfile(cfg.Network)
		cfg.RPCPort = profile.RPCPort
	}
	cfg.AdvertiseAddress = strings.TrimSpace(cfg.AdvertiseAddress)
	if cfg.AdvertiseAddress != "" {
		if err := ValidatePublicNodeAdvertiseAddress(cfg.AdvertiseAddress); err != nil {
			return nil, err
		}
	}
	if cfg.PublicNode && cfg.AdvertiseAddress == "" {
		return nil, ErrDesktopPublicNodeAddress
	}
	if strings.TrimSpace(cfg.BinaryPath) == "" {
		binary, err := bundledBinaryPath("valdrd")
		if err != nil {
			return nil, err
		}
		cfg.BinaryPath = binary
	}
	if cfg.Stdout == nil {
		cfg.Stdout = io.Discard
	}
	if cfg.Stderr == nil {
		cfg.Stderr = io.Discard
	}
	return &NodeManager{config: cfg}, nil
}

func (m *NodeManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.command != nil {
		return ErrNodeAlreadyRunning
	}
	m.stopping = false
	m.lastExit = nil

	args, err := desktopNodeArgs(m.config)
	if err != nil {
		return err
	}
	cmd := exec.Command(m.config.BinaryPath, args...)
	cmd.Stdout = m.config.Stdout
	cmd.Stderr = m.config.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return err
	}

	waitCh := make(chan error, 1)
	m.command = cmd
	m.stdin = stdin
	m.waitCh = waitCh
	go func() {
		err := cmd.Wait()
		m.recordProcessExit(cmd, err)
		waitCh <- err
		close(waitCh)
	}()
	return nil
}

func (m *NodeManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	cmd := m.command
	stdin := m.stdin
	waitCh := m.waitCh
	if cmd != nil {
		m.stopping = true
	}
	m.mu.Unlock()
	if cmd == nil {
		return nil
	}

	if stdin != nil {
		_ = stdin.Close()
	}
	select {
	case err := <-waitCh:
		if err == nil {
			return nil
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil
		}
		return err
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		select {
		case <-waitCh:
		case <-time.After(time.Second):
		}
		return ctx.Err()
	}
}

func (m *NodeManager) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.command != nil
}

func (m *NodeManager) ConfigureDataDir(path string) error {
	clean, err := NormalizeNodeDataDirectory(path)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.command != nil {
		return ErrNodeAlreadyRunning
	}
	m.config.DataDir = clean
	return nil
}

func (m *NodeManager) ConfigurePublicNode(
	enabled bool,
	advertiseAddress string,
) error {
	advertiseAddress = strings.TrimSpace(advertiseAddress)
	if advertiseAddress != "" {
		if err := ValidatePublicNodeAdvertiseAddress(advertiseAddress); err != nil {
			return err
		}
	}
	if enabled && advertiseAddress == "" {
		return ErrDesktopPublicNodeAddress
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.PublicNode = enabled
	m.config.AdvertiseAddress = advertiseAddress
	return nil
}

func ValidatePublicNodeAdvertiseAddress(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return ErrDesktopPublicNodeAddress
	}
	host, portText, err := net.SplitHostPort(value)
	if err != nil || host == "" {
		return ErrDesktopPublicNodeAddress
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return ErrDesktopPublicNodeAddress
	}

	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "" ||
		host == "localhost" ||
		strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") {
		return ErrDesktopPublicNodeAddress
	}

	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() ||
			ip.IsPrivate() ||
			ip.IsUnspecified() ||
			ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() ||
			ip.IsMulticast() {
			return ErrDesktopPublicNodeAddress
		}
		return nil
	}

	if len(host) > 253 || !strings.Contains(host, ".") {
		return ErrDesktopPublicNodeAddress
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 ||
			label[0] == '-' || label[len(label)-1] == '-' {
			return ErrDesktopPublicNodeAddress
		}
		for _, r := range label {
			if (r >= 'a' && r <= 'z') ||
				(r >= '0' && r <= '9') ||
				r == '-' {
				continue
			}
			return ErrDesktopPublicNodeAddress
		}
	}
	return nil
}

// ProcessID returns the PID of the currently managed valdrd child.
// It is exposed to the Desktop backend for local diagnostics and runtime
// recovery checks; it is not part of node RPC or the frontend API.
func (m *NodeManager) ProcessID() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.command == nil || m.command.Process == nil {
		return 0
	}
	return m.command.Process.Pid
}

func (m *NodeManager) LastExitError() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastExit
}

func (m *NodeManager) recordProcessExit(cmd *exec.Cmd, waitErr error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.command != cmd {
		return
	}

	unexpected := !m.stopping
	m.command = nil
	m.stdin = nil
	m.stopping = false

	if !unexpected {
		return
	}
	if waitErr == nil {
		m.lastExit = ErrNodeExitedUnexpectedly
		return
	}
	m.lastExit = fmt.Errorf("%w: %v", ErrNodeExitedUnexpectedly, waitErr)
}

func (m *NodeManager) Endpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%d", m.config.RPCPort)
}

func (m *NodeManager) Status(
	ctx context.Context,
) (rpc.StatusResult, error) {
	var status rpc.StatusResult
	err := rpc.NewClient(m.Endpoint()).Call(
		ctx,
		rpc.MethodGetStatus,
		nil,
		&status,
	)
	return status, err
}

func desktopNodeArgs(cfg NodeProcessConfig) ([]string, error) {
	profile, err := config.ResolveNetworkProfile(cfg.Network)
	if err != nil {
		return nil, err
	}
	if profile.Name != config.NetworkTestnetV029 &&
		profile.Name != config.NetworkDevnetV02 {
		return nil, ErrDesktopMainnet
	}
	if strings.TrimSpace(cfg.DataDir) == "" {
		return nil, errors.New("desktop node data directory is required")
	}
	if strings.TrimSpace(cfg.NodeID) == "" {
		return nil, errors.New("desktop node id is required")
	}
	port := cfg.RPCPort
	if port == 0 {
		port = profile.RPCPort
	}

	args := []string{
		"start",
		"--network", profile.Name,
		"--data", cfg.DataDir,
		"--node-id", cfg.NodeID,
	}
	if cfg.PublicNode {
		advertiseAddress := strings.TrimSpace(cfg.AdvertiseAddress)
		if err := ValidatePublicNodeAdvertiseAddress(advertiseAddress); err != nil {
			return nil, err
		}
		args = append(
			args,
			"--p2p-host", "0.0.0.0",
			"--advertise-address", advertiseAddress,
		)
	} else {
		args = append(args, "--outbound-only")
	}
	args = append(
		args,
		"--managed-stdin-shutdown",
		"--rpc-host", "127.0.0.1",
		"--rpc-port", strconv.Itoa(int(port)),
	)
	for _, seed := range cfg.Seeds {
		seed = strings.TrimSpace(seed)
		if seed == "" {
			continue
		}
		args = append(args, "--seed", seed)
	}
	return args, nil
}

func bundledBinaryPath(name string) (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(filepath.Dir(executable), name), nil
}
