package storage

import (
	"errors"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	badger "github.com/dgraph-io/badger/v4"
)

func TestBadgerCorruptActiveTipDetectedAfterRestart(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	genesis, err := block.NewGenesisForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	store, err := NewBadgerStore(dir, profile.ChainID, profile.GenesisHash)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save([]*block.Block{genesis}); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}

	if err := store.db.Update(func(txn *badger.Txn) error {
		return txn.Set(keyActiveTip, []byte("corrupt-tip"))
	}); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewBadgerStore(dir, profile.ChainID, profile.GenesisHash)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()

	if _, err := reopened.Load(); !errors.Is(err, ErrStorageMetadataCorrupt) {
		t.Fatalf("Load error=%v want ErrStorageMetadataCorrupt", err)
	}
}

func TestBadgerCorruptSchemaRejectedOnRestart(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	genesis, err := block.NewGenesisForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	store, err := NewBadgerStore(dir, profile.ChainID, profile.GenesisHash)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save([]*block.Block{genesis}); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}

	if err := store.db.Update(func(txn *badger.Txn) error {
		return txn.Set(keySchemaVersion, []byte{0x00, 0x00, 0x00, 0x01})
	}); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewBadgerStore(dir, profile.ChainID, profile.GenesisHash)
	if reopened != nil {
		_ = reopened.Close()
	}
	if !errors.Is(err, ErrStorageSchemaMismatch) {
		t.Fatalf("Open error=%v want ErrStorageSchemaMismatch", err)
	}
}
