package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/transaction"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
	"github.com/Sheff1981/valdr-core/mining"
)

func TestTestnet2MultiHopTransactionAndBlockRelay(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}

	chainA, err := blockchain.NewForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	chainB, err := blockchain.NewForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	chainC, err := blockchain.NewForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}

	minerKey, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	minerAddress, err := valdrcrypto.AddressFromPublicKey(&minerKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	miner2Key, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	miner2Address, err := valdrcrypto.AddressFromPublicKey(&miner2Key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	recipientKey, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	recipientAddress, err := valdrcrypto.AddressFromPublicKey(&recipientKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	nodeA := mustStartNode(t, NodeConfig{
		NodeID: "testnet2-relay-a", ListenAddress: "127.0.0.1:0",
		NetworkProfile: &profile, EnableV2: true, Blockchain: chainA,
	})
	defer nodeA.Close()
	nodeB := mustStartNode(t, NodeConfig{
		NodeID: "testnet2-relay-b", ListenAddress: "127.0.0.1:0",
		NetworkProfile: &profile, EnableV2: true, Blockchain: chainB,
	})
	defer nodeB.Close()
	nodeC := mustStartNode(t, NodeConfig{
		NodeID: "testnet2-relay-c", ListenAddress: "127.0.0.1:0",
		NetworkProfile: &profile, EnableV2: true, Blockchain: chainC,
	})
	defer nodeC.Close()

	connect := func(from *Node, to string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := from.Connect(ctx, to); err != nil {
			t.Fatal(err)
		}
	}
	connect(nodeB, nodeA.Address())
	connect(nodeC, nodeB.Address())
	waitForPeerCount(t, nodeA, 1)
	waitForPeerCount(t, nodeB, 2)
	waitForPeerCount(t, nodeC, 1)

	block1, err := mining.MineBlock(
		chainA,
		minerAddress,
		profile.GenesisTimestamp+profile.TargetBlockTimeSeconds,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := nodeA.BroadcastBlock(block1); err != nil {
		t.Fatal(err)
	}
	waitForSameTip(t, chainB, chainA, 10*time.Second)
	waitForSameTip(t, chainC, chainA, 10*time.Second)

	funding := block1.Transactions[0]
	const amount = uint64(25_000_000)
	const fee = uint64(10_000)
	inputValue := profile.InitialSubsidyVDR * config.AtomicUnitsPerVDR
	if inputValue <= amount+fee {
		t.Fatalf("invalid test funding value=%d", inputValue)
	}
	spend := transaction.NewForChain(
		profile.ChainID,
		[]transaction.Input{{
			PreviousTransactionID: funding.TransactionID,
			OutputIndex: 0,
		}},
		[]transaction.Output{
			{Amount: amount, Recipient: recipientAddress},
			{Amount: inputValue - amount - fee, Recipient: minerAddress},
		},
		profile.GenesisTimestamp+2*profile.TargetBlockTimeSeconds,
	)
	if err := spend.Sign(minerKey); err != nil {
		t.Fatal(err)
	}
	if fee < uint64(spend.SerializedSize())*profile.MinRelayFeePerByte {
		t.Fatalf("test fee=%d below relay minimum for size=%d", fee, spend.SerializedSize())
	}

	if err := nodeA.BroadcastTransaction(spend); err != nil {
		t.Fatal(err)
	}
	waitMempool := func(node *Node) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			for _, tx := range node.MempoolTransactions() {
				if tx != nil && tx.TransactionID == spend.TransactionID {
					return
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("transaction %s did not reach node %s", spend.TransactionID, node.NodeID())
	}
	waitMempool(nodeA)
	waitMempool(nodeB)
	waitMempool(nodeC)

	block2, err := mining.MineBlock(
		chainB,
		miner2Address,
		profile.GenesisTimestamp+2*profile.TargetBlockTimeSeconds,
		nodeB.MempoolTransactionsForMining(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := nodeB.BroadcastBlock(block2); err != nil {
		t.Fatal(err)
	}

	waitForSameTip(t, chainA, chainB, 10*time.Second)
	waitForSameTip(t, chainC, chainB, 10*time.Second)

	for _, item := range []struct {
		name  string
		chain *blockchain.Blockchain
		node  *Node
	}{
		{name: "A", chain: chainA, node: nodeA},
		{name: "B", chain: chainB, node: nodeB},
		{name: "C", chain: chainC, node: nodeC},
	} {
		if item.node.MempoolLen() != 0 {
			t.Fatalf("node %s mempool=%d want=0", item.name, item.node.MempoolLen())
		}
		balance, err := item.chain.Balance(recipientAddress)
		if err != nil {
			t.Fatal(err)
		}
		if balance != amount {
			t.Fatalf("node %s recipient balance=%d want=%d", item.name, balance, amount)
		}
	}
}
