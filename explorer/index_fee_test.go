package explorer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/rpc"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestExplorerIndexRecordsSenderFee(t *testing.T) {
	owner, err := wallet.New("fee-owner")
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := wallet.New("fee-recipient")
	if err != nil {
		t.Fatal(err)
	}

	genesis := *block.NewGenesis()
	coinbase, err := transaction.NewCoinbase(
		1,
		owner.Address,
		config.InitialMiningReward,
		config.GenesisTimestamp+60,
	)
	if err != nil {
		t.Fatal(err)
	}
	const fee = uint64(1_000)
	payment := &transaction.Transaction{
		Version: transaction.VersionLegacy,
		Inputs: []transaction.Input{{
			PreviousTransactionID: coinbase.TransactionID,
			OutputIndex:           0,
		}},
		Outputs: []transaction.Output{{
			Amount:    config.InitialMiningReward - fee,
			Recipient: recipient.Address,
		}},
		Timestamp:     config.GenesisTimestamp + 120,
		TransactionID: strings.Repeat("e", 64),
	}

	h1 := strings.Repeat("a", 64)
	h2 := strings.Repeat("b", 64)
	client := &mockClient{
		status: rpc.StatusResult{
			Network:  config.NetworkLegacyV01,
			ChainID:  config.ChainID,
			Height:   2,
			TipHash:  h2,
			Chainwork: "01",
		},
		blocks: map[uint64]rpc.BlockResult{
			0: genesis,
			1: {
				Version:           1,
				Height:            1,
				PreviousBlockHash: genesis.BlockHash,
				BlockHash:         h1,
				ChainID:           config.ChainID,
				Timestamp:         config.GenesisTimestamp + 60,
				Transactions:      []*transaction.Transaction{coinbase},
			},
			2: {
				Version:           1,
				Height:            2,
				PreviousBlockHash: h1,
				BlockHash:         h2,
				ChainID:           config.ChainID,
				Timestamp:         config.GenesisTimestamp + 120,
				Transactions:      []*transaction.Transaction{payment},
			},
		},
	}

	idx, err := NewIndex(client, filepath.Join(t.TempDir(), "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := idx.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}

	history := idx.Activities(owner.Address)
	if len(history) != 2 {
		t.Fatalf("owner history=%+v want coinbase+spend", history)
	}
	var spend AddressActivity
	for _, item := range history {
		if item.TransactionID == payment.TransactionID {
			spend = item
			break
		}
	}
	if spend.TransactionID == "" {
		t.Fatalf("spend activity missing: %+v", history)
	}
	if spend.SpentVal != config.InitialMiningReward ||
		spend.ReceivedVal != 0 ||
		spend.FeeVal != fee {
		t.Fatalf("unexpected sender activity: %+v", spend)
	}
}

func TestExplorerOldDerivedIndexRebuilds(t *testing.T) {
	client, _, _, _ := explorerFixture(t)
	path := filepath.Join(t.TempDir(), "index.json")
	old := map[string]any{
		"version":      1,
		"chain_id":     config.ChainID,
		"genesis_hash": config.GenesisBlockHash,
		"height":       99,
		"tip_hash":     strings.Repeat("f", 64),
		"blocks":       map[string]string{"0": config.GenesisBlockHash},
		"activities":   map[string]any{},
		"outpoints":    map[string]any{},
	}
	raw, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	idx, err := NewIndex(client, path)
	if err != nil {
		t.Fatal(err)
	}
	if height, tip := idx.State(); height != 0 || tip != "" {
		t.Fatalf("old derived index was not discarded: %d/%s", height, tip)
	}
	if err := idx.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if height, tip := idx.State(); height != client.status.Height ||
		tip != client.status.TipHash {
		t.Fatalf(
			"rebuilt state=%d/%s want=%d/%s",
			height,
			tip,
			client.status.Height,
			client.status.TipHash,
		)
	}
}
