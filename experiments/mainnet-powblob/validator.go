package powblob

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/Sheff1981/valdr-core/core/block"
)

var (
	ErrInvalidPoWHash = errors.New("invalid RandomX proof-of-work hash")
	ErrInvalidTarget  = errors.New("invalid proof-of-work target")
	ErrTargetNotMet   = errors.New("RandomX proof-of-work target not met")
)

// ValidatePoWHash validates an already-computed RandomX hash against the
// candidate block's exact unsigned 256-bit target.
//
// RandomX execution itself remains outside this pure-Go helper so the target
// comparison can be tested independently from the native RandomX runtime.
func ValidatePoWHash(b *block.Block, powHash []byte) error {
	if b == nil || len(powHash) != 32 {
		return ErrInvalidPoWHash
	}
	if b.Version != block.VersionV2 {
		return ErrInvalidTarget
	}

	targetBytes, err := block.DecodeTarget(b.Target)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTarget, err)
	}

	target := new(big.Int).SetBytes(targetBytes)
	if target.Sign() <= 0 {
		return ErrInvalidTarget
	}

	value := new(big.Int).SetBytes(powHash)
	if value.Cmp(target) > 0 {
		return fmt.Errorf(
			"%w: pow_hash=%s target=%s",
			ErrTargetNotMet,
			hex.EncodeToString(powHash),
			b.Target,
		)
	}
	return nil
}
