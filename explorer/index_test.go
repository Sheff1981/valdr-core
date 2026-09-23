package explorer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/rpc"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestIndexPersistsAndReloadsAddressActivity(t *testing.T) {
	owner, err := wallet.New("persistent-index-owner")
	if err != nil {
		t.Fatal(err)
	}
	coinbase, err := transaction.NewCoinbase(
		1,
		owner.Address,
		config.InitialMiningReward,
		config.GenesisTimestamp+60,
	)
	if err != nil {
		t.Fatal(err)
	}
	const blockHash = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	block1 := block.Block{
		Height:            1,
		PreviousBlockHash: config.GenesisBlockHash,
		BlockHash:         blockHash,
		Timestamp:         config.GenesisTimestamp + 60,
		Transactions:      []*transaction.Transaction{coinbase},
	}
	client := &mockClient{
		status: rpc.StatusResult{ChainID: config.ChainID, Height: 1, TipHash: blockHash},
		blocks: map[uint64]rpc.BlockResult{
			0: *block.NewGenesis(),
			1: block1,
		},
		byHash:   map[string]rpc.BlockResult{blockHash: block1},
		txs:      map[string]rpc.TransactionResult{},
		balances: map[string]rpc.BalanceResult{},
	}

	path := filepath.Join(t.TempDir(), "explorer", "index.json")
	index, err := NewIndex(client, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := index.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	activities := index.Activities(owner.Address)
	if len(activities) != 1 || activities[0].ReceivedVal != config.InitialMiningReward {
		t.Fatalf("activities=%+v", activities)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("index permissions=%o, want no group/other access", info.Mode().Perm())
	}

	reloaded, err := NewIndex(client, path)
	if err != nil {
		t.Fatal(err)
	}
	activities = reloaded.Activities(owner.Address)
	if len(activities) != 1 ||
		activities[0].TransactionID != coinbase.TransactionID ||
		activities[0].ReceivedVal != config.InitialMiningReward {
		t.Fatalf("reloaded activities=%+v", activities)
	}
}
