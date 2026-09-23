package blockchain_test

import (
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/mining"
	"github.com/Sheff1981/valdr-core/storage"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestPersistentBlockchainSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	chain, err := blockchain.NewPersistent(store)
	if err != nil {
		t.Fatal(err)
	}

	minerWallet, err := wallet.New("miner")
	if err != nil {
		t.Fatal(err)
	}
	recipientWallet, err := wallet.New("recipient")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := mining.MineBlock(
		chain,
		minerWallet.Address,
		config.GenesisTimestamp+60,
		nil,
	); err != nil {
		t.Fatal(err)
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

	block2, err := mining.MineBlock(
		chain,
		minerWallet.Address,
		config.GenesisTimestamp+120,
		[]*transaction.Transaction{payment},
	)
	if err != nil {
		t.Fatal(err)
	}
	tipHash := block2.BlockHash

	reopenedStore, err := storage.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := blockchain.NewPersistent(reopenedStore)
	if err != nil {
		t.Fatal(err)
	}

	if reopened.Height() != 2 {
		t.Fatalf("reopened height = %d, want 2", reopened.Height())
	}
	if reopened.Tip() == nil || reopened.Tip().BlockHash != tipHash {
		t.Fatalf("reopened tip = %+v, want hash %s", reopened.Tip(), tipHash)
	}

	recipientBalance, err := reopened.Balance(recipientWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	if recipientBalance != 10*config.AtomicUnitsPerVDR {
		t.Fatalf("recipient balance = %d, want 10 VDR", recipientBalance)
	}

	minerBalance, err := reopened.Balance(minerWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	if minerBalance != 90*config.AtomicUnitsPerVDR {
		t.Fatalf("miner balance = %d, want 90 VDR", minerBalance)
	}

	location, ok := reopened.TransactionByID(payment.TransactionID)
	if !ok {
		t.Fatal("confirmed transaction missing after restart")
	}
	if location.BlockHeight != 2 || location.BlockHash != tipHash {
		t.Fatalf("transaction location = %+v", location)
	}
}
