package storage

import (
	"fmt"

	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus"
	badger "github.com/dgraph-io/badger/v4"
)

// ValidateStoredMainnetRandomXCandidate binds the Mainnet RandomX candidate
// validator to the actual parent branch persisted in Badger.
//
// This is an explicit candidate-only path and is not called by Testnet2.
func (s *BadgerStore) ValidateStoredMainnetRandomXCandidate(
	candidate *block.Block,
	hasher consensus.RandomXHasher,
) ([]byte, error) {
	if s == nil || s.db == nil {
		return nil, ErrBadgerStoreClosed
	}
	if candidate == nil || candidate.PreviousBlockHash == "" {
		return nil, consensus.ErrMainnetPoWUnsupportedBlock
	}

	parentHash := candidate.PreviousBlockHash
	var parentHeight uint64
	err := s.db.View(func(txn *badger.Txn) error {
		raw, err := getBytes(txn, headerKey(parentHash))
		if err != nil {
			return fmt.Errorf("%w: Mainnet RandomX parent %s: %v", ErrStorageStateMismatch, parentHash, err)
		}
		var hdr headerRecord
		if err := strictJSON(raw, &hdr); err != nil {
			return fmt.Errorf("%w: Mainnet RandomX parent header: %v", ErrStorageMetadataCorrupt, err)
		}
		parentHeight = hdr.Height
		return nil
	})
	if err != nil {
		return nil, err
	}

	source, err := s.NewRandomXAncestrySource(parentHash)
	if err != nil {
		return nil, err
	}
	return consensus.ValidateMainnetRandomXCandidate(
		candidate,
		parentHeight,
		parentHash,
		s.network,
		source,
		hasher,
	)
}
