package p2p

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
	"github.com/Sheff1981/valdr-core/mining"
	"github.com/Sheff1981/valdr-core/storage"
)

func TestTestnet2ChainworkReorgPersistsAfterRestart(t *testing.T) {
	if os.Getenv("VALDR_TESTNET2_RUNTIME") != "1" {
		t.Skip("set VALDR_TESTNET2_RUNTIME=1 for the dedicated Testnet2 runtime gate")
	}

	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}

	sourceChain, err := blockchain.NewForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}

	dataDir := t.TempDir()
	store, err := storage.NewBadgerStore(
		dataDir,
		profile.ChainID,
		profile.GenesisHash,
	)
	if err != nil {
		t.Fatal(err)
	}
	targetChain, err := blockchain.NewPersistentForProfile(store, profile)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}

	sourceKey, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	sourceMiner, err := valdrcrypto.AddressFromPublicKey(&sourceKey.PublicKey)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	targetKey, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	targetMiner, err := valdrcrypto.AddressFromPublicKey(&targetKey.PublicKey)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}

	for i := int64(1); i <= 2; i++ {
		if _, err := mining.MineBlock(
			targetChain,
			targetMiner,
			profile.GenesisTimestamp+i*profile.TargetBlockTimeSeconds,
			nil,
		); err != nil {
			_ = store.Close()
			t.Fatalf("mine target branch block %d: %v", i, err)
		}
	}
	oldTargetTip := targetChain.Tip().BlockHash
	oldTargetWork := targetChain.Chainwork()

	for i := int64(1); i <= 3; i++ {
		if _, err := mining.MineBlock(
			sourceChain,
			sourceMiner,
			profile.GenesisTimestamp+i*profile.TargetBlockTimeSeconds,
			nil,
		); err != nil {
			_ = store.Close()
			t.Fatalf("mine source branch block %d: %v", i, err)
		}
	}
	if sourceChain.Chainwork() == oldTargetWork {
		_ = store.Close()
		t.Fatal("source chainwork unexpectedly equals shorter target branch")
	}

	source := mustStartNode(t, NodeConfig{
		NodeID: "testnet2-reorg-source", ListenAddress: "127.0.0.1:0",
		NetworkProfile: &profile, EnableV2: true, Blockchain: sourceChain,
	})
	defer source.Close()
	target := mustStartNode(t, NodeConfig{
		NodeID: "testnet2-reorg-target", ListenAddress: "127.0.0.1:0",
		NetworkProfile: &profile, EnableV2: true, Blockchain: targetChain,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := target.Connect(ctx, source.Address()); err != nil {
		cancel()
		_ = target.Close()
		_ = store.Close()
		t.Fatalf("connect target to source: %v", err)
	}
	cancel()
	waitForSameTip(t, targetChain, sourceChain, 15*time.Second)

	if targetChain.Height() != 3 ||
		targetChain.Tip().BlockHash != sourceChain.Tip().BlockHash ||
		targetChain.Chainwork() != sourceChain.Chainwork() {
		_ = target.Close()
		_ = store.Close()
		t.Fatalf(
			"reorg result height/tip/work=%d/%s/%s source=%d/%s/%s",
			targetChain.Height(),
			targetChain.Tip().BlockHash,
			targetChain.Chainwork(),
			sourceChain.Height(),
			sourceChain.Tip().BlockHash,
			sourceChain.Chainwork(),
		)
	}
	if _, ok := targetChain.BlockByHash(oldTargetTip); !ok {
		_ = target.Close()
		_ = store.Close()
		t.Fatal("disconnected Testnet2 branch was not retained")
	}
	targetBalance, err := targetChain.Balance(targetMiner)
	if err != nil {
		_ = target.Close()
		_ = store.Close()
		t.Fatal(err)
	}
	if targetBalance != 0 {
		_ = target.Close()
		_ = store.Close()
		t.Fatalf("disconnected target miner balance=%d want=0", targetBalance)
	}
	sourceBalance, err := targetChain.Balance(sourceMiner)
	if err != nil {
		_ = target.Close()
		_ = store.Close()
		t.Fatal(err)
	}
	wantSource := uint64(3) * profile.InitialSubsidyVDR * config.AtomicUnitsPerVDR
	if sourceBalance != wantSource {
		_ = target.Close()
		_ = store.Close()
		t.Fatalf("source miner balance=%d want=%d", sourceBalance, wantSource)
	}

	if err := target.Close(); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopenedStore, err := storage.NewBadgerStore(
		dataDir,
		profile.ChainID,
		profile.GenesisHash,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer reopenedStore.Close()
	reopened, err := blockchain.NewPersistentForProfile(reopenedStore, profile)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Height() != 3 ||
		reopened.Tip().BlockHash != sourceChain.Tip().BlockHash ||
		reopened.Chainwork() != sourceChain.Chainwork() {
		t.Fatalf(
			"reopened Testnet2 height/tip/work=%d/%s/%s source=%d/%s/%s",
			reopened.Height(),
			reopened.Tip().BlockHash,
			reopened.Chainwork(),
			sourceChain.Height(),
			sourceChain.Tip().BlockHash,
			sourceChain.Chainwork(),
		)
	}
	if _, ok := reopened.BlockByHash(oldTargetTip); !ok {
		t.Fatal("restarted database lost the disconnected Testnet2 branch")
	}
	reopenedTargetBalance, err := reopened.Balance(targetMiner)
	if err != nil {
		t.Fatal(err)
	}
	if reopenedTargetBalance != 0 {
		t.Fatalf("reopened disconnected miner balance=%d want=0", reopenedTargetBalance)
	}
	reopenedSourceBalance, err := reopened.Balance(sourceMiner)
	if err != nil {
		t.Fatal(err)
	}
	if reopenedSourceBalance != wantSource {
		t.Fatalf("reopened source miner balance=%d want=%d", reopenedSourceBalance, wantSource)
	}
}
