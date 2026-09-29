package powblob

import "crypto/sha256"

const (
	SeedEpochBlocks = uint64(2048)
	SeedLagBlocks   = uint64(64)
)

var epoch0Seed = sha256.Sum256([]byte("VALDR/RandomX/Mainnet/v1/epoch-0"))

func Epoch0Seed() [sha256.Size]byte {
	return epoch0Seed
}

// SeedHeight returns the historical block height whose block ID is used as the
// RandomX seed key. Heights before SeedLagBlocks use the fixed epoch-0 seed.
func SeedHeight(height uint64) (uint64, bool) {
	if height < SeedLagBlocks {
		return 0, false
	}
	return ((height - SeedLagBlocks) / SeedEpochBlocks) * SeedEpochBlocks, true
}
