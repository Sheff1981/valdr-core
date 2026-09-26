package main

import (
	"bytes"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/mempool"
	"github.com/Sheff1981/valdr-core/p2p"
	"github.com/Sheff1981/valdr-core/rpc"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestMinerStartMinesOneBlockThroughRPC(t *testing.T) {
	chain := blockchain.New()
	pool := mempool.New()
	node, err := p2p.NewNode(p2p.NodeConfig{
		NodeID:        "miner-test-node",
		ListenAddress: "127.0.0.1:0",
		Blockchain:    chain,
		Mempool:       pool,
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := rpc.NewServer(chain, node)
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	rewardWallet, err := wallet.New("reward")
	if err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	pidFile := filepath.Join(t.TempDir(), "miner.pid")
	code := run([]string{
		"start",
		"--node", httpServer.URL,
		"--reward-address", rewardWallet.Address,
		"--blocks", "1",
		"--interval", "0s",
		"--pid-file", pidFile,
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("start exit=%d stderr=%s", code, errOut.String())
	}

	if chain.Height() != 1 {
		t.Fatalf("height = %d, want 1", chain.Height())
	}
	balance, err := chain.Balance(rewardWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	if balance != config.InitialMiningReward {
		t.Fatalf("reward balance = %d, want %d", balance, config.InitialMiningReward)
	}
	if !strings.Contains(out.String(), "\"height\": 1") {
		t.Fatalf("miner output = %s", out.String())
	}
}

func TestMinerStatus(t *testing.T) {
	chain := blockchain.New()
	node, err := p2p.NewNode(p2p.NodeConfig{
		NodeID:        "miner-status-node",
		ListenAddress: "127.0.0.1:0",
		Blockchain:    chain,
		Mempool:       mempool.New(),
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := rpc.NewServer(chain, node)
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	var out, errOut bytes.Buffer
	code := run([]string{
		"status",
		"--node", httpServer.URL,
		"--pid-file", filepath.Join(t.TempDir(), "missing.pid"),
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("status exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "\"next_height\": 1") {
		t.Fatalf("status output = %s", out.String())
	}
	if !strings.Contains(out.String(), "\"running\": false") {
		t.Fatalf("status output = %s", out.String())
	}
}


func TestDefaultNodeEndpointTargetsTestnet2(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	want := "http://127.0.0.1:" + strconv.Itoa(int(profile.RPCPort))
	if defaultNodeEndpoint != want {
		t.Fatalf("defaultNodeEndpoint=%q want=%q", defaultNodeEndpoint, want)
	}
}
