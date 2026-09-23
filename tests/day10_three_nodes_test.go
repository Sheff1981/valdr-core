package tests

import (
	"context"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/mempool"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/mining"
	"github.com/Sheff1981/valdr-core/p2p"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestDay10ThreeNodesConvergeOnSameConfirmedChain(t *testing.T) {
	minerWallet, err := wallet.New("miner")
	if err != nil {
		t.Fatal(err)
	}
	recipientWallet, err := wallet.New("recipient")
	if err != nil {
		t.Fatal(err)
	}

	chainA := blockchain.New()
	chainB := blockchain.New()
	chainC := blockchain.New()
	poolA := mempool.New()
	poolB := mempool.New()
	poolC := mempool.New()

	for i := int64(1); i <= 2; i++ {
		if _, err := mining.MineBlock(
			chainA,
			minerWallet.Address,
			config.GenesisTimestamp+i*60,
			nil,
		); err != nil {
			t.Fatalf("mine initial block %d: %v", i, err)
		}
	}

	nodeA := startDay10Node(t, "node-a", chainA, poolA)
	defer nodeA.Close()
	nodeB := startDay10Node(t, "node-b", chainB, poolB)
	defer nodeB.Close()
	nodeC := startDay10Node(t, "node-c", chainC, poolC)
	defer nodeC.Close()

	connectDay10(t, nodeB, nodeA)
	waitDay10(t, "Node B sync to height 2", func() bool {
		return chainB.Height() == 2
	})

	connectDay10(t, nodeC, nodeB)
	waitDay10(t, "Node C sync to height 2 through Node B", func() bool {
		return chainC.Height() == 2
	})

	assertDay10ChainsEqual(t, chainA, chainB, chainC)

	if nodeA.PeerCount() != 1 {
		t.Fatalf("Node A peer count = %d, want 1", nodeA.PeerCount())
	}
	if nodeB.PeerCount() != 2 {
		t.Fatalf("Node B peer count = %d, want 2", nodeB.PeerCount())
	}
	if nodeC.PeerCount() != 1 {
		t.Fatalf("Node C peer count = %d, want 1", nodeC.PeerCount())
	}

	available, err := chainC.UTXOs(minerWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	payment, err := minerWallet.CreateTransaction(
		available,
		recipientWallet.Address,
		25*config.AtomicUnitsPerVDR,
		config.GenesisTimestamp+150,
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := nodeC.BroadcastTransaction(payment); err != nil {
		t.Fatalf("Node C transaction broadcast: %v", err)
	}

	waitDay10(t, "transaction reaches all three mempools", func() bool {
		return poolA.Contains(payment.TransactionID) &&
			poolB.Contains(payment.TransactionID) &&
			poolC.Contains(payment.TransactionID)
	})

	block3, err := mining.MineBlock(
		chainA,
		minerWallet.Address,
		config.GenesisTimestamp+180,
		[]*transaction.Transaction{payment},
	)
	if err != nil {
		t.Fatalf("Node A mine block 3: %v", err)
	}

	if err := nodeA.BroadcastBlock(block3); err != nil {
		t.Fatalf("Node A block broadcast: %v", err)
	}

	waitDay10(t, "Node B and Node C receive block 3", func() bool {
		return chainB.Height() == 3 && chainC.Height() == 3
	})
	waitDay10(t, "confirmed transaction leaves all mempools", func() bool {
		return poolA.Len() == 0 && poolB.Len() == 0 && poolC.Len() == 0
	})

	assertDay10ChainsEqual(t, chainA, chainB, chainC)

	for name, chain := range map[string]*blockchain.Blockchain{
		"A": chainA,
		"B": chainB,
		"C": chainC,
	} {
		recipientBalance, err := chain.Balance(recipientWallet.Address)
		if err != nil {
			t.Fatalf("Node %s recipient balance: %v", name, err)
		}
		if recipientBalance != 25*config.AtomicUnitsPerVDR {
			t.Fatalf(
				"Node %s recipient balance = %d, want 25 VDR",
				name,
				recipientBalance,
			)
		}

		minerBalance, err := chain.Balance(minerWallet.Address)
		if err != nil {
			t.Fatalf("Node %s miner balance: %v", name, err)
		}
		if minerBalance != 125*config.AtomicUnitsPerVDR {
			t.Fatalf(
				"Node %s miner balance = %d, want 125 VDR",
				name,
				minerBalance,
			)
		}
	}
}

func startDay10Node(
	t *testing.T,
	nodeID string,
	chain *blockchain.Blockchain,
	pool *mempool.Pool,
) *p2p.Node {
	t.Helper()

	node, err := p2p.NewNode(p2p.NodeConfig{
		NodeID:        nodeID,
		ListenAddress: "127.0.0.1:0",
		Blockchain:    chain,
		Mempool:       pool,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := node.Start(); err != nil {
		t.Fatal(err)
	}
	return node
}

func connectDay10(t *testing.T, from, to *p2p.Node) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := from.Connect(ctx, to.Address()); err != nil {
		t.Fatalf("%s -> %s connect: %v", from.NodeID(), to.NodeID(), err)
	}
}

func waitDay10(t *testing.T, description string, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(7 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", description)
}

func assertDay10ChainsEqual(
	t *testing.T,
	chains ...*blockchain.Blockchain,
) {
	t.Helper()

	if len(chains) < 2 {
		t.Fatal("need at least two chains to compare")
	}
	height := chains[0].Height()
	for i, chain := range chains[1:] {
		if chain.Height() != height {
			t.Fatalf(
				"chain %d height = %d, want %d",
				i+1,
				chain.Height(),
				height,
			)
		}
	}

	for h := uint64(0); h <= height; h++ {
		reference, ok := chains[0].BlockAt(h)
		if !ok {
			t.Fatalf("reference chain missing block %d", h)
		}
		for i, chain := range chains[1:] {
			candidate, ok := chain.BlockAt(h)
			if !ok {
				t.Fatalf("chain %d missing block %d", i+1, h)
			}
			if candidate.BlockHash != reference.BlockHash {
				t.Fatalf(
					"chain %d block %d hash = %s, want %s",
					i+1,
					h,
					candidate.BlockHash,
					reference.BlockHash,
				)
			}
		}
	}
}
