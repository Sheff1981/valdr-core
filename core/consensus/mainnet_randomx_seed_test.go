package consensus

import (
	"crypto/sha256"
	"errors"
	"testing"
)

type seedSourceMap map[uint64][sha256.Size]byte

func (s seedSourceMap) BlockIDAtHeight(height uint64) ([sha256.Size]byte, error) {
	id, ok := s[height]
	if !ok {
		return [sha256.Size]byte{}, errors.New("missing ancestor")
	}
	return id, nil
}

func taggedID(tag string) [sha256.Size]byte {
	return sha256.Sum256([]byte(tag))
}

func TestMainnetRandomXSeedHeightBoundaries(t *testing.T) {
	tests := []struct {
		height uint64
		want   uint64
		hist   bool
	}{
		{0, 0, false},
		{63, 0, false},
		{64, 0, true},
		{2111, 0, true},
		{2112, 2048, true},
		{4159, 2048, true},
		{4160, 4096, true},
	}
	for _, tc := range tests {
		got, hist := MainnetRandomXSeedHeight(tc.height)
		if got != tc.want || hist != tc.hist {
			t.Fatalf("height=%d got=(%d,%v) want=(%d,%v)", tc.height, got, hist, tc.want, tc.hist)
		}
	}
}

func TestResolveMainnetRandomXSeedEpoch0(t *testing.T) {
	got, err := ResolveMainnetRandomXSeed(63, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != MainnetEpoch0Seed() {
		t.Fatal("epoch-0 seed mismatch")
	}
}

func TestResolveMainnetRandomXSeedUsesCandidateAncestry(t *testing.T) {
	genesis := taggedID("genesis")
	a2048 := taggedID("branch-a-2048")
	b2048 := taggedID("branch-b-2048")

	branchA := seedSourceMap{0: genesis, 2048: a2048}
	branchB := seedSourceMap{0: genesis, 2048: b2048}

	// Height 64 begins using historical block 0 as the seed.
	got64, err := ResolveMainnetRandomXSeed(64, branchA)
	if err != nil {
		t.Fatal(err)
	}
	if got64 != genesis {
		t.Fatal("height 64 did not use block 0")
	}

	// At height 2112 the epoch advances to block 2048. Competing ancestries
	// must therefore resolve different seeds after a fork crossing that point.
	gotA, err := ResolveMainnetRandomXSeed(2112, branchA)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := ResolveMainnetRandomXSeed(2112, branchB)
	if err != nil {
		t.Fatal(err)
	}
	if gotA != a2048 || gotB != b2048 || gotA == gotB {
		t.Fatal("seed resolver is not branch/reorg aware")
	}
}

func TestResolveMainnetRandomXSeedMissingAncestorFailsClosed(t *testing.T) {
	_, err := ResolveMainnetRandomXSeed(2112, seedSourceMap{})
	if !errors.Is(err, ErrMainnetSeedSource) {
		t.Fatalf("error=%v want ErrMainnetSeedSource", err)
	}
}
