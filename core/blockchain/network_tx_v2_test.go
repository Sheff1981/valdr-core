package blockchain

import (
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestTestnet2RejectsDevnet2TransactionReplay(t *testing.T) {
	testnet, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	devnet2, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}
	chain, err := NewForProfile(testnet)
	if err != nil {
		t.Fatal(err)
	}
	source, err := wallet.New("replay-source")
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := wallet.New("replay-recipient")
	if err != nil {
		t.Fatal(err)
	}

	coinbase, err := transaction.NewCoinbaseForChain(
		testnet.ChainID,
		1,
		source.Address,
		testnet.InitialSubsidyVDR*config.AtomicUnitsPerVDR,
		testnet.GenesisTimestamp+60,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chain.Append(
		testnet.GenesisTimestamp+60,
		[]*transaction.Transaction{coinbase},
	); err != nil {
		t.Fatal(err)
	}

	key, err := source.Private()
	if err != nil {
		t.Fatal(err)
	}
	foreign := transaction.NewForChain(
		devnet2.ChainID,
		[]transaction.Input{{
			PreviousTransactionID: coinbase.TransactionID,
			OutputIndex:           0,
		}},
		[]transaction.Output{{
			Amount:    testnet.InitialSubsidyVDR*config.AtomicUnitsPerVDR - 1_000,
			Recipient: recipient.Address,
		}},
		testnet.GenesisTimestamp+120,
	)
	if err := foreign.Sign(key); err != nil {
		t.Fatal(err)
	}

	err = chain.ValidateTransaction(foreign)
	if err == nil || !strings.Contains(err.Error(), "chain id") {
		t.Fatalf("foreign-network transaction error=%v want chain-id rejection", err)
	}
}
