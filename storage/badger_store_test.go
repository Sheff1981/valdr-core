package storage_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/mining"
	"github.com/Sheff1981/valdr-core/storage"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestBadgerStoreIndexesRestartAndAtomicFailure(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewBadgerStore(dir, config.ChainID, config.GenesisBlockHash)
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

	info, err := store.Info()
	if err != nil {
		t.Fatal(err)
	}
	if info.SchemaVersion != storage.StorageSchemaVersion ||
		info.Network != config.ChainID ||
		info.GenesisHash != config.GenesisBlockHash ||
		info.Height != 2 ||
		info.ActiveTip != block2.BlockHash ||
		info.UTXOHash != storage.HashUTXOSet(chain.UTXOSnapshot()) {
		t.Fatalf("unexpected storage info: %+v", info)
	}

	for prefix, want := range map[string]int{
		"block/":  3,
		"height/": 3,
		"header/": 3,
		"tx/":     3,
		"undo/":   2,
		"utxo/":   3,
	} {
		got, err := store.KeyCount(prefix)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s key count = %d, want %d", prefix, got, want)
		}
	}

	tipHash := block2.BlockHash
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopenedStore, err := storage.NewBadgerStore(dir, config.ChainID, config.GenesisBlockHash)
	if err != nil {
		t.Fatal(err)
	}
	reopenedChain, err := blockchain.NewPersistent(reopenedStore)
	if err != nil {
		t.Fatal(err)
	}
	if reopenedChain.Height() != 2 || reopenedChain.Tip().BlockHash != tipHash {
		t.Fatalf("reopened chain height/tip = %d/%s", reopenedChain.Height(), reopenedChain.Tip().BlockHash)
	}
	recipientBalance, err := reopenedChain.Balance(recipientWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	if recipientBalance != 10*config.AtomicUnitsPerVDR {
		t.Fatalf("recipient balance = %d, want 10 VDR", recipientBalance)
	}

	// Closing the store forces persistence failure. Blockchain must roll back
	// the candidate so confirmed RAM state cannot outrun disk state.
	if err := reopenedStore.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = mining.MineBlock(
		reopenedChain,
		minerWallet.Address,
		config.GenesisTimestamp+180,
		nil,
	)
	if !errors.Is(err, blockchain.ErrPersistence) {
		t.Fatalf("mine with closed store error = %v, want ErrPersistence", err)
	}
	if reopenedChain.Height() != 2 || reopenedChain.Tip().BlockHash != tipHash {
		t.Fatalf("chain mutated after persistence failure")
	}

	finalStore, err := storage.NewBadgerStore(dir, config.ChainID, config.GenesisBlockHash)
	if err != nil {
		t.Fatal(err)
	}
	defer finalStore.Close()
	finalChain, err := blockchain.NewPersistent(finalStore)
	if err != nil {
		t.Fatal(err)
	}
	if finalChain.Height() != 2 || finalChain.Tip().BlockHash != tipHash {
		t.Fatalf("disk state changed after failed commit")
	}
}

func TestMigrateV01ToBadgerPreservesSourceAndState(t *testing.T) {
	dir := t.TempDir()
	legacyStore, err := storage.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	legacyChain, err := blockchain.NewPersistent(legacyStore)
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
		legacyChain,
		minerWallet.Address,
		config.GenesisTimestamp+60,
		nil,
	); err != nil {
		t.Fatal(err)
	}
	available, err := legacyChain.UTXOs(minerWallet.Address)
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
	if _, err := mining.MineBlock(
		legacyChain,
		minerWallet.Address,
		config.GenesisTimestamp+120,
		[]*transaction.Transaction{payment},
	); err != nil {
		t.Fatal(err)
	}

	legacyPath := filepath.Join(dir, "blockchain.json")
	before, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatal(err)
	}

	report, err := storage.MigrateV01(dir, "valdr-devnet-1")
	if err != nil {
		t.Fatal(err)
	}
	if !report.OriginalPreserved ||
		report.Height != legacyChain.Height() ||
		report.TipHash != legacyChain.Tip().BlockHash ||
		report.UTXOHash != storage.HashUTXOSet(legacyChain.UTXOSnapshot()) {
		t.Fatalf("migration report = %+v", report)
	}

	after, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("migration modified original blockchain.json")
	}

	targetStore, err := storage.NewBadgerStore(dir, config.ChainID, config.GenesisBlockHash)
	if err != nil {
		t.Fatal(err)
	}
	defer targetStore.Close()
	info, err := targetStore.Info()
	if err != nil {
		t.Fatal(err)
	}
	if info.MigrationStatus != "complete" {
		t.Fatalf("migration status = %q, want complete", info.MigrationStatus)
	}
	targetChain, err := blockchain.NewPersistent(targetStore)
	if err != nil {
		t.Fatal(err)
	}
	if targetChain.Height() != legacyChain.Height() ||
		targetChain.Tip().BlockHash != legacyChain.Tip().BlockHash ||
		storage.HashUTXOSet(targetChain.UTXOSnapshot()) != storage.HashUTXOSet(legacyChain.UTXOSnapshot()) {
		t.Fatal("migrated chain state differs from legacy source")
	}

	if _, err := storage.MigrateV01(dir, "valdr-devnet-1"); !errors.Is(err, storage.ErrMigrationTargetExists) {
		t.Fatalf("second migration error = %v, want ErrMigrationTargetExists", err)
	}
}
