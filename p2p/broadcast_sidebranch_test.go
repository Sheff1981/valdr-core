package p2p

import (
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/mempool"
	"github.com/Sheff1981/valdr-core/mining"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestBroadcastBlockRelaysKnownSideBranch(t *testing.T) {
	activeChain := blockchain.New()
	siblingChain := blockchain.New()

	activeMiner, err := wallet.New("active-miner")
	if err != nil {
		t.Fatal(err)
	}
	siblingMiner, err := wallet.New("sibling-miner")
	if err != nil {
		t.Fatal(err)
	}

	activeBlock, err := mining.MineBlock(
		activeChain,
		activeMiner.Address,
		config.GenesisTimestamp+60,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	siblingBlock, err := mining.MineBlock(
		siblingChain,
		siblingMiner.Address,
		config.GenesisTimestamp+60,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if activeBlock.BlockHash == siblingBlock.BlockHash {
		t.Fatal("independent height-1 blocks unexpectedly have the same hash")
	}

	if err := activeChain.AddBlock(siblingBlock); err != nil {
		t.Fatalf("store sibling block: %v", err)
	}

	activeAtHeight, ok := activeChain.BlockAt(1)
	if !ok {
		t.Fatal("missing active block at height 1")
	}
	if activeAtHeight.BlockHash != activeBlock.BlockHash {
		t.Fatalf(
			"equal-work sibling replaced active tip: got=%s want=%s",
			activeAtHeight.BlockHash,
			activeBlock.BlockHash,
		)
	}
	if _, ok := activeChain.BlockByHash(siblingBlock.BlockHash); !ok {
		t.Fatal("side-branch block was not persisted")
	}

	node := mustStartNode(t, NodeConfig{
		NodeID:        "side-branch-relay",
		ListenAddress: "127.0.0.1:0",
		Blockchain:    activeChain,
		Mempool:       mempool.New(),
	})
	defer node.Close()

	if err := node.BroadcastBlock(siblingBlock); err != nil {
		t.Fatalf("relay known side-branch block: %v", err)
	}
}
