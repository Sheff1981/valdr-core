package config

const (
	ChainID                   = "valdr-devnet-1"
	GenesisVersion            = uint32(1)
	GenesisTimestamp          = int64(1790121600) // 2026-09-23 00:00:00 UTC
	GenesisDifficulty         = uint64(1)
	GenesisNonce              = uint64(0)
	GenesisMessage            = "VALDR genesis block | valdr-devnet-1 | 2026-09-23"
	GenesisBlockHash          = "47e3a6c15cab1a41c54a36a65f7133261fa6f75976a2e59825694e001716bfe5"
	TargetBlockTimeSeconds    = int64(60)
	PowLimitLeadingZeroBits   = uint(12)
	DifficultyAdjustmentClamp = uint64(4)
)
