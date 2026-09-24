package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
	"github.com/Sheff1981/valdr-core/mining"
	"github.com/Sheff1981/valdr-core/storage"
)

func TestV2FreshNodeHeadersFirstSync(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}

	sourceChain, err := blockchain.NewForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	targetChain, err := blockchain.NewForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}

	minerKey, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	minerAddress, err := valdrcrypto.AddressFromPublicKey(&minerKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	for i := int64(1); i <= 4; i++ {
		if _, err := mining.MineBlock(
			sourceChain,
			minerAddress,
			profile.GenesisTimestamp+i*profile.TargetBlockTimeSeconds,
			nil,
		); err != nil {
			t.Fatalf("mine source block %d: %v", i, err)
		}
	}

	source := mustStartNode(t, NodeConfig{
		NodeID:         "v2-sync-source",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Blockchain:     sourceChain,
	})
	defer source.Close()

	target := mustStartNode(t, NodeConfig{
		NodeID:         "v2-sync-target",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Blockchain:     targetChain,
	})
	defer target.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := target.Connect(ctx, source.Address()); err != nil {
		t.Fatalf("target connect source: %v", err)
	}

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if targetChain.Height() == sourceChain.Height() &&
			targetChain.Tip() != nil &&
			sourceChain.Tip() != nil &&
			targetChain.Tip().BlockHash == sourceChain.Tip().BlockHash {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf(
		"sync did not converge: target height=%d tip=%v source height=%d tip=%v",
		targetChain.Height(),
		targetChain.Tip(),
		sourceChain.Height(),
		sourceChain.Tip(),
	)
}


func TestV2SyncResumesFromPersistedChainAfterRestart(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}

	sourceChain, err := blockchain.NewForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	minerKey, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	minerAddress, err := valdrcrypto.AddressFromPublicKey(&minerKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	for i := int64(1); i <= 3; i++ {
		if _, err := mining.MineBlock(
			sourceChain,
			minerAddress,
			profile.GenesisTimestamp+i*profile.TargetBlockTimeSeconds,
			nil,
		); err != nil {
			t.Fatalf("mine source block %d: %v", i, err)
		}
	}

	source := mustStartNode(t, NodeConfig{
		NodeID:         "v2-resume-source",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Blockchain:     sourceChain,
	})
	defer source.Close()

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
	target := mustStartNode(t, NodeConfig{
		NodeID:         "v2-resume-target",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Blockchain:     targetChain,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	if err := target.Connect(ctx, source.Address()); err != nil {
		cancel()
		target.Close()
		_ = store.Close()
		t.Fatalf("initial target connect source: %v", err)
	}
	cancel()
	waitForSameTip(t, targetChain, sourceChain, 5*time.Second)

	if err := target.Close(); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	waitForPeerCount(t, source, 0)

	for i := int64(4); i <= 6; i++ {
		if _, err := mining.MineBlock(
			sourceChain,
			minerAddress,
			profile.GenesisTimestamp+i*profile.TargetBlockTimeSeconds,
			nil,
		); err != nil {
			t.Fatalf("mine offline source block %d: %v", i, err)
		}
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

	reopenedChain, err := blockchain.NewPersistentForProfile(reopenedStore, profile)
	if err != nil {
		t.Fatal(err)
	}
	if reopenedChain.Height() != 3 {
		t.Fatalf("reopened height=%d want=3", reopenedChain.Height())
	}

	restarted := mustStartNode(t, NodeConfig{
		NodeID:         "v2-resume-target",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Blockchain:     reopenedChain,
	})
	defer restarted.Close()

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := restarted.Connect(ctx, source.Address()); err != nil {
		t.Fatalf("restarted target connect source: %v", err)
	}
	waitForSameTip(t, reopenedChain, sourceChain, 5*time.Second)
}

func waitForSameTip(
	t *testing.T,
	target *blockchain.Blockchain,
	source *blockchain.Blockchain,
	timeout time.Duration,
) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		targetTip := target.Tip()
		sourceTip := source.Tip()
		if targetTip != nil &&
			sourceTip != nil &&
			target.Height() == source.Height() &&
			targetTip.BlockHash == sourceTip.BlockHash {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf(
		"chains did not converge: target height=%d tip=%v source height=%d tip=%v",
		target.Height(),
		target.Tip(),
		source.Height(),
		source.Tip(),
	)
}


func TestV2HeadersFirstSyncReorgsToGreaterChainwork(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}
	sourceChain, err := blockchain.NewForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	targetChain, err := blockchain.NewForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}

	commonKey, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	commonMiner, err := valdrcrypto.AddressFromPublicKey(&commonKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	sourceKey, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	sourceMiner, err := valdrcrypto.AddressFromPublicKey(&sourceKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	targetKey, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	targetMiner, err := valdrcrypto.AddressFromPublicKey(&targetKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	for i := int64(1); i <= 2; i++ {
		mined, err := mining.MineBlock(
			sourceChain,
			commonMiner,
			profile.GenesisTimestamp+i*profile.TargetBlockTimeSeconds,
			nil,
		)
		if err != nil {
			t.Fatalf("mine common block %d: %v", i, err)
		}
		if err := targetChain.AddBlock(mined); err != nil {
			t.Fatalf("copy common block %d: %v", i, err)
		}
	}

	var oldTargetTip string
	for i := int64(3); i <= 4; i++ {
		mined, err := mining.MineBlock(
			targetChain,
			targetMiner,
			profile.GenesisTimestamp+i*profile.TargetBlockTimeSeconds,
			nil,
		)
		if err != nil {
			t.Fatalf("mine target branch block %d: %v", i, err)
		}
		oldTargetTip = mined.BlockHash
	}

	for i := int64(3); i <= 5; i++ {
		if _, err := mining.MineBlock(
			sourceChain,
			sourceMiner,
			profile.GenesisTimestamp+i*profile.TargetBlockTimeSeconds,
			nil,
		); err != nil {
			t.Fatalf("mine source branch block %d: %v", i, err)
		}
	}

	source := mustStartNode(t, NodeConfig{
		NodeID:         "v2-reorg-source",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Blockchain:     sourceChain,
	})
	defer source.Close()

	target := mustStartNode(t, NodeConfig{
		NodeID:         "v2-reorg-target",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Blockchain:     targetChain,
	})
	defer target.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := target.Connect(ctx, source.Address()); err != nil {
		t.Fatalf("target connect source: %v", err)
	}
	waitForSameTip(t, targetChain, sourceChain, 5*time.Second)

	if _, ok := targetChain.BlockByHash(oldTargetTip); !ok {
		t.Fatal("disconnected target-branch block was not retained as side branch")
	}
	targetBalance, err := targetChain.Balance(targetMiner)
	if err != nil {
		t.Fatal(err)
	}
	if targetBalance != 0 {
		t.Fatalf("disconnected branch miner balance=%d want=0", targetBalance)
	}
	sourceBalance, err := targetChain.Balance(sourceMiner)
	if err != nil {
		t.Fatal(err)
	}
	if sourceBalance != 3*profile.InitialSubsidyVDR*config.AtomicUnitsPerVDR {
		t.Fatalf(
			"source branch miner balance=%d want=%d",
			sourceBalance,
			3*profile.InitialSubsidyVDR*config.AtomicUnitsPerVDR,
		)
	}
}
