package storage

import (
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	badger "github.com/dgraph-io/badger/v4"
)

func TestBadgerTransactionAbortLeavesCanonicalStateUntouched(t *testing.T) {
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
	defer store.Close()

	if err := store.Save([]*block.Block{genesis}); err != nil {
		t.Fatal(err)
	}
	before, err := store.Info()
	if err != nil {
		t.Fatal(err)
	}

	sentinel := errors.New("injected transaction failure")
	err = store.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(keyActiveTip, []byte("must-not-commit")); err != nil {
			return err
		}
		if err := txn.Set(keyActiveHeight, encodeUint64(99)); err != nil {
			return err
		}
		if err := txn.Set(keyUTXOHash, []byte("must-not-commit")); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("Update error=%v want injected failure", err)
	}

	after, err := store.Info()
	if err != nil {
		t.Fatal(err)
	}
	if after.ActiveTip != before.ActiveTip ||
		after.Height != before.Height ||
		after.UTXOHash != before.UTXOHash ||
		after.Chainwork != before.Chainwork {
		t.Fatalf("aborted transaction mutated state: before=%+v after=%+v", before, after)
	}
}

func TestBadgerRecoversAfterProcessExitWithoutClose(t *testing.T) {
	if os.Getenv("VALDR_BADGER_CRASH_HELPER") == "1" {
		runBadgerCrashHelper(t)
		return
	}

	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestBadgerRecoversAfterProcessExitWithoutClose$")
	cmd.Env = append(
		os.Environ(),
		"VALDR_BADGER_CRASH_HELPER=1",
		"VALDR_BADGER_CRASH_DIR="+dir,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("crash helper failed: %v\n%s", err, output)
	}

	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewBadgerStore(dir, profile.ChainID, profile.GenesisHash)
	if err != nil {
		t.Fatalf("reopen after abrupt process exit: %v", err)
	}
	defer store.Close()

	info, err := store.Info()
	if err != nil {
		t.Fatal(err)
	}
	if info.Height != 0 ||
		info.ActiveTip != profile.GenesisHash ||
		info.GenesisHash != profile.GenesisHash ||
		info.Network != profile.ChainID ||
		info.UTXOHash != HashUTXOSet(nil) {
		t.Fatalf("unexpected recovered state: %+v", info)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].BlockHash != profile.GenesisHash {
		t.Fatalf("recovered chain len/tip=%d/%v", len(loaded), loaded)
	}
}

func runBadgerCrashHelper(t *testing.T) {
	t.Helper()

	dir := os.Getenv("VALDR_BADGER_CRASH_DIR")
	if dir == "" {
		t.Fatal("VALDR_BADGER_CRASH_DIR is empty")
	}
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	genesis, err := block.NewGenesisForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewBadgerStore(dir, profile.ChainID, profile.GenesisHash)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save([]*block.Block{genesis}); err != nil {
		t.Fatal(err)
	}

	// Deliberately bypass Close to simulate sudden process termination.
	os.Exit(0)
}
