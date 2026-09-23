package storage_test

import (
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/storage"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestDevnetV02BadgerRestartPreservesExactTargetChain(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	store, err := storage.NewBadgerStore(dir, profile.ChainID, profile.GenesisHash)
	if err != nil {
		t.Fatal(err)
	}
	chain, err := blockchain.NewPersistentForProfile(store, profile)
	if err != nil {
		t.Fatal(err)
	}
	miner, err := wallet.New("devnet2-storage-miner")
	if err != nil {
		t.Fatal(err)
	}
	timestamp := profile.GenesisTimestamp + 60
	coinbase, err := transaction.NewCoinbase(
		1,
		miner.Address,
		config.InitialMiningReward,
		timestamp,
	)
	if err != nil {
		t.Fatal(err)
	}
	mined, err := chain.Append(timestamp, []*transaction.Transaction{coinbase})
	if err != nil {
		t.Fatal(err)
	}
	work := chain.Chainwork()
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopenedStore, err := storage.NewBadgerStore(dir, profile.ChainID, profile.GenesisHash)
	if err != nil {
		t.Fatal(err)
	}
	defer reopenedStore.Close()
	reopened, err := blockchain.NewPersistentForProfile(reopenedStore, profile)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Height() != 1 ||
		reopened.Tip().BlockHash != mined.BlockHash ||
		reopened.Chainwork() != work {
		t.Fatalf(
			"reopened height/tip/work=%d/%s/%s want=1/%s/%s",
			reopened.Height(),
			reopened.Tip().BlockHash,
			reopened.Chainwork(),
			mined.BlockHash,
			work,
		)
	}
	info, err := reopenedStore.Info()
	if err != nil {
		t.Fatal(err)
	}
	if info.Network != profile.ChainID ||
		info.GenesisHash != profile.GenesisHash ||
		info.Chainwork != work {
		t.Fatalf("storage info=%+v", info)
	}
}
