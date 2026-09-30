package mempool

import (
	"path/filepath"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/transaction"
)

func TestMempoolPersistenceRoundTrip(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	tx := &transaction.Transaction{
		Version:   2,
		ChainID:   profile.ChainID,
		Timestamp: profile.GenesisTimestamp + 120,
		Inputs: []transaction.Input{{
			PreviousTransactionID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			OutputIndex:           0,
			Signature:             "sig",
		}},
		Outputs: []transaction.Output{{
			Recipient: "VDR1test",
			Amount:    100,
		}},
		PublicKey: "pub",
	}
	// Persistence is deliberately format-only. Full transaction and UTXO
	// validation happens when valdrd reloads the snapshot into the Pool.
	tx.TransactionID = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	path := filepath.Join(t.TempDir(), "mempool-v2.json")
	if err := SaveFile(path, profile.ChainID, []*transaction.Transaction{tx}); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFile(path, profile.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("loaded=%d want=1", len(got))
	}
	if got[0].TransactionID != tx.TransactionID || got[0].ChainID != profile.ChainID {
		t.Fatalf("unexpected loaded transaction: %+v", got[0])
	}
}

func TestMempoolPersistenceRejectsWrongChain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mempool-v2.json")
	if err := SaveFile(path, "valdr-testnet-2", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(path, "valdr-other"); err == nil {
		t.Fatal("expected chain-id mismatch")
	}
}
