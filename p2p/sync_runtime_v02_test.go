package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
	"github.com/Sheff1981/valdr-core/mining"
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
