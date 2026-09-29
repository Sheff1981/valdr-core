package consensus

import (
	"errors"
	"fmt"

	"github.com/Sheff1981/valdr-core/core/block"
)

const MainnetCandidateChainID = "valdr-mainnet-1"

var (
	ErrMainnetCandidateChainID = errors.New("Mainnet candidate has wrong chain id")
	ErrMainnetCandidateParent  = errors.New("Mainnet candidate parent mismatch")
	ErrMainnetCandidateHeight  = errors.New("Mainnet candidate height mismatch")
	ErrMainnetCandidateBlockID = errors.New("Mainnet candidate block id mismatch")
)

// ValidateMainnetRandomXCandidate validates the candidate block identity,
// parent linkage and RandomX PoW against the exact candidate ancestry.
//
// It intentionally does not enable a Mainnet network profile. DAA, monetary
// policy and full transaction/UTXO validation remain separate gates.
func ValidateMainnetRandomXCandidate(
	candidate *block.Block,
	parentHeight uint64,
	parentHash string,
	source MainnetSeedSource,
	hasher RandomXHasher,
) ([]byte, error) {
	if candidate == nil || candidate.Version != block.VersionV2 || candidate.Difficulty != 0 {
		return nil, ErrMainnetPoWUnsupportedBlock
	}
	if candidate.ChainID != MainnetCandidateChainID {
		return nil, fmt.Errorf("%w: got %q want %q", ErrMainnetCandidateChainID, candidate.ChainID, MainnetCandidateChainID)
	}
	if candidate.PreviousBlockHash != parentHash || parentHash == "" {
		return nil, fmt.Errorf("%w: got %q want %q", ErrMainnetCandidateParent, candidate.PreviousBlockHash, parentHash)
	}
	if candidate.Height != parentHeight+1 {
		return nil, fmt.Errorf("%w: got %d want %d", ErrMainnetCandidateHeight, candidate.Height, parentHeight+1)
	}

	expectedID := candidate.CalculateHash()
	if expectedID == "" || candidate.BlockHash != expectedID {
		return nil, fmt.Errorf("%w: got %q want %q", ErrMainnetCandidateBlockID, candidate.BlockHash, expectedID)
	}

	return ValidateMainnetRandomXPoWForHeight(candidate, source, hasher)
}
