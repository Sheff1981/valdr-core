package powblob

import (
	"bytes"
	"crypto/sha256"
	"errors"

	"github.com/Sheff1981/valdr-core/core/block"
)

var (
	ErrNilBlock      = errors.New("nil block")
	ErrHeaderEncoding = errors.New("invalid block header encoding")
)

var domainV1 = []byte("VALDR/RANDOMX/POW/V1\x00")

// BuildV1 returns the candidate Mainnet RandomX mining blob.
//
// Design rule: do not invent a second header serialization. The mining blob is
// a fixed domain separator followed by the existing canonical v2 block header.
// This keeps block identity and PoW separate while preserving one canonical
// header encoding:
//   block_id = SHA256(header)
//   pow_hash = RandomX(seed_key, domain || header)
func BuildV1(b *block.Block) ([]byte, error) {
	if b == nil {
		return nil, ErrNilBlock
	}
	if b.Version != block.VersionV2 {
		return nil, ErrHeaderEncoding
	}
	header, err := b.HeaderBytesChecked()
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(domainV1)+len(header))
	out = append(out, domainV1...)
	out = append(out, header...)
	return out, nil
}

func DomainV1() []byte {
	return bytes.Clone(domainV1)
}

// BlockID returns the existing inexpensive canonical SHA-256 header identifier.
func BlockID(b *block.Block) ([sha256.Size]byte, error) {
	var zero [sha256.Size]byte
	if b == nil {
		return zero, ErrNilBlock
	}
	header, err := b.HeaderBytesChecked()
	if err != nil {
		return zero, err
	}
	return sha256.Sum256(header), nil
}
