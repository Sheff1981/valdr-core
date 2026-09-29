package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus"
	badger "github.com/dgraph-io/badger/v4"
)

// RandomXAncestrySource resolves RandomX seed block IDs from one exact stored
// branch. The anchor is normally the parent of the candidate block being
// validated, so a side branch/reorg is resolved from its own ancestry rather
// than from the active height/* mapping.
type RandomXAncestrySource struct {
	store    *BadgerStore
	tipHash  string
	tipHeight uint64
}

var _ consensus.MainnetSeedSource = (*RandomXAncestrySource)(nil)

// NewRandomXAncestrySource anchors seed lookup to a specific already-stored
// block. It fails closed when the anchor is missing or internally inconsistent.
func (s *BadgerStore) NewRandomXAncestrySource(tipHash string) (*RandomXAncestrySource, error) {
	if s == nil || s.db == nil {
		return nil, ErrBadgerStoreClosed
	}
	if tipHash == "" {
		return nil, ErrStorageStateMismatch
	}

	var height uint64
	err := s.db.View(func(txn *badger.Txn) error {
		raw, err := getBytes(txn, headerKey(tipHash))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return fmt.Errorf("%w: RandomX ancestry tip %s", ErrStorageStateMismatch, tipHash)
			}
			return err
		}
		var hdr headerRecord
		if err := strictJSON(raw, &hdr); err != nil {
			return fmt.Errorf("%w: RandomX ancestry tip header: %v", ErrStorageMetadataCorrupt, err)
		}

		rawBlock, err := getBytes(txn, blockKey(tipHash))
		if err != nil {
			return fmt.Errorf("%w: RandomX ancestry tip block %s: %v", ErrStorageMetadataCorrupt, tipHash, err)
		}
		var b block.Block
		if err := strictJSON(rawBlock, &b); err != nil {
			return fmt.Errorf("%w: RandomX ancestry tip block: %v", ErrStorageMetadataCorrupt, err)
		}
		if b.BlockHash != tipHash || b.Height != hdr.Height {
			return fmt.Errorf("%w: RandomX ancestry anchor mismatch", ErrStorageMetadataCorrupt)
		}
		height = hdr.Height
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &RandomXAncestrySource{store: s, tipHash: tipHash, tipHeight: height}, nil
}

// BlockIDAtHeight walks parent links on the anchored branch until the requested
// height is reached. It never consults the active height/* mapping.
func (s *RandomXAncestrySource) BlockIDAtHeight(height uint64) ([sha256.Size]byte, error) {
	var zero [sha256.Size]byte
	if s == nil || s.store == nil || s.store.db == nil {
		return zero, ErrBadgerStoreClosed
	}
	if height > s.tipHeight {
		return zero, fmt.Errorf("%w: requested seed height %d above ancestry tip %d", ErrStorageStateMismatch, height, s.tipHeight)
	}

	currentHash := s.tipHash
	currentHeight := s.tipHeight

	err := s.store.db.View(func(txn *badger.Txn) error {
		for currentHeight > height {
			raw, err := getBytes(txn, headerKey(currentHash))
			if err != nil {
				return fmt.Errorf("%w: missing ancestry header %s: %v", ErrStorageMetadataCorrupt, currentHash, err)
			}
			var hdr headerRecord
			if err := strictJSON(raw, &hdr); err != nil {
				return fmt.Errorf("%w: ancestry header %s: %v", ErrStorageMetadataCorrupt, currentHash, err)
			}
			if hdr.Height != currentHeight || hdr.Parent == "" {
				return fmt.Errorf("%w: broken ancestry at height %d", ErrStorageMetadataCorrupt, currentHeight)
			}
			currentHash = hdr.Parent
			currentHeight--
		}

		raw, err := getBytes(txn, headerKey(currentHash))
		if err != nil {
			return fmt.Errorf("%w: seed header %s: %v", ErrStorageMetadataCorrupt, currentHash, err)
		}
		var hdr headerRecord
		if err := strictJSON(raw, &hdr); err != nil {
			return fmt.Errorf("%w: seed header %s: %v", ErrStorageMetadataCorrupt, currentHash, err)
		}
		if hdr.Height != height {
			return fmt.Errorf("%w: seed height mismatch got=%d want=%d", ErrStorageMetadataCorrupt, hdr.Height, height)
		}
		return nil
	})
	if err != nil {
		return zero, err
	}

	rawHash, err := hex.DecodeString(currentHash)
	if err != nil || len(rawHash) != sha256.Size {
		return zero, fmt.Errorf("%w: invalid block id %q", ErrStorageMetadataCorrupt, currentHash)
	}
	var out [sha256.Size]byte
	copy(out[:], rawHash)
	return out, nil
}
