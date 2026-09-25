package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
	"github.com/Sheff1981/valdr-core/rpc"
)

const (
	defaultNodeEndpoint = "http://127.0.0.1:7332"
	mineBlockRPCTimeout = 5 * time.Minute
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, out, errOut io.Writer) int {
	if len(args) == 0 || args[0] == "version" {
		fmt.Fprintf(
			out,
			"%s %s valdr-miner %s\n",
			config.ProjectName,
			config.Ticker,
			config.Version,
		)
		return 0
	}

	switch args[0] {
	case "start":
		return startCommand(args[1:], out, errOut)
	case "status":
		return statusCommand(args[1:], out, errOut)
	case "stop":
		return stopCommand(args[1:], out, errOut)
	default:
		fmt.Fprintf(errOut, "unknown command %q\n", args[0])
		return 2
	}
}

func startCommand(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	fs.SetOutput(errOut)

	node := fs.String("node", defaultNodeEndpoint, "VALDR RPC endpoint")
	rewardAddress := fs.String("reward-address", "", "VDR reward address")
	blocks := fs.Uint64("blocks", 0, "number of blocks to mine; 0 means until stopped")
	interval := fs.Duration("interval", time.Second, "delay between blocks")
	pidFile := fs.String("pid-file", "", "miner PID file")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 || *rewardAddress == "" || *interval < 0 {
		fmt.Fprintln(
			errOut,
			"usage: valdr-miner start --node URL --reward-address VDR1... [--blocks N] [--interval DURATION] [--pid-file PATH]",
		)
		return 2
	}
	if !valdrcrypto.ValidateAddress(*rewardAddress) {
		fmt.Fprintln(errOut, "invalid reward address")
		return 2
	}

	resolvedPID, err := resolvePIDFile(*pidFile)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if err := claimPIDFile(resolvedPID); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer releasePIDFile(resolvedPID)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	client := rpc.NewClient(*node)
	var mined uint64

	for {
		select {
		case sig := <-signals:
			fmt.Fprintf(errOut, "miner stopping on signal %s\n", sig)
			return 0
		default:
		}

		ctx, cancel := context.WithTimeout(context.Background(), mineBlockRPCTimeout)
		var result rpc.MineBlockResult
		err := client.Call(
			ctx,
			rpc.MethodMineBlock,
			rpc.MineBlockParams{RewardAddress: *rewardAddress},
			&result,
		)
		cancel()
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}

		if code := writeJSON(out, result, errOut); code != 0 {
			return code
		}
		mined++
		if *blocks > 0 && mined >= *blocks {
			return 0
		}

		if *interval == 0 {
			continue
		}
		timer := time.NewTimer(*interval)
		select {
		case <-timer.C:
		case sig := <-signals:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			fmt.Fprintf(errOut, "miner stopping on signal %s\n", sig)
			return 0
		}
	}
}

func statusCommand(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(errOut)

	node := fs.String("node", defaultNodeEndpoint, "VALDR RPC endpoint")
	pidFile := fs.String("pid-file", "", "miner PID file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(errOut, "usage: valdr-miner status [--node URL] [--pid-file PATH]")
		return 2
	}

	resolvedPID, err := resolvePIDFile(*pidFile)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	pid, running := readRunningPID(resolvedPID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var info rpc.MiningInfoResult
	if err := rpc.NewClient(*node).Call(ctx, rpc.MethodGetMiningInfo, nil, &info); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	result := struct {
		Running    bool                 `json:"running"`
		PID        int                  `json:"pid,omitempty"`
		MiningInfo rpc.MiningInfoResult `json:"mining_info"`
	}{
		Running:    running,
		PID:        pid,
		MiningInfo: info,
	}
	return writeJSON(out, result, errOut)
}

func stopCommand(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("stop", flag.ContinueOnError)
	fs.SetOutput(errOut)
	pidFile := fs.String("pid-file", "", "miner PID file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(errOut, "usage: valdr-miner stop [--pid-file PATH]")
		return 2
	}

	resolvedPID, err := resolvePIDFile(*pidFile)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	pid, running := readRunningPID(resolvedPID)
	if !running {
		fmt.Fprintln(out, "miner is not running")
		_ = os.Remove(resolvedPID)
		return 0
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if err := process.Signal(syscall.SIGTERM); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	fmt.Fprintf(out, "stop signal sent to miner PID %d\n", pid)
	return 0
}

func resolvePIDFile(value string) (string, error) {
	if value != "" {
		return value, nil
	}
	if override := os.Getenv("VALDR_MINER_PID_FILE"); override != "" {
		return override, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".valdr", "miner.pid"), nil
}

func claimPIDFile(path string) error {
	if pid, running := readRunningPID(path); running {
		return fmt.Errorf("miner already running with PID %d", pid)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".miner-pid-*.tmp")
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
	if _, err := fmt.Fprintf(tmp, "%d\n", os.Getpid()); err != nil {
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

func releasePIDFile(path string) {
	pid, err := readPID(path)
	if err == nil && pid == os.Getpid() {
		_ = os.Remove(path)
	}
}

func readRunningPID(path string) (int, bool) {
	pid, err := readPID(path)
	if err != nil {
		return 0, false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return 0, false
	}
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return pid, false
	}
	return pid, true
}

func readPID(path string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(string(bytesTrimSpace(raw)))
	if err != nil || pid <= 0 {
		return 0, errors.New("invalid miner PID file")
	}
	return pid, nil
}

func bytesTrimSpace(raw []byte) []byte {
	start := 0
	for start < len(raw) {
		switch raw[start] {
		case ' ', '\n', '\r', '\t':
			start++
		default:
			goto trimmedStart
		}
	}
trimmedStart:
	end := len(raw)
	for end > start {
		switch raw[end-1] {
		case ' ', '\n', '\r', '\t':
			end--
		default:
			return raw[start:end]
		}
	}
	return raw[start:end]
}

func writeJSON(out io.Writer, value any, errOut io.Writer) int {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return 0
}
