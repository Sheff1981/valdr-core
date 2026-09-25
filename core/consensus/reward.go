package consensus

import (
	"math"

	"github.com/Sheff1981/valdr-core/config"
)

// BlockReward preserves the frozen legacy v0.1 subsidy behavior.
func BlockReward(height uint64) uint64 {
	if height == 0 {
		return 0
	}
	return config.InitialMiningReward
}

// BlockRewardForProfile returns the subsidy for a network profile.
// Testnet v0.2.9 deliberately uses a fixed 1 Testnet VDR subsidy.
// This does not freeze Mainnet emission or halving rules.
func BlockRewardForProfile(profile config.NetworkProfile, height uint64) uint64 {
	if height == 0 || profile.InitialSubsidyVDR == 0 {
		return 0
	}
	if profile.InitialSubsidyVDR > math.MaxUint64/config.AtomicUnitsPerVDR {
		return 0
	}
	return profile.InitialSubsidyVDR * config.AtomicUnitsPerVDR
}
