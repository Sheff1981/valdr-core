package p2p

import (
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/mempool"
	"github.com/Sheff1981/valdr-core/core/transaction"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
	"github.com/Sheff1981/valdr-core/mining"
)

func TestMempoolReconsidersDisconnectedTransaction(t *testing.T) {
	chain := blockchain.New()
	ownerKey, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	ownerAddress, err := valdrcrypto.AddressFromPublicKey(&ownerKey.PublicKey)
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

	fundingBlock, err := mining.MineBlock(
		chain,
		ownerAddress,
		config.GenesisTimestamp+60,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	funding := fundingBlock.Transactions[0]

	const fee = uint64(25)
	spend := transaction.New(
		[]transaction.Input{{
			PreviousTransactionID: funding.TransactionID,
			OutputIndex:           0,
		}},
		[]transaction.Output{{
			Amount:    config.InitialMiningReward - fee,
			Recipient: recipientAddress,
		}},
		config.GenesisTimestamp+120,
	)
	if err := spend.Sign(ownerKey); err != nil {
		t.Fatal(err)
	}

	pool := mempool.New()
	node, err := NewNode(NodeConfig{
		NodeID:        "mempool-policy-node",
		ListenAddress: "127.0.0.1:0",
		Blockchain:    chain,
		Mempool:       pool,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := node.BroadcastTransaction(spend); err != nil {
		t.Fatal(err)
	}
	if !pool.Contains(spend.TransactionID) {
		t.Fatal("broadcast transaction missing from mempool")
	}

	node.applyMempoolChainUpdate(blockchain.ChainUpdate{
		Activated: true,
		Connected: []*block.Block{{
			Transactions: []*transaction.Transaction{spend},
		}},
	})
	if pool.Contains(spend.TransactionID) {
		t.Fatal("connected transaction remained in mempool")
	}

	node.applyMempoolChainUpdate(blockchain.ChainUpdate{
		Activated: true,
		Disconnected: []*block.Block{{
			Transactions: []*transaction.Transaction{spend},
		}},
	})
	if !pool.Contains(spend.TransactionID) {
		t.Fatal("valid disconnected transaction was not reconsidered")
	}
}
