package storage

import (
	"math/big"
	"testing"

	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus"
)

func TestLoadMainnetDAAHistoryFollowsExactBranch(t *testing.T) {
	s, err := OpenBadgerStore(t.TempDir(), "valdr-mainnet-1", "genesis-placeholder")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	targetHex := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	g := makeStoredBranchBlock(t, 0, "", "g")
	g.Target = targetHex
	g.BlockHash = g.CalculateHash()
	a1 := makeStoredBranchBlock(t, 1, g.BlockHash, "a1")
	a1.Target = targetHex
	a1.BlockHash = a1.CalculateHash()
	b1 := makeStoredBranchBlock(t, 1, g.BlockHash, "b1")
	b1.Target = targetHex
	b1.BlockHash = b1.CalculateHash()
	a2 := makeStoredBranchBlock(t, 2, a1.BlockHash, "a2")
	a2.Target = targetHex
	a2.BlockHash = a2.CalculateHash()
	b2 := makeStoredBranchBlock(t, 2, b1.BlockHash, "b2")
	b2.Target = targetHex
	b2.BlockHash = b2.CalculateHash()

	for _, b := range []*block.Block{g, a1, b1, a2, b2} {
		putRandomXBranchBlock(t, s, b)
	}

	hA, err := s.LoadMainnetDAAHistory(a2.BlockHash, 3)
	if err != nil {
		t.Fatal(err)
	}
	hB, err := s.LoadMainnetDAAHistory(b2.BlockHash, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(hA) != 3 || len(hB) != 3 {
		t.Fatalf("history lengths A=%d B=%d", len(hA), len(hB))
	}
	if hA[2].Height != 2 || hB[2].Height != 2 {
		t.Fatal("branch tips missing")
	}
	if a2.BlockHash == b2.BlockHash {
		t.Fatal("fixture branches did not diverge")
	}
}

func TestValidateStoredMainnetConsensusCandidateM4(t *testing.T) {
	s, err := OpenBadgerStore(t.TempDir(), "valdr-mainnet-1", "genesis-placeholder")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	initial := new(big.Int)
	if _, ok := initial.SetString("0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16); !ok {
		t.Fatal("bad initial target")
	}
	powLimit := new(big.Int)
	if _, ok := powLimit.SetString("00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16); !ok {
		t.Fatal("bad pow limit")
	}
	targetHex, err := consensus.TargetHexV2(initial)
	if err != nil {
		t.Fatal(err)
	}

	parent := ""
	var tip *block.Block
	for height := uint64(0); height < 31; height++ {
		b := makeStoredBranchBlock(t, height, parent, "m4")
		b.Timestamp = 1800000000 + int64(height)*600
		b.Target = targetHex
		b.BlockHash = b.CalculateHash()
		putRandomXBranchBlock(t, s, b)
		parent = b.BlockHash
		tip = b
	}

	history, err := s.LoadMainnetDAAHistory(tip.BlockHash, consensus.MainnetDAAWindow+1)
	if err != nil {
		t.Fatal(err)
	}
	nextTarget, err := consensus.MainnetNextTargetLWMA(history, initial, powLimit)
	if err != nil {
		t.Fatal(err)
	}
	nextTargetHex, err := consensus.TargetHexV2(nextTarget)
	if err != nil {
		t.Fatal(err)
	}

	candidate := makeStoredBranchBlock(t, 31, tip.BlockHash, "candidate")
	candidate.Timestamp = tip.Timestamp + 600
	candidate.Target = nextTargetHex
	candidate.BlockHash = candidate.CalculateHash()

	h := &fixedStorageRandomXHasher{hash: lowStorageHash()}
	got, err := s.ValidateStoredMainnetConsensusCandidate(
		candidate,
		h,
		initial,
		powLimit,
		candidate.Timestamp,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 32 {
		t.Fatalf("pow hash length=%d", len(got))
	}
}
