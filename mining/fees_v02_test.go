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
