package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/mempool"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/mining"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestDay9BasicSyncTransactionAndBlockBroadcast(t *testing.T) {
	chainA := blockchain.New()
	chainB := blockchain.New()
	poolA := mempool.New()
	poolB := mempool.New()

	minerWallet, err := wallet.New("miner")
	if err != nil {
		t.Fatal(err)
	}
	recipientWallet, err := wallet.New("recipient")
	if err != nil {
		t.Fatal(err)
	}

	block1, err := mining.MineBlock(
		chainA,
		minerWallet.Address,
		config.GenesisTimestamp+60,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if block1.Height != 1 {
		t.Fatalf("block 1 height = %d, want 1", block1.Height)
	}

	block2, err := mining.MineBlock(
		chainA,
		minerWallet.Address,
		config.GenesisTimestamp+120,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if block2.Height != 2 {
		t.Fatalf("block 2 height = %d, want 2", block2.Height)
	}

	nodeA := mustStartNode(t, NodeConfig{
		NodeID:        "node-a",
		ListenAddress: "127.0.0.1:0",
		Blockchain:    chainA,
		Mempool:       poolA,
	})
	defer nodeA.Close()

	nodeB := mustStartNode(t, NodeConfig{
		NodeID:        "node-b",
		ListenAddress: "127.0.0.1:0",
		Blockchain:    chainB,
		Mempool:       poolB,
	})
	defer nodeB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := nodeB.Connect(ctx, nodeA.Address()); err != nil {
		t.Fatalf("Node B -> Node A connect: %v", err)
	}

	waitForCondition(t, "Node B sync to height 2", func() bool {
		return chainB.Height() == 2
	})

	for height := uint64(0); height <= 2; height++ {
		a, okA := chainA.BlockAt(height)
		b, okB := chainB.BlockAt(height)
		if !okA || !okB {
			t.Fatalf("missing block at height %d: A=%v B=%v", height, okA, okB)
		}
		if a.BlockHash != b.BlockHash {
			t.Fatalf(
				"block hash mismatch at height %d: A=%s B=%s",
				height,
				a.BlockHash,
				b.BlockHash,
			)
		}
	}

	available, err := chainA.UTXOs(minerWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	payment, err := minerWallet.CreateTransaction(
		available,
		recipientWallet.Address,
		10*config.AtomicUnitsPerVDR,
		config.GenesisTimestamp+150,
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := nodeA.BroadcastTransaction(payment); err != nil {
		t.Fatalf("broadcast transaction: %v", err)
	}

	waitForCondition(t, "transaction reaches Node B mempool", func() bool {
		return poolB.Contains(payment.TransactionID)
	})
	if !poolA.Contains(payment.TransactionID) {
		t.Fatal("Node A mempool does not contain locally broadcast transaction")
	}

	block3, err := mining.MineBlock(
		chainA,
		minerWallet.Address,
		config.GenesisTimestamp+180,
		[]*transaction.Transaction{payment},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := nodeA.BroadcastBlock(block3); err != nil {
		t.Fatalf("broadcast block: %v", err)
	}

	waitForCondition(t, "Node B receives block 3", func() bool {
		return chainB.Height() == 3
	})
	waitForCondition(t, "confirmed transaction removed from both mempools", func() bool {
		return poolA.Len() == 0 && poolB.Len() == 0
	})

	aTip := chainA.Tip()
	bTip := chainB.Tip()
	if aTip == nil || bTip == nil || aTip.BlockHash != bTip.BlockHash {
		t.Fatalf("tip mismatch: A=%v B=%v", aTip, bTip)
	}

	recipientBalance, err := chainB.Balance(recipientWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	if recipientBalance != 10*config.AtomicUnitsPerVDR {
		t.Fatalf("recipient balance = %d, want 10 VDR", recipientBalance)
	}

	minerBalance, err := chainB.Balance(minerWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	if minerBalance != 140*config.AtomicUnitsPerVDR {
		t.Fatalf("miner balance = %d, want 140 VDR", minerBalance)
	}
}

func TestDay9PeerDiscovery(t *testing.T) {
	nodeC := mustStartNode(t, NodeConfig{
		NodeID:        "node-c",
		ListenAddress: "127.0.0.1:0",
	})
	defer nodeC.Close()

	nodeB := mustStartNode(t, NodeConfig{
		NodeID:        "node-b",
		ListenAddress: "127.0.0.1:0",
	})
	defer nodeB.Close()

	ctxBC, cancelBC := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelBC()
	if err := nodeB.Connect(ctxBC, nodeC.Address()); err != nil {
		t.Fatalf("Node B -> Node C connect: %v", err)
	}
	waitForPeerCount(t, nodeB, 1)
	waitForPeerCount(t, nodeC, 1)

	nodeA := mustStartNode(t, NodeConfig{
		NodeID:        "node-a",
		ListenAddress: "127.0.0.1:0",
	})
	defer nodeA.Close()

	ctxAB, cancelAB := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelAB()
	if err := nodeA.Connect(ctxAB, nodeB.Address()); err != nil {
		t.Fatalf("Node A -> Node B connect: %v", err)
	}

	waitForCondition(t, "Node A discovers Node C through Node B", func() bool {
		for _, peer := range nodeA.DiscoveredPeers() {
			if peer.NodeID == "node-c" && peer.Address == nodeC.Address() {
				return true
			}
		}
		return false
	})
}

func waitForCondition(t *testing.T, description string, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", description)
}
