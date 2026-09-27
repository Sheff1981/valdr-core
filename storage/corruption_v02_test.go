package storage

import (
	"errors"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
	"github.com/Sheff1981/valdr-core/mining"
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


func TestBadgerCorruptUTXOHashDetectedOnLoad(t *testing.T) {
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
		return txn.Set(keyUTXOHash, []byte("corrupt-utxo-hash"))
	}); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrStorageMetadataCorrupt) {
		_ = store.Close()
		t.Fatalf("Load error=%v want ErrStorageMetadataCorrupt", err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}


func TestBadgerCorruptActiveChainworkDetectedOnLoad(t *testing.T) {
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
		return txn.Set(keyActiveChainwork, []byte("01"))
	}); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrStorageMetadataCorrupt) {
		_ = store.Close()
		t.Fatalf("Load error=%v want ErrStorageMetadataCorrupt", err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}


func TestBadgerCorruptUndoDetectedOnBranchLoad(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	store, err := NewBadgerStore(dir, profile.ChainID, profile.GenesisHash)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	chain, err := blockchain.NewPersistentForProfile(store, profile)
	if err != nil {
		t.Fatal(err)
	}
	key, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	minerAddress, err := valdrcrypto.AddressFromPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	mined, err := mining.MineBlock(
		chain,
		minerAddress,
		profile.GenesisTimestamp+profile.TargetBlockTimeSeconds,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(undoKey(mined.BlockHash))
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := store.LoadAllBlocks(); !errors.Is(err, ErrStorageMetadataCorrupt) {
		t.Fatalf("LoadAllBlocks error=%v want ErrStorageMetadataCorrupt", err)
	}
}


func TestBadgerMissingSideBranchHeaderDetectedOnLoad(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	store, err := NewBadgerStore(dir, profile.ChainID, profile.GenesisHash)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	active, err := blockchain.NewPersistentForProfile(store, profile)
	if err != nil {
		t.Fatal(err)
	}
	side, err := blockchain.NewForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	activeKey, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	activeMiner, err := valdrcrypto.AddressFromPublicKey(&activeKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	sideKey, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	sideMiner, err := valdrcrypto.AddressFromPublicKey(&sideKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := mining.MineBlock(
		active,
		activeMiner,
		profile.GenesisTimestamp+profile.TargetBlockTimeSeconds,
		nil,
	); err != nil {
		t.Fatal(err)
	}
	sideBlock, err := mining.MineBlock(
		side,
		sideMiner,
		profile.GenesisTimestamp+profile.TargetBlockTimeSeconds,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	update, err := active.AddBlockWithUpdate(sideBlock)
	if err != nil {
		t.Fatal(err)
	}
	if update.Activated {
		t.Fatal("equal-work side branch unexpectedly activated")
	}

	if err := store.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(headerKey(sideBlock.BlockHash))
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadAllBlocks(); !errors.Is(err, ErrStorageMetadataCorrupt) {
		t.Fatalf("LoadAllBlocks error=%v want ErrStorageMetadataCorrupt", err)
	}
}
