package desktop

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	ErrNodeAlreadyRunning = errors.New("VALDR Desktop node is already running")
	ErrDesktopMainnet     = errors.New("VALDR Desktop Mainnet is not enabled")
)

type NodeProcessConfig struct {
	BinaryPath string
	Network    string
	DataDir    string
	NodeID     string
	RPCPort    uint16
	Seeds      []string
	Stdout     io.Writer
	Stderr     io.Writer
}

type NodeManager struct {
	mu      sync.Mutex
	config  NodeProcessConfig
	command *exec.Cmd
	waitCh  chan error
}

func NewNodeManager(cfg NodeProcessConfig) (*NodeManager, error) {
	if cfg.Network == "" {
		cfg.Network = config.NetworkTestnetV02
	}
	if cfg.Network != config.NetworkTestnetV02 &&
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

	args, err := desktopNodeArgs(m.config)
	if err != nil {
		return err
	}
	cmd := exec.Command(m.config.BinaryPath, args...)
	cmd.Stdout = m.config.Stdout
	cmd.Stderr = m.config.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}

	waitCh := make(chan error, 1)
	m.command = cmd
	m.waitCh = waitCh
	go func() {
		err := cmd.Wait()
		m.mu.Lock()
		if m.command == cmd {
			m.command = nil
		}
		m.mu.Unlock()
		waitCh <- err
		close(waitCh)
	}()
	return nil
}

func (m *NodeManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	cmd := m.command
	waitCh := m.waitCh
	m.mu.Unlock()
	if cmd == nil {
		return nil
	}

	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		_ = cmd.Process.Kill()
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
	if profile.Name != config.NetworkTestnetV02 &&
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
		"--outbound-only",
		"--rpc-host", "127.0.0.1",
		"--rpc-port", strconv.Itoa(int(port)),
	}
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
