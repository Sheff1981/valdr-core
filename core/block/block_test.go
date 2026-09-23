package block

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
)

func TestGenesisBlockDeterministic(t *testing.T) {
	genesis := NewGenesis()
	if genesis.Height != 0 {
		t.Fatalf("genesis height = %d, want 0", genesis.Height)
	}
	if genesis.PreviousBlockHash != "" {
		t.Fatalf("genesis previous hash = %q, want empty", genesis.PreviousBlockHash)
	}
	if genesis.ChainID != config.ChainID {
		t.Fatalf("genesis chain id = %q, want %q", genesis.ChainID, config.ChainID)
	}
	if genesis.ExtraData != config.GenesisMessage {
		t.Fatalf("genesis message = %q, want %q", genesis.ExtraData, config.GenesisMessage)
	}
	if genesis.BlockHash != config.GenesisBlockHash {
		t.Fatalf("genesis hash = %q, want %q", genesis.BlockHash, config.GenesisBlockHash)
	}
}

func TestBlockHashIsSHA256OfHeader(t *testing.T) {
	b := New(1, "previous", 1790121660, 1, 0, []string{"tx-a", "tx-b"}, "")
	digest := sha256.Sum256(b.HeaderBytes())
	want := hex.EncodeToString(digest[:])
	if b.BlockHash != want {
		t.Fatalf("block hash = %q, want %q", b.BlockHash, want)
	}
}

func TestBlockHashChangesWhenHeaderChanges(t *testing.T) {
	b := New(1, "previous", 1790121660, 1, 0, nil, "")
	original := b.BlockHash
	b.Nonce++
	if b.CalculateHash() == original {
		t.Fatal("block hash did not change after nonce changed")
	}
}
