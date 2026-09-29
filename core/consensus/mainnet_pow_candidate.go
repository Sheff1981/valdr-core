package consensus

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/Sheff1981/valdr-core/core/block"
)

var (
	ErrMainnetPoWUnsupportedBlock = errors.New("unsupported block for Mainnet RandomX PoW")
	ErrMainnetPoWHashSize         = errors.New("invalid RandomX proof-of-work hash size")
	ErrMainnetPoWTargetNotMet     = errors.New("RandomX proof-of-work target not met")
)

var mainnetPoWDomainV1 = []byte("VALDR/RANDOMX/POW/V1\x00")

// RandomXHasher is the narrow consensus boundary to a native RandomX engine.
// Implementations MUST deterministically calculate RandomX(key, input) and
// return exactly 32 bytes. The node must never trust a peer-provided PoW hash.
type RandomXHasher interface {
	Hash(key, input []byte) ([]byte, error)
}

// MainnetPoWBlobV1 builds the candidate Mainnet mining blob from the existing
// canonical v2 header. This function does not enable Mainnet.
func MainnetPoWBlobV1(b *block.Block) ([]byte, error) {
	if b == nil || b.Version != block.VersionV2 {
		return nil, ErrMainnetPoWUnsupportedBlock
	}
	header, err := b.HeaderBytesChecked()
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(mainnetPoWDomainV1)+len(header))
	out = append(out, mainnetPoWDomainV1...)
	out = append(out, header...)
	return out, nil
}

func MainnetPoWDomainV1() []byte {
	return bytes.Clone(mainnetPoWDomainV1)
}

// ValidateMainnetRandomXPoW recomputes RandomX locally and compares it to the
// exact unsigned 256-bit target encoded in the block header.
//
// This is a candidate Mainnet consensus path. It is intentionally not wired
// into Testnet2 or any enabled network profile.
func ValidateMainnetRandomXPoW(
	b *block.Block,
	seedKey []byte,
	hasher RandomXHasher,
) ([]byte, error) {
	if b == nil || hasher == nil {
		return nil, ErrMainnetPoWUnsupportedBlock
	}
	target, err := ParseTargetHexV2(b.Target)
	if err != nil {
		return nil, err
	}

	blob, err := MainnetPoWBlobV1(b)
	if err != nil {
		return nil, err
	}
	powHash, err := hasher.Hash(seedKey, blob)
	if err != nil {
		return nil, err
	}
	if len(powHash) != 32 {
		return nil, ErrMainnetPoWHashSize
	}

	value := new(big.Int).SetBytes(powHash)
	if value.Cmp(target) > 0 {
		return nil, fmt.Errorf(
			"%w: pow_hash=%s target=%064x",
			ErrMainnetPoWTargetNotMet,
			hex.EncodeToString(powHash),
			target,
		)
	}
	return append([]byte(nil), powHash...), nil
}
