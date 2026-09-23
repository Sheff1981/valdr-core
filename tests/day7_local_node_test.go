package tests

import (
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/mining"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestDay7LocalNodeMineReceiveSendAndMineAgain(t *testing.T) {
	minerWallet, err := wallet.New("miner")
	if err != nil {
		t.Fatal(err)
	}
	recipientWallet, err := wallet.New("recipient")
	if err != nil {
		t.Fatal(err)
	}

	chain := blockchain.New()

	firstBlock, err := mining.MineBlock(
		chain,
		minerWallet.Address,
		config.GenesisTimestamp+60,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstBlock.Transactions) != 1 || !firstBlock.Transactions[0].IsCoinbase() {
		t.Fatal("first mined block does not contain exactly one coinbase")
	}

	minerBalance, err := chain.Balance(minerWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	if minerBalance != 50*config.AtomicUnitsPerVDR {
		t.Fatalf("miner balance after first block = %d, want 50 VDR", minerBalance)
	}

	available, err := chain.UTXOs(minerWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	payment, err := minerWallet.CreateTransaction(
		available,
		recipientWallet.Address,
		10*config.AtomicUnitsPerVDR,
		config.GenesisTimestamp+90,
	)
	if err != nil {
		t.Fatal(err)
	}

	secondBlock, err := mining.MineBlock(
		chain,
		minerWallet.Address,
		config.GenesisTimestamp+120,
		[]*transaction.Transaction{payment},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondBlock.Transactions) != 2 {
		t.Fatalf("second block transaction count = %d, want 2", len(secondBlock.Transactions))
	}

	minerBalance, err = chain.Balance(minerWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	recipientBalance, err := chain.Balance(recipientWallet.Address)
	if err != nil {
		t.Fatal(err)
	}

	if minerBalance != 90*config.AtomicUnitsPerVDR {
		t.Fatalf("miner balance = %d, want 90 VDR", minerBalance)
	}
	if recipientBalance != 10*config.AtomicUnitsPerVDR {
		t.Fatalf("recipient balance = %d, want 10 VDR", recipientBalance)
	}
	if chain.Len() != 3 {
		t.Fatalf("chain length = %d, want genesis + 2 blocks", chain.Len())
	}
}
