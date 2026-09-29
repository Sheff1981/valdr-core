package storage

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/Sheff1981/valdr-core/core/block"
	badger "github.com/dgraph-io/badger/v4"
)

func putRandomXBranchBlock(t *testing.T, s *BadgerStore, b *block.Block) {
	t.Helper()
	rawBlock, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	rawHeader, err := json.Marshal(headerRecord{
		Parent: b.PreviousBlockHash,
		Height: b.Height,
		Status: "side",
		Target: b.Target,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(blockKey(b.BlockHash), rawBlock); err != nil {
			return err
		}
		return txn.Set(headerKey(b.BlockHash), rawHeader)
	}); err != nil {
		t.Fatal(err)
	}
}

func makeStoredBranchBlock(t *testing.T, height uint64, parent, tag string) *block.Block {
	t.Helper()
	b, err := block.NewV2(
		height,
		parent,
		1800000000+int64(height),
		"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		height+1,
		nil,
		"valdr-mainnet-1",
		tag,
	)
	if err != nil {
		t.Fatal(err)
	}
	b.BlockHash = b.CalculateHash()
	return b
}

func TestRandomXAncestrySourceFollowsAnchoredBranch(t *testing.T) {
	path := t.TempDir()
	s, err := OpenBadgerStore(path, "valdr-mainnet-1", "genesis-placeholder")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	g := makeStoredBranchBlock(t, 0, "", "g")
	a1 := makeStoredBranchBlock(t, 1, g.BlockHash, "a1")
	a2 := makeStoredBranchBlock(t, 2, a1.BlockHash, "a2")
	b2 := makeStoredBranchBlock(t, 2, a1.BlockHash, "b2")
	a3 := makeStoredBranchBlock(t, 3, a2.BlockHash, "a3")
	b3 := makeStoredBranchBlock(t, 3, b2.BlockHash, "b3")

	for _, b := range []*block.Block{g, a1, a2, b2, a3, b3} {
		putRandomXBranchBlock(t, s, b)
	}

	srcA, err := s.NewRandomXAncestrySource(a3.BlockHash)
	if err != nil {
		t.Fatal(err)
	}
	srcB, err := s.NewRandomXAncestrySource(b3.BlockHash)
	if err != nil {
		t.Fatal(err)
	}

	gotA, err := srcA.BlockIDAtHeight(2)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := srcB.BlockIDAtHeight(2)
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(gotA[:]) != a2.BlockHash {
		t.Fatalf("branch A seed=%x want=%s", gotA, a2.BlockHash)
	}
	if hex.EncodeToString(gotB[:]) != b2.BlockHash {
		t.Fatalf("branch B seed=%x want=%s", gotB, b2.BlockHash)
	}
	if gotA == gotB {
		t.Fatal("competing branch ancestry collapsed to active-height lookup")
	}

	commonA, err := srcA.BlockIDAtHeight(1)
	if err != nil {
		t.Fatal(err)
	}
	commonB, err := srcB.BlockIDAtHeight(1)
	if err != nil {
		t.Fatal(err)
	}
	if commonA != commonB || hex.EncodeToString(commonA[:]) != a1.BlockHash {
		t.Fatal("common ancestor lookup mismatch")
	}
}

func TestRandomXAncestrySourceRejectsHeightAboveTip(t *testing.T) {
	path := t.TempDir()
	s, err := OpenBadgerStore(path, "valdr-mainnet-1", "genesis-placeholder")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	g := makeStoredBranchBlock(t, 0, "", "g")
	putRandomXBranchBlock(t, s, g)

	src, err := s.NewRandomXAncestrySource(g.BlockHash)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.BlockIDAtHeight(1); err == nil {
		t.Fatal("expected lookup above tip to fail")
	}
}
