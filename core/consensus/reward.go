package consensus

import "github.com/Sheff1981/valdr-core/config"

// BlockReward returns the current v0.1 devnet block subsidy.
// The halving interval is intentionally not frozen yet in the master specification.
func BlockReward(height uint64) uint64 {
	if height == 0 {
		return 0
	}
	return config.InitialMiningReward
}
