package consensus

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/Sheff1981/valdr-core/core/block"
)

const (
	MainnetRandomXSeedEpochBlocks = uint64(2048)
	MainnetRandomXSeedLagBlocks   = uint64(64)
)

var (
	ErrMainnetSeedSource = errors.New("invalid Mainnet RandomX seed source")
	mainnetEpoch0Seed     = sha256.Sum256([]byte("VALDR/RandomX/Mainnet/v1/epoch-0"))
)

// MainnetSeedSource must resolve block IDs from the exact ancestry being
// validated, not blindly from the currently active chain. This is required for
// deterministic validation during competing forks and reorgs.
type MainnetSeedSource interface {
	BlockIDAtHeight(height uint64) ([sha256.Size]byte, error)
}

func MainnetEpoch0Seed() [sha256.Size]byte {
	return mainnetEpoch0Seed
}

// MainnetRandomXSeedHeight returns the historical block height used as the
// RandomX seed for a candidate block.
//
// Heights 0..63 use the fixed epoch-0 seed. Starting at height 64, the seed is
// the block ID at floor((height-64)/2048)*2048 in the candidate's ancestry.
func MainnetRandomXSeedHeight(height uint64) (uint64, bool) {
	if height < MainnetRandomXSeedLagBlocks {
		return 0, false
	}
	return ((height - MainnetRandomXSeedLagBlocks) / MainnetRandomXSeedEpochBlocks) *
		MainnetRandomXSeedEpochBlocks, true
}

func ResolveMainnetRandomXSeed(height uint64, source MainnetSeedSource) ([sha256.Size]byte, error) {
	if seedHeight, historical := MainnetRandomXSeedHeight(height); historical {
		if source == nil {
			return [sha256.Size]byte{}, ErrMainnetSeedSource
		}
		id, err := source.BlockIDAtHeight(seedHeight)
		if err != nil {
			return [sha256.Size]byte{}, fmt.Errorf("%w: height=%d: %v", ErrMainnetSeedSource, seedHeight, err)
		}
		return id, nil
	}
	return MainnetEpoch0Seed(), nil
}

// ValidateMainnetRandomXPoWForHeight resolves the seed from the exact candidate
// ancestry and then performs local RandomX validation.
//
// This is still a disabled Mainnet candidate path; Testnet2 does not call it.
func ValidateMainnetRandomXPoWForHeight(
	b *block.Block,
	source MainnetSeedSource,
	hasher RandomXHasher,
) ([]byte, error) {
	if b == nil {
		return nil, ErrMainnetPoWUnsupportedBlock
	}
	seed, err := ResolveMainnetRandomXSeed(b.Height, source)
	if err != nil {
		return nil, err
	}
	return ValidateMainnetRandomXPoW(b, seed[:], hasher)
}
