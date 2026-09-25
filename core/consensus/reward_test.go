package consensus

import (
	"testing"

	"github.com/Sheff1981/valdr-core/config"
)

func TestBlockRewardLegacyCompatibility(t *testing.T) {
	if got := BlockReward(0); got != 0 {
		t.Fatalf("genesis reward = %d, want 0", got)
	}
	if got := BlockReward(1); got != config.InitialMiningReward {
		t.Fatalf("height 1 reward = %d, want %d", got, config.InitialMiningReward)
	}
	if got := BlockReward(100); got != config.InitialMiningReward {
		t.Fatalf("height 100 reward = %d, want %d", got, config.InitialMiningReward)
	}
}

func TestBlockRewardForProfile(t *testing.T) {
	oldTestnet, err := config.ResolveNetworkProfile(config.NetworkTestnetV02)
	if err != nil { t.Fatal(err) }
	activeTestnet, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil { t.Fatal(err) }

	if got := BlockRewardForProfile(activeTestnet, 0); got != 0 {
		t.Fatalf("active Testnet genesis reward=%d want=0", got)
	}
	if got := BlockRewardForProfile(oldTestnet, 1); got != 50*config.AtomicUnitsPerVDR {
		t.Fatalf("historical Testnet reward=%d", got)
	}
	for _, height := range []uint64{1, 10, 100, 1_000_000} {
		if got := BlockRewardForProfile(activeTestnet, height); got != config.AtomicUnitsPerVDR {
			t.Fatalf("height=%d active Testnet reward=%d want=%d", height, got, config.AtomicUnitsPerVDR)
		}
	}
}
