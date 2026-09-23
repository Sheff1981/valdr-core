package consensus

import (
	"testing"

	"github.com/Sheff1981/valdr-core/config"
)

func TestBlockReward(t *testing.T) {
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
