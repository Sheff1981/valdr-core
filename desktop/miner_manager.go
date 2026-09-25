package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
	"github.com/Sheff1981/valdr-core/rpc"
)

var (
	ErrDesktopMinerAlreadyRunning = errors.New("VALDR Desktop miner is already running")
	ErrDesktopMinerConfig         = errors.New("invalid VALDR Desktop miner configuration")
	ErrDesktopMinerRewardAddress  = errors.New("invalid VALDR mining reward address")
	ErrDesktopMinerExited         = errors.New("VALDR Desktop miner exited unexpectedly")
)

type MinerProcessConfig struct {
	BinaryPath   string
	NodeEndpoint string
	PIDFile      string
	Interval     time.Duration
	Stdout       io.Writer
	Stderr       io.Writer
}

type MinerStatus struct {
	Running                bool    `json:"running"`
	RewardAddress          string  `json:"reward_address,omitempty"`
	AcceptedBlocks         uint64  `json:"accepted_blocks"`
	LastError              string  `json:"last_error,omitempty"`
	Height                 uint64  `json:"height"`
	NextHeight             uint64  `json:"next_height"`
	CurrentTarget          string  `json:"current_target,omitempty"`
	BlockRewardVal         uint64  `json:"block_reward_val"`
	BlockRewardVDR         string  `json:"block_reward_vdr"`
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

type MinerManager struct {
	mu                    sync.Mutex
	config                MinerProcessConfig
	command               *exec.Cmd
	waitCh                chan error
	stopping              bool
	lastExit              error
	rewardAddress         string
	acceptedBlocks        uint64
	lastBlockHashrateHPS  float64
	lastBlockHashes       uint64
	lastBlockDurationMS   float64
	totalHashes           uint64
	totalMiningDurationMS float64
}

func NewMinerManager(cfg MinerProcessConfig) (*MinerManager, error) {
	cfg.NodeEndpoint = strings.TrimSpace(cfg.NodeEndpoint)
	cfg.PIDFile = strings.TrimSpace(cfg.PIDFile)
	if cfg.NodeEndpoint == "" || cfg.PIDFile == "" || cfg.Interval < 0 {
		return nil, ErrDesktopMinerConfig
	}
	if cfg.Interval == 0 {
		cfg.Interval = time.Second
	}
	if strings.TrimSpace(cfg.BinaryPath) == "" {
		binary, err := bundledBinaryPath("valdr-miner")
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
	return &MinerManager{config: cfg}, nil
}

func (m *MinerManager) Start(rewardAddress string) error {
	rewardAddress = strings.TrimSpace(rewardAddress)
	if !valdrcrypto.ValidateAddress(rewardAddress) {
		return ErrDesktopMinerRewardAddress
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.command != nil {
		return ErrDesktopMinerAlreadyRunning
	}

	args, err := desktopMinerArgs(m.config, rewardAddress)
	if err != nil {
		return err
	}
	cmd := exec.Command(m.config.BinaryPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = m.config.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}

	waitCh := make(chan error, 1)
	m.command = cmd
	m.waitCh = waitCh
	m.stopping = false
	m.lastExit = nil
	m.rewardAddress = rewardAddress
	m.acceptedBlocks = 0
	m.lastBlockHashrateHPS = 0
	m.lastBlockHashes = 0
	m.lastBlockDurationMS = 0
	m.totalHashes = 0
	m.totalMiningDurationMS = 0

	go m.consumeOutput(stdout)
	go func() {
		err := cmd.Wait()
		m.recordProcessExit(cmd, err)
		waitCh <- err
		close(waitCh)
	}()
	return nil
}

func (m *MinerManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	cmd := m.command
	waitCh := m.waitCh
	if cmd != nil {
		m.stopping = true
	}
	m.mu.Unlock()
	if cmd == nil {
		_ = os.Remove(m.config.PIDFile)
		return nil
	}

	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		_ = cmd.Process.Kill()
	}

	select {
	case <-waitCh:
		_ = os.Remove(m.config.PIDFile)
		return nil
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		select {
		case <-waitCh:
		case <-time.After(time.Second):
		}
		_ = os.Remove(m.config.PIDFile)
		return ctx.Err()
	}
}

func (m *MinerManager) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.command != nil
}

func (m *MinerManager) LastExitError() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastExit
}

