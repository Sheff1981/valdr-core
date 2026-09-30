package consensus

import (
	"fmt"
	"math/big"

	"github.com/Sheff1981/valdr-core/core/block"
)

var ErrMainnetCandidateTarget = fmt.Errorf("Mainnet candidate target mismatch")

// ValidateMainnetConsensusCandidate binds the M3 RandomX candidate path to the
// M4 LWMA/timestamp candidate rules. InitialTarget and PowLimit are explicit
// inputs until M5 calibration freezes them.
//
// The supplied history must be the exact parent branch ancestry ending at the
// candidate's parent.
func ValidateMainnetConsensusCandidate(
	candidate *block.Block,
	history []MainnetDAAHeader,
	parentHash string,
	expectedChainID string,
	initialTarget *big.Int,
	powLimit *big.Int,
	localSystemTime int64,
	source MainnetSeedSource,
	hasher RandomXHasher,
) ([]byte, error) {
	if candidate == nil {
		return nil, ErrMainnetPoWUnsupportedBlock
	}
	if len(history) == 0 {
		return nil, ErrMainnetDAAHistory
	}
	parent := history[len(history)-1]
	if parentHash == "" || candidate.PreviousBlockHash != parentHash {
		return nil, fmt.Errorf("%w: got %q want %q", ErrMainnetCandidateParent, candidate.PreviousBlockHash, parentHash)
	}
	if candidate.Height != parent.Height+1 {
		return nil, fmt.Errorf("%w: got %d want %d", ErrMainnetCandidateHeight, candidate.Height, parent.Height+1)
	}

	if err := ValidateMainnetTimestampCandidate(history, candidate.Timestamp, localSystemTime); err != nil {
		return nil, err
	}

	expectedTarget, err := MainnetNextTargetLWMA(history, initialTarget, powLimit)
	if err != nil {
		return nil, err
	}
	gotTarget, err := ParseTargetHexV2(candidate.Target)
	if err != nil {
		return nil, err
	}
	if gotTarget.Cmp(expectedTarget) != 0 {
		return nil, fmt.Errorf(
			"%w: got=%064x want=%064x",
			ErrMainnetCandidateTarget,
			gotTarget,
			expectedTarget,
		)
	}

	return ValidateMainnetRandomXCandidate(
		candidate,
		parent.Height,
		parentHash,
		expectedChainID,
		source,
		hasher,
	)
}
