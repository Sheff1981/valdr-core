package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/mempool"
	"github.com/Sheff1981/valdr-core/p2p"
	"github.com/Sheff1981/valdr-core/rpc"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

type stringListFlag []string

func (s *stringListFlag) String() string { return fmt.Sprint([]string(*s)) }

func (s *stringListFlag) Set(value string) error {
	if value == "" {
		return errors.New("peer address is empty")
	}
	*s = append(*s, value)
	return nil
}

func run(args []string, out, errOut io.Writer) int {
	if len(args) == 0 || args[0] == "version" {
		fmt.Fprintf(out, "%s %s valdrd %s\n", config.ProjectName, config.Ticker, config.Version)
		return 0
	}

	switch args[0] {
	case "init":
		return initCommand(args[1:], out, errOut)
	case "status":
		return statusCommand(args[1:], out, errOut)
	case "start":
		return startCommand(args[1:], out, errOut)
	default:
		fmt.Fprintf(errOut, "unknown command %q\n", args[0])
		return 2
	}
}

func initCommand(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(errOut)
	dataDir := fs.String("data", "./data", "VALDR node data directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(errOut, "usage: valdrd init [--data PATH]")
		return 2
	}

	if err := os.MkdirAll(*dataDir, 0o700); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if err := os.Chmod(*dataDir, 0o700); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	metadata := struct {
		ChainID string `json:"chain_id"`
		Version string `json:"version"`
	}{
		ChainID: config.ChainID,
		Version: config.Version,
	}
	raw, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(*dataDir, "node.json"), raw, 0o600); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	return writeJSON(out, map[string]any{
		"data":     *dataDir,
		"chain_id": config.ChainID,
	}, errOut)
}

func statusCommand(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(errOut)
	node := fs.String("node", "http://127.0.0.1:7332", "VALDR RPC endpoint")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(errOut, "usage: valdrd status [--node URL]")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var result rpc.StatusResult
	if err := rpc.NewClient(*node).Call(ctx, rpc.MethodGetStatus, nil, &result); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return writeJSON(out, result, errOut)
}

func startCommand(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	fs.SetOutput(errOut)

	dataDir := fs.String("data", "./data", "VALDR node data directory")
	nodeID := fs.String("node-id", "valdr-node", "P2P node id")
	p2pHost := fs.String("p2p-host", "127.0.0.1", "P2P listen host")
	p2pPort := fs.Uint("p2p-port", uint(config.DefaultP2PPort), "P2P listen port")
	rpcHost := fs.String("rpc-host", "127.0.0.1", "RPC listen host")
	rpcPort := fs.Uint("rpc-port", uint(config.DefaultRPCPort), "RPC listen port")
	var peers stringListFlag
	fs.Var(&peers, "peer", "P2P peer address; may be repeated")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 || *p2pPort > 65535 || *rpcPort > 65535 {
		fmt.Fprintln(errOut, "invalid valdrd start arguments")
		return 2
	}

	if err := os.MkdirAll(*dataDir, 0o700); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	chain := blockchain.New()
	pool := mempool.New()
	p2pAddress := *p2pHost + ":" + strconv.FormatUint(uint64(*p2pPort), 10)
	node, err := p2p.NewNode(p2p.NodeConfig{
		NodeID:        *nodeID,
		ListenAddress: p2pAddress,
		Blockchain:    chain,
		Mempool:       pool,
	})
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if err := node.Start(); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer node.Close()

	for _, peerAddress := range peers {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := node.Connect(ctx, peerAddress)
		cancel()
		if err != nil {
			fmt.Fprintf(errOut, "connect peer %s: %v\n", peerAddress, err)
			return 1
		}
	}

	rpcServer, err := rpc.NewServer(chain, node)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	httpServer := &http.Server{
		Addr:              *rpcHost + ":" + strconv.FormatUint(uint64(*rpcPort), 10),
		Handler:           rpcServer.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	if err := writeJSON(out, map[string]any{
		"node_id":     node.NodeID(),
		"chain_id":    config.ChainID,
		"p2p_address": node.Address(),
		"rpc_address": httpServer.Addr,
		"data":        *dataDir,
	}, errOut); err != 0 {
		return err
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case sig := <-signals:
		fmt.Fprintf(errOut, "stopping on signal %s\n", sig)
	case err := <-serverErr:
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return 0
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
