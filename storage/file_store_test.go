package storage

import (
	"os"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
)

func TestFileStoreRoundTrip(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if blocks, err := store.Load(); err != nil || blocks != nil {
		t.Fatalf("initial Load = %v, %v; want nil, nil", blocks, err)
	}

	genesis := block.NewGenesis()
	if err := store.Save([]*block.Block{genesis}); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 {
		t.Fatalf("loaded block count = %d, want 1", len(loaded))
	}
	if loaded[0].BlockHash != config.GenesisBlockHash {
		t.Fatalf("genesis hash = %s, want %s", loaded[0].BlockHash, config.GenesisBlockHash)
	}

	info, err := os.Stat(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("blockchain file permissions = %o, want no group/other access", info.Mode().Perm())
	}
}
