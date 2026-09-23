package mining

import (
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/transaction"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

func TestMinerClaimsSubsidyPlusFees(t *testing.T) {
	chain := blockchain.New()
	minerKey, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	minerAddress, err := valdrcrypto.AddressFromPublicKey(&minerKey.PublicKey)
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

	first, err := MineBlock(
		chain,
		minerAddress,
		config.GenesisTimestamp+60,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	funding := first.Transactions[0]

	const fee = uint64(7)
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
	if err := spend.Sign(minerKey); err != nil {
		t.Fatal(err)
	}

	second, err := MineBlock(
		chain,
		minerAddress,
		config.GenesisTimestamp+120,
		[]*transaction.Transaction{spend},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := second.Transactions[0].Outputs[0].Amount; got != config.InitialMiningReward+fee {
		t.Fatalf("coinbase claim=%d want=%d", got, config.InitialMiningReward+fee)
	}
}

func TestMinerOrdersTransactionsByFeeRate(t *testing.T) {
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

	first, err := MineBlock(chain, ownerAddress, config.GenesisTimestamp+60, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := MineBlock(chain, ownerAddress, config.GenesisTimestamp+120, nil)
	if err != nil {
		t.Fatal(err)
	}

	makeSpend := func(funding *transaction.Transaction, fee uint64, timestamp int64) *transaction.Transaction {
		t.Helper()
		tx := transaction.New(
			[]transaction.Input{{
				PreviousTransactionID: funding.TransactionID,
				OutputIndex:           0,
			}},
			[]transaction.Output{{
				Amount:    config.InitialMiningReward - fee,
				Recipient: recipientAddress,
			}},
			timestamp,
		)
		if err := tx.Sign(ownerKey); err != nil {
			t.Fatal(err)
		}
		return tx
	}

	low := makeSpend(first.Transactions[0], 100, config.GenesisTimestamp+180)
	high := makeSpend(second.Transactions[0], 10_000, config.GenesisTimestamp+180)

	mined, err := MineBlock(
		chain,
		ownerAddress,
		config.GenesisTimestamp+180,
		[]*transaction.Transaction{low, high},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(mined.Transactions) != 3 {
		t.Fatalf("mined transaction count=%d want=3", len(mined.Transactions))
	}
	if mined.Transactions[1].TransactionID != high.TransactionID ||
		mined.Transactions[2].TransactionID != low.TransactionID {
		t.Fatalf(
			"miner order=%s,%s want high=%s low=%s",
			mined.Transactions[1].TransactionID,
			mined.Transactions[2].TransactionID,
			high.TransactionID,
			low.TransactionID,
		)
	}
}