func (m *MinerManager) Status(ctx context.Context) (MinerStatus, error) {
	m.mu.Lock()
	status := MinerStatus{
		Running:                m.command != nil,
		RewardAddress:          m.rewardAddress,
		AcceptedBlocks:         m.acceptedBlocks,
		LastBlockHashrateHPS:   m.lastBlockHashrateHPS,
		LastBlockHashes:        m.lastBlockHashes,
		LastBlockDurationMS:    m.lastBlockDurationMS,
		TotalHashes:            m.totalHashes,
		TotalMiningDurationMS:  m.totalMiningDurationMS,
	}
	status.HashrateHPS = effectiveHashrate(
		status.TotalHashes,
		status.TotalMiningDurationMS,
	)
	if m.lastExit != nil {
		status.LastError = m.lastExit.Error()
	}
	endpoint := m.config.NodeEndpoint
	m.mu.Unlock()

	var info rpc.MiningInfoResult
	if err := rpc.NewClient(endpoint).Call(
		ctx,
		rpc.MethodGetMiningInfo,
		nil,
		&info,
	); err != nil {
		return status, err
	}
	status.Height = info.Height
	status.NextHeight = info.NextHeight
	status.CurrentTarget = info.CurrentTarget
	status.BlockRewardVal = info.BlockRewardVal
	status.BlockRewardVDR = FormatVDR(info.BlockRewardVal)
	status.TargetBlockTimeSeconds = info.TargetBlockTimeSeconds
	status.RetargetInterval = info.RetargetInterval
	status.BlocksUntilRetarget = info.BlocksUntilRetarget
	return status, nil
}

func desktopMinerArgs(
	cfg MinerProcessConfig,
	rewardAddress string,
) ([]string, error) {
	rewardAddress = strings.TrimSpace(rewardAddress)
	if cfg.NodeEndpoint == "" || cfg.PIDFile == "" || cfg.Interval <= 0 {
		return nil, ErrDesktopMinerConfig
	}
	if !valdrcrypto.ValidateAddress(rewardAddress) {
		return nil, ErrDesktopMinerRewardAddress
	}
	return []string{
		"start",
		"--node", cfg.NodeEndpoint,
		"--reward-address", rewardAddress,
		"--blocks", "0",
		"--interval", cfg.Interval.String(),
		"--pid-file", cfg.PIDFile,
	}, nil
}

func (m *MinerManager) consumeOutput(reader io.Reader) {
	stream := reader
	if m.config.Stdout != nil {
		stream = io.TeeReader(reader, m.config.Stdout)
	}
	decoder := json.NewDecoder(stream)
	for {
		var result rpc.MineBlockResult
		if err := decoder.Decode(&result); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			return
		}
		if result.BlockHash == "" {
			continue
		}
		m.mu.Lock()
		m.acceptedBlocks++
		m.lastBlockHashrateHPS = result.HashrateHPS
		m.lastBlockHashes = result.HashesTried
		m.lastBlockDurationMS = result.MiningDurationMS
		if ^uint64(0)-m.totalHashes < result.HashesTried {
			m.totalHashes = ^uint64(0)
		} else {
			m.totalHashes += result.HashesTried
		}
		m.totalMiningDurationMS += result.MiningDurationMS
		m.mu.Unlock()
	}
}

func effectiveHashrate(totalHashes uint64, durationMS float64) float64 {
	if totalHashes == 0 || durationMS <= 0 {
		return 0
	}
	return float64(totalHashes) / (durationMS / 1000)
}

func (m *MinerManager) recordProcessExit(cmd *exec.Cmd, waitErr error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.command != cmd {
		return
	}
	unexpected := !m.stopping
	m.command = nil
	m.waitCh = nil
	m.stopping = false
	if !unexpected {
		return
	}
	if waitErr == nil {
		m.lastExit = ErrDesktopMinerExited
		return
	}
	m.lastExit = fmt.Errorf("%w: %v", ErrDesktopMinerExited, waitErr)
}
