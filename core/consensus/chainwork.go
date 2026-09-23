package consensus

import (
	"fmt"
	"math/big"
)

var twoTo256 = new(big.Int).Lsh(big.NewInt(1), 256)

// BlockWork returns the consensus work represented by one block:
//
//   floor(2^256 / (target + 1))
//
// The returned integer is independent of the observed block hash. Only the
// validated target implied by the block difficulty contributes to chainwork.
func BlockWork(difficulty uint64) (*big.Int, error) {
	target, err := TargetForDifficulty(difficulty)
	if err != nil {
		return nil, err
	}
	denominator := new(big.Int).Add(target, big.NewInt(1))
	if denominator.Sign() <= 0 {
		return nil, ErrInvalidDifficulty
	}
	return BlockWorkTarget(target)
}

func BlockWorkTarget(target *big.Int) (*big.Int, error) {
	if target == nil || target.Sign() <= 0 || target.BitLen() > 256 {
		return nil, ErrInvalidTarget
	}
	denominator := new(big.Int).Add(new(big.Int).Set(target), big.NewInt(1))
	work := new(big.Int).Div(new(big.Int).Set(twoTo256), denominator)
	if work.Sign() <= 0 {
		return nil, fmt.Errorf("%w: zero block work", ErrInvalidTarget)
	}
	return work, nil
}

func AddTargetWork(parent, target *big.Int) (*big.Int, error) {
	blockWork, err := BlockWorkTarget(target)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return blockWork, nil
	}
	return new(big.Int).Add(new(big.Int).Set(parent), blockWork), nil
}

func AddWork(parent *big.Int, difficulty uint64) (*big.Int, error) {
	blockWork, err := BlockWork(difficulty)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return blockWork, nil
	}
	return new(big.Int).Add(new(big.Int).Set(parent), blockWork), nil
}

func ChainworkHex(work *big.Int) string {
	if work == nil || work.Sign() < 0 {
		return ""
	}
	return fmt.Sprintf("%064x", work)
}
