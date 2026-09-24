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
	"github.com/Sheff1981/valdr-core/logging"
	"github.com/Sheff1981/valdr-core/p2p"
	"github.com/Sheff1981/valdr-core/rpc"
	"github.com/Sheff1981/valdr-core/storage"
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
	case "migrate":
		return migrateCommand(args[1:], out, errOut)
	case "verify-db":
		return verifyDBCommand(args[1:], out, errOut)
	default:
		fmt.Fprintf(errOut, "unknown command %q\n", args[0])
		return 2
	}
}

func initCommand(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(errOut)
	dataDir := fs.String("data", "./data", "VALDR node data directory")
	networkName := fs.String("network", config.NetworkLegacyV01, "VALDR network profile")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(errOut, "usage: valdrd init [--data PATH] [--network PROFILE]")
		return 2
	}
	profile, err := config.ResolveNetworkProfile(*networkName)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}

	if err := rejectUnmigratedLegacy(*dataDir); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
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
		ChainID: profile.ChainID,
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

	blockStore, err := storage.NewBadgerStore(*dataDir, profile.ChainID, profile.GenesisHash)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer blockStore.Close()
	chain, err := blockchain.NewPersistentForProfile(blockStore, profile)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	return writeJSON(out, map[string]any{
		"data":              *dataDir,
		"network":           profile.Name,
		"chain_id":          profile.ChainID,
		"blockchain_db":     blockStore.Path(),
		"storage_schema":    storage.StorageSchemaVersion,
		"blockchain_height": chain.Height(),
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
	networkName := fs.String("network", config.NetworkLegacyV01, "VALDR network profile")
	nodeID := fs.String("node-id", "valdr-node", "P2P node id")
	p2pHost := fs.String("p2p-host", "127.0.0.1", "P2P listen host")
	advertiseAddress := fs.String("advertise-address", "", "P2P address advertised to peers; empty uses listen address")
	outboundOnly := fs.Bool("outbound-only", false, "disable inbound P2P listener; intended for Desktop clients")
	managedStdinShutdown := fs.Bool("managed-stdin-shutdown", false, "stop gracefully when managed stdin closes")
	p2pPort := fs.Uint("p2p-port", 0, "P2P listen port; 0 uses network default")
	rpcHost := fs.String("rpc-host", "127.0.0.1", "RPC listen host")
	rpcPort := fs.Uint("rpc-port", 0, "RPC listen port; 0 uses network default")
	var peers stringListFlag
	var seeds stringListFlag
	fs.Var(&peers, "peer", "P2P peer address; may be repeated")
	fs.Var(&seeds, "seed", "P2P seed address; may be repeated; failures are non-fatal")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 || *p2pPort > 65535 || *rpcPort > 65535 {
		fmt.Fprintln(errOut, "invalid valdrd start arguments")
		return 2
	}
	profile, err := config.ResolveNetworkProfile(*networkName)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	resolvedP2PPort := *p2pPort
	if resolvedP2PPort == 0 {
		resolvedP2PPort = uint(profile.P2PPort)
	}
	resolvedRPCPort := *rpcPort
	if resolvedRPCPort == 0 {
		resolvedRPCPort = uint(profile.RPCPort)
	}

	if err := rejectUnmigratedLegacy(*dataDir); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if err := os.MkdirAll(*dataDir, 0o700); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	blockStore, err := storage.NewBadgerStore(*dataDir, profile.ChainID, profile.GenesisHash)
	if err != nil {
		fmt.Fprintln(errOut, err)
		logging.Printf(logging.CategoryError, "storage init failed data=%s error=%v", *dataDir, err)
		return 1
	}
	defer blockStore.Close()
	chain, err := blockchain.NewPersistentForProfile(blockStore, profile)
	if err != nil {
		fmt.Fprintln(errOut, err)
		logging.Printf(logging.CategoryError, "blockchain load failed data=%s error=%v", *dataDir, err)
		return 1
	}
	pool := mempool.New()
	p2pAddress := ""
	if !*outboundOnly {
		p2pAddress = *p2pHost + ":" +
			strconv.FormatUint(uint64(resolvedP2PPort), 10)
	}
	node, err := p2p.NewNode(p2p.NodeConfig{
		NodeID:           *nodeID,
		ListenAddress:    p2pAddress,
		AdvertiseAddress: *advertiseAddress,
		OutboundOnly:     *outboundOnly,
		NetworkProfile:   &profile,
		EnableV2:         profile.ProtocolMax >= 2,
		Blockchain:       chain,
		Mempool:          pool,
	})
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if err := node.Start(); err != nil {
		fmt.Fprintln(errOut, err)
		logging.Printf(logging.CategoryError, "P2P start failed node=%s error=%v", *nodeID, err)
		return 1
	}
	defer node.Close()
	logging.Printf(
		logging.CategoryNode,
		"started node=%s chain=%s height=%d p2p=%s data=%s",
		*nodeID,
		profile.ChainID,
		chain.Height(),
		node.Address(),
		*dataDir,
	)

	for _, peerAddress := range peers {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := node.Connect(ctx, peerAddress)
		cancel()
		if err != nil {
			fmt.Fprintf(errOut, "connect peer %s: %v\n", peerAddress, err)
			return 1
		}
	}

	bootstrapCtx, bootstrapCancel := context.WithTimeout(context.Background(), 15*time.Second)
	bootstrapResult := node.BootstrapAndMaintain(bootstrapCtx, seeds)
	bootstrapCancel()
	for _, failure := range bootstrapResult.Failures {
		logging.Printf(
			logging.CategoryP2P,
			"seed bootstrap failed address=%s error=%s",
			failure.Address,
			failure.Error,
		)
	}

	rpcServer, err := rpc.NewServer(chain, node)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	httpServer := &http.Server{
		Addr:              *rpcHost + ":" + strconv.FormatUint(uint64(resolvedRPCPort), 10),
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
		"network":     profile.Name,
		"chain_id":    profile.ChainID,
		"p2p_address":       node.Address(),
		"advertise_address": node.AdvertiseAddress(),
		"outbound_only":     *outboundOnly,
		"rpc_address":       httpServer.Addr,
		"data":        *dataDir,
		"blockchain_db": blockStore.Path(),
		"storage_schema": storage.StorageSchemaVersion,
		"height":      chain.Height(),
		"seed_attempted": bootstrapResult.Attempted,
		"seed_connected": bootstrapResult.Connected,
	}, errOut); err != 0 {
		return err
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	var managedStop <-chan struct{}
	if *managedStdinShutdown {
		ch := make(chan struct{})
		managedStop = ch
		go func() {
			_, _ = io.Copy(io.Discard, os.Stdin)
			close(ch)
		}()
	}

	select {
	case sig := <-signals:
		fmt.Fprintf(errOut, "stopping on signal %s\n", sig)
	case <-managedStop:
		fmt.Fprintln(errOut, "stopping on managed stdin close")
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


func migrateCommand(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.SetOutput(errOut)
	source := fs.String("from-v0.1", "", "v0.1 VALDR data directory")
	network := fs.String("network", "", "source network chain ID")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 || *source == "" || *network == "" {
		fmt.Fprintln(errOut, "usage: valdrd migrate --from-v0.1 PATH --network valdr-devnet-1")
		return 2
	}
	report, err := storage.MigrateV01(*source, *network)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return writeJSON(out, report, errOut)
}

func verifyDBCommand(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("verify-db", flag.ContinueOnError)
	fs.SetOutput(errOut)
	dataDir := fs.String("data", "./data", "VALDR v0.2 data directory")
	networkName := fs.String("network", config.NetworkLegacyV01, "VALDR network profile")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(errOut, "usage: valdrd verify-db --data PATH [--network PROFILE]")
		return 2
	}
	profile, err := config.ResolveNetworkProfile(*networkName)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	store, err := storage.NewBadgerStore(*dataDir, profile.ChainID, profile.GenesisHash)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer store.Close()
	chain, err := blockchain.NewPersistentForProfile(store, profile)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	info, err := store.Info()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if info.Height != chain.Height() ||
		chain.Tip() == nil ||
		info.ActiveTip != chain.Tip().BlockHash ||
		info.UTXOHash != storage.HashUTXOSet(chain.UTXOSnapshot()) {
		fmt.Fprintln(errOut, "storage verification mismatch")
		return 1
	}
	return writeJSON(out, map[string]any{
		"valid":          true,
		"database":       store.Path(),
		"schema_version": info.SchemaVersion,
		"network":        info.Network,
		"height":         info.Height,
		"tip_hash":       info.ActiveTip,
		"utxo_hash":      info.UTXOHash,
	}, errOut)
}

func rejectUnmigratedLegacy(dataDir string) error {
	legacyPath := filepath.Join(dataDir, "blockchain.json")
	if _, err := os.Stat(legacyPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	entries, err := os.ReadDir(storage.BadgerPath(dataDir))
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf(
			"legacy v0.1 blockchain detected at %s; run: valdrd migrate --from-v0.1 %s --network valdr-devnet-1",
			legacyPath,
			dataDir,
		)
	}
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf(
			"legacy v0.1 blockchain detected without initialized v0.2 database; run migration first",
		)
	}
	return nil
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
