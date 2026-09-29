//go:build randomx_native && cgo

package storage

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus/randomxnative"
	badger "github.com/dgraph-io/badger/v4"
)

func persistMainnetRandomXTestBlock(t *testing.T, s *BadgerStore, b *block.Block) {
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

func makeMainnetRandomXTestBlock(
	t *testing.T,
	height uint64,
	parent string,
	target string,
	nonce uint64,
) *block.Block {
	t.Helper()

	b, err := block.NewV2(
		height,
		parent,
		1800000000+int64(height),
		target,
		nonce,
		nil,
		"valdr-mainnet-1",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	b.BlockHash = b.CalculateHash()
	return b
}

// End-to-end candidate path:
// persisted parent ancestry -> seed resolution -> native RandomX -> target check.
func TestStoredMainnetRandomXCandidateNativeEndToEnd(t *testing.T) {
	s, err := OpenBadgerStore(t.TempDir(), "valdr-mainnet-1", "genesis-placeholder")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	target := strings.Repeat("ff", 32)
	parent := ""
	var tip *block.Block
	for height := uint64(0); height < 10; height++ {
		b := makeMainnetRandomXTestBlock(t, height, parent, target, height+1)
		persistMainnetRandomXTestBlock(t, s, b)
		parent = b.BlockHash
		tip = b
	}
	if tip == nil {
		t.Fatal("missing test tip")
	}

	candidate := makeMainnetRandomXTestBlock(t, 10, tip.BlockHash, target, 0x0102030405060708)

	h, err := randomxnative.New()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	powHash, err := s.ValidateStoredMainnetRandomXCandidate(candidate, h)
	if err != nil {
		t.Fatalf("candidate validation failed: %v", err)
	}
	if len(powHash) != 32 {
		t.Fatalf("pow hash length=%d want=32", len(powHash))
	}
	t.Logf("mainnet_candidate_pow_hash=%s", hex.EncodeToString(powHash))
}
