package p2p

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/mempool"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/mining"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestTestnet2PendingRelayThenAnyMinerConfirmation(t *testing.T) {
	if os.Getenv("VALDR_TESTNET2_RUNTIME") != "1" {
		t.Skip("set VALDR_TESTNET2_RUNTIME=1 for the dedicated Testnet2 runtime gate")
	}

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

	sender, err := wallet.New("testnet2-relay-sender")
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := wallet.New("testnet2-relay-recipient")
	if err != nil {
		t.Fatal(err)
	}

	funding, err := mining.MineBlock(
		chainA,
		sender.Address,
		profile.GenesisTimestamp+profile.TargetBlockTimeSeconds,
		nil,
	)
	if err != nil {
		t.Fatalf("mine funding block: %v", err)
	}
	if funding.Height != 1 {
		t.Fatalf("funding height=%d want=1", funding.Height)
	}

	poolA := mempool.NewWithConfig(mempool.Config{
		ChainID:            profile.ChainID,
		MinRelayFeePerByte: profile.MinRelayFeePerByte,
	})
	poolB := mempool.NewWithConfig(mempool.Config{
		ChainID:            profile.ChainID,
		MinRelayFeePerByte: profile.MinRelayFeePerByte,
	})

	nodeA := mustStartNode(t, NodeConfig{
		NodeID:         "testnet2-relay-node-a",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Blockchain:     chainA,
		Mempool:        poolA,
	})
	defer nodeA.Close()
	nodeB := mustStartNode(t, NodeConfig{
		NodeID:         "testnet2-relay-node-b",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Blockchain:     chainB,
		Mempool:        poolB,
	})
	defer nodeB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := nodeB.Connect(ctx, nodeA.Address()); err != nil {
		cancel()
		t.Fatalf("connect nodes: %v", err)
	}
	cancel()
	waitForSameTip(t, chainB, chainA, 15*time.Second)

	available, err := chainA.UTXOs(sender.Address)
	if err != nil {
		t.Fatal(err)
	}
	const amount = uint64(25_000_000)
	payment, fee, err := sender.CreateTransactionForChain(
		profile.ChainID,
		available,
		recipient.Address,
		amount,
		profile.MinRelayFeePerByte,
		profile.GenesisTimestamp+2*profile.TargetBlockTimeSeconds,
	)
	if err != nil {
		t.Fatal(err)
	}
	if fee == 0 {
		t.Fatal("expected non-zero Testnet2 relay fee")
	}

	// Mining is deliberately OFF here. The transaction must still relay and
	// become visible in the receiving node's mempool at 0 confirmations.
	if err := nodeA.BroadcastTransaction(payment); err != nil {
		t.Fatalf("broadcast transaction: %v", err)
	}
	waitForCondition(t, "Testnet2 payment reaches remote mempool without mining", func() bool {
		return poolB.Contains(payment.TransactionID)
	})
	if !poolA.Contains(payment.TransactionID) {
		t.Fatal("sender mempool lost locally broadcast transaction")
	}
	if chainB.Height() != 1 {
		t.Fatalf("receiver chain advanced before mining: height=%d", chainB.Height())
	}
	before, err := chainB.Balance(recipient.Address)
	if err != nil {
		t.Fatal(err)
	}
	if before != 0 {
		t.Fatalf("unconfirmed recipient balance=%d want=0 active-chain balance", before)
	}

	// Any miner may confirm it. Mine only on node A's chain and relay the block
	// to node B; the receiver itself never starts a miner.
	block2, err := mining.MineBlock(
		chainA,
		sender.Address,
		profile.GenesisTimestamp+2*profile.TargetBlockTimeSeconds,
		[]*transaction.Transaction{payment},
	)
	if err != nil {
		t.Fatalf("mine confirmation block: %v", err)
	}
	if err := nodeA.BroadcastBlock(block2); err != nil {
		t.Fatalf("broadcast confirmation block: %v", err)
	}
	waitForSameTip(t, chainB, chainA, 15*time.Second)
	waitForCondition(t, "confirmed payment removed from both mempools", func() bool {
		return poolA.Len() == 0 && poolB.Len() == 0
	})

	after, err := chainB.Balance(recipient.Address)
	if err != nil {
		t.Fatal(err)
	}
	if after != amount {
		t.Fatalf("confirmed recipient balance=%d want=%d", after, amount)
	}
	location, ok := chainB.TransactionByID(payment.TransactionID)
	if !ok || location.Transaction == nil || location.BlockHeight != 2 {
		t.Fatalf("receiver did not index confirmed txid=%s", payment.TransactionID)
	}
}
