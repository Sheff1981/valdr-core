package consensus

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/core/block"
)

const testMainnetChainID = "valdr-mainnet-1"

type candidateSeedSource struct {
	id [sha256.Size]byte
}

func (s candidateSeedSource) BlockIDAtHeight(height uint64) ([sha256.Size]byte, error) {
	return s.id, nil
}

func mainnetCandidateForValidation(t *testing.T, height uint64, parent, target string) *block.Block {
	t.Helper()
	b, err := block.NewV2(
		height,
		parent,
		1800000000,
		target,
		0x0102030405060708,
		nil,
		testMainnetChainID,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	b.MerkleRoot = strings.Repeat("20", 32)
	b.BlockHash = b.CalculateHash()
	return b
}

func TestValidateMainnetRandomXCandidateChecksIdentityAndLinkage(t *testing.T) {
	raw, _ := hex.DecodeString("01" + strings.Repeat("00", 31))
	h := &fixedRandomXHasher{hash: raw}
	parent := strings.Repeat("11", 32)
	b := mainnetCandidateForValidation(t, 10, parent, strings.Repeat("ff", 32))

	if _, err := ValidateMainnetRandomXCandidate(b, 9, parent, testMainnetChainID, nil, h); err != nil {
		t.Fatalf("valid candidate rejected: %v", err)
	}

	badChain := *b
	badChain.ChainID = "wrong"
	badChain.BlockHash = badChain.CalculateHash()
	if _, err := ValidateMainnetRandomXCandidate(&badChain, 9, parent, testMainnetChainID, nil, h); !errors.Is(err, ErrMainnetCandidateChainID) {
		t.Fatalf("wrong chain id error=%v", err)
	}

	badParent := *b
	if _, err := ValidateMainnetRandomXCandidate(&badParent, 9, strings.Repeat("22", 32), testMainnetChainID, nil, h); !errors.Is(err, ErrMainnetCandidateParent) {
		t.Fatalf("wrong parent error=%v", err)
	}

	badHeight := *b
	if _, err := ValidateMainnetRandomXCandidate(&badHeight, 8, parent, testMainnetChainID, nil, h); !errors.Is(err, ErrMainnetCandidateHeight) {
		t.Fatalf("wrong height error=%v", err)
	}

	badID := *b
	badID.BlockHash = strings.Repeat("00", 32)
	if _, err := ValidateMainnetRandomXCandidate(&badID, 9, parent, testMainnetChainID, nil, h); !errors.Is(err, ErrMainnetCandidateBlockID) {
		t.Fatalf("wrong block id error=%v", err)
	}
}

func TestValidateMainnetRandomXCandidateUsesHistoricalSeed(t *testing.T) {
	parent := strings.Repeat("11", 32)
	b := mainnetCandidateForValidation(t, 2112, parent, strings.Repeat("ff", 32))
	expectedSeed := sha256.Sum256([]byte("branch-seed-2048"))
	src := candidateSeedSource{id: expectedSeed}

	raw, _ := hex.DecodeString("01" + strings.Repeat("00", 31))
	h := &fixedRandomXHasher{hash: raw}

	if _, err := ValidateMainnetRandomXCandidate(b, 2111, parent, testMainnetChainID, src, h); err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(h.key) != hex.EncodeToString(expectedSeed[:]) {
		t.Fatalf("hasher seed=%x want=%x", h.key, expectedSeed)
	}
}
