package storage

import (
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
	"github.com/Sheff1981/valdr-core/mining"
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


func TestCommitCandidateFailureIsFullyAtomic(t *testing.T) {
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
	beforeInfo, err := store.Info()
	if err != nil {
		t.Fatal(err)
	}

	memoryChain, err := blockchain.NewForProfile(profile)
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
	candidate, err := mining.MineBlock(
		memoryChain,
		minerAddress,
		profile.GenesisTimestamp+profile.TargetBlockTimeSeconds,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Deliberately supply an invalid active chain that does not end at the
	// candidate. CommitCandidate writes candidate records first, so this proves
	// the surrounding Badger transaction rolls all of them back on later error.
	err = store.CommitCandidate(
		candidate,
		nil,
		memoryChain.UTXOSnapshot(),
		memoryChain.Chainwork(),
		[]*block.Block{genesis},
		memoryChain.UTXOSnapshot(),
	)
	if !errors.Is(err, ErrStorageStateMismatch) {
		t.Fatalf("CommitCandidate error=%v want ErrStorageStateMismatch", err)
	}

	afterInfo, err := store.Info()
	if err != nil {
		t.Fatal(err)
	}
	if afterInfo != beforeInfo {
		t.Fatalf("failed candidate mutated active metadata: before=%+v after=%+v", beforeInfo, afterInfo)
	}
	all, err := store.LoadAllBlocks()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].BlockHash != genesis.BlockHash {
		t.Fatalf("failed candidate leaked into block store: %+v", all)
	}
	for _, prefix := range []string{"block/", "header/", "undo/", "height/", "tx/", "utxo/"} {
		count, err := store.KeyCount(prefix)
		if err != nil {
			t.Fatal(err)
		}
		want := 0
		switch prefix {
		case "block/", "header/", "height/":
			want = 1
		}
		if count != want {
			t.Fatalf("%s key count=%d want=%d after failed candidate", prefix, count, want)
		}
	}
}
