package consensus

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/core/block"
)

type boundarySeedSource map[uint64][sha256.Size]byte

func (s boundarySeedSource) BlockIDAtHeight(height uint64) ([sha256.Size]byte, error) {
	return s[height], nil
}

func boundaryCandidate(t *testing.T, height uint64, parent string) *block.Block {
	t.Helper()
	b, err := block.NewV2(
		height,
		parent,
		1800000000+int64(height),
		strings.Repeat("ff", 32),
		height+1,
		nil,
		MainnetCandidateChainID,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	b.BlockHash = b.CalculateHash()
	return b
}

func lowFixedHash() []byte {
	raw, _ := hex.DecodeString("01" + strings.Repeat("00", 31))
	return raw
}

func TestMainnetRandomXSeedBoundaryBeforeEpochSwitch(t *testing.T) {
	parent := strings.Repeat("11", 32)
	block0 := sha256.Sum256([]byte("block-0"))

	src := boundarySeedSource{0: block0}
	h := &fixedRandomXHasher{hash: lowFixedHash()}
	b := boundaryCandidate(t, 2111, parent)

	if _, err := ValidateMainnetRandomXCandidate(b, 2110, parent, src, h); err != nil {
		t.Fatal(err)
	}
	if string(h.key) != string(block0[:]) {
		t.Fatalf("height 2111 seed=%x want block0=%x", h.key, block0)
	}
}

func TestMainnetRandomXSeedBoundaryAtEpochSwitchUsesBranch2048(t *testing.T) {
	parentA := strings.Repeat("22", 32)
	parentB := strings.Repeat("33", 32)
	seedA := sha256.Sum256([]byte("branch-a-block-2048"))
	seedB := sha256.Sum256([]byte("branch-b-block-2048"))

	candidateA := boundaryCandidate(t, 2112, parentA)
	candidateB := boundaryCandidate(t, 2112, parentB)

	hA := &fixedRandomXHasher{hash: lowFixedHash()}
	if _, err := ValidateMainnetRandomXCandidate(
		candidateA,
		2111,
		parentA,
		boundarySeedSource{2048: seedA},
		hA,
	); err != nil {
		t.Fatal(err)
	}

	hB := &fixedRandomXHasher{hash: lowFixedHash()}
	if _, err := ValidateMainnetRandomXCandidate(
		candidateB,
		2111,
		parentB,
		boundarySeedSource{2048: seedB},
		hB,
	); err != nil {
		t.Fatal(err)
	}

	if string(hA.key) != string(seedA[:]) {
		t.Fatalf("branch A seed=%x want=%x", hA.key, seedA)
	}
	if string(hB.key) != string(seedB[:]) {
		t.Fatalf("branch B seed=%x want=%x", hB.key, seedB)
	}
	if string(hA.key) == string(hB.key) {
		t.Fatal("competing branches collapsed to the same RandomX seed")
	}
}

func TestMainnetRandomXSeedBoundaryAtNextEpochSwitch(t *testing.T) {
	parent := strings.Repeat("44", 32)
	seed4096 := sha256.Sum256([]byte("block-4096"))
	src := boundarySeedSource{4096: seed4096}
	h := &fixedRandomXHasher{hash: lowFixedHash()}
	b := boundaryCandidate(t, 4160, parent)

	if _, err := ValidateMainnetRandomXCandidate(b, 4159, parent, src, h); err != nil {
		t.Fatal(err)
	}
	if string(h.key) != string(seed4096[:]) {
		t.Fatalf("height 4160 seed=%x want block4096=%x", h.key, seed4096)
	}
}
