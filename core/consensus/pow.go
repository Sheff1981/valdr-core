package consensus

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
)

var (
	ErrInvalidDifficulty = errors.New("difficulty must be greater than zero")
	ErrInvalidHash       = errors.New("invalid proof-of-work hash")
	ErrInvalidPoW        = errors.New("proof of work target not met")
	ErrNonceExhausted    = errors.New("nonce space exhausted")
)

// PowLimit returns the easiest target allowed on the v0.1 devnet.
// The MVP devnet uses a 12-leading-zero-bit limit so tests remain fast
// while mining still performs real nonce search work.
func PowLimit() *big.Int {
	limit := new(big.Int).Lsh(big.NewInt(1), 256-config.PowLimitLeadingZeroBits)
	return limit.Sub(limit, big.NewInt(1))
}

// TargetForDifficulty converts the integer difficulty multiplier into
// a 256-bit target. Higher difficulty always produces a smaller target.
func TargetForDifficulty(difficulty uint64) (*big.Int, error) {
	if difficulty == 0 {
		return nil, ErrInvalidDifficulty
	}
	target := PowLimit()
	target.Div(target, new(big.Int).SetUint64(difficulty))
	return target, nil
}

// TargetHex returns the target as a fixed-width 256-bit hexadecimal string.
func TargetHex(difficulty uint64) (string, error) {
	target, err := TargetForDifficulty(difficulty)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%064x", target), nil
}

// ValidatePoW checks that the block hash is a valid 256-bit value
// and is less than or equal to the target for the block difficulty.
func ValidatePoW(b *block.Block) error {
	if b == nil {
		return ErrInvalidHash
	}

	target, err := TargetForDifficulty(b.Difficulty)
	if err != nil {
		return err
	}

	raw, err := hex.DecodeString(b.BlockHash)
	if err != nil || len(raw) != 32 {
		return ErrInvalidHash
	}

	value := new(big.Int).SetBytes(raw)
	if value.Cmp(target) > 0 {
		return fmt.Errorf("%w: hash=%s target=%064x", ErrInvalidPoW, b.BlockHash, target)
	}

	return nil
}

// Mine searches the uint64 nonce space until the block hash satisfies
// the target for the block difficulty.
func Mine(b *block.Block) error {
	if b == nil {
		return ErrInvalidHash
	}
	if b.Difficulty == 0 {
		return ErrInvalidDifficulty
	}

	for nonce := uint64(0); ; nonce++ {
		b.Nonce = nonce
		b.BlockHash = b.CalculateHash()

		if err := ValidatePoW(b); err == nil {
			return nil
		}

		if nonce == math.MaxUint64 {
			return ErrNonceExhausted
		}
	}
}

// NextDifficulty is the simplified v0.1 devnet retarget algorithm.
// It compares the latest block interval with the 60-second target and
// limits each step to at most a 4x increase or decrease.
func NextDifficulty(previousDifficulty uint64, previousTimestamp, candidateTimestamp int64) uint64 {
	if previousDifficulty == 0 {
		previousDifficulty = 1
	}

	elapsed := candidateTimestamp - previousTimestamp
	minElapsed := config.TargetBlockTimeSeconds / int64(config.DifficultyAdjustmentClamp)
	maxElapsed := config.TargetBlockTimeSeconds * int64(config.DifficultyAdjustmentClamp)

	if elapsed < minElapsed {
		elapsed = minElapsed
	}
	if elapsed > maxElapsed {
		elapsed = maxElapsed
	}

	next := new(big.Int).Mul(
		new(big.Int).SetUint64(previousDifficulty),
		big.NewInt(config.TargetBlockTimeSeconds),
	)
	next.Div(next, big.NewInt(elapsed))
	if next.Sign() < 1 {
		return 1
	}

	max := new(big.Int).Mul(
		new(big.Int).SetUint64(previousDifficulty),
		new(big.Int).SetUint64(config.DifficultyAdjustmentClamp),
	)
	min := new(big.Int).Div(
		new(big.Int).SetUint64(previousDifficulty),
		new(big.Int).SetUint64(config.DifficultyAdjustmentClamp),
	)
	if min.Sign() < 1 {
		min.SetUint64(1)
	}

	if next.Cmp(max) > 0 {
		next = max
	}
	if next.Cmp(min) < 0 {
		next = min
	}
	if !next.IsUint64() {
		return math.MaxUint64
	}

	return next.Uint64()
}
