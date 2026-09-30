package storage

import (
	"fmt"
	"math/big"

	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus"
	badger "github.com/dgraph-io/badger/v4"
)

// LoadMainnetDAAHistory returns up to maxHeaders headers from one exact stored
// branch, oldest to newest, ending at tipHash. It never uses active height/*
// indexes, so competing-fork validation remains branch-local.
func (s *BadgerStore) LoadMainnetDAAHistory(
	tipHash string,
	maxHeaders int,
) ([]consensus.MainnetDAAHeader, error) {
	if s == nil || s.db == nil {
		return nil, ErrBadgerStoreClosed
	}
	if tipHash == "" || maxHeaders <= 0 {
		return nil, ErrStorageStateMismatch
	}

	reversed := make([]consensus.MainnetDAAHeader, 0, maxHeaders)
	current := tipHash

	err := s.db.View(func(txn *badger.Txn) error {
		for len(reversed) < maxHeaders {
			raw, err := getBytes(txn, headerKey(current))
			if err != nil {
				return fmt.Errorf("%w: Mainnet DAA header %s: %v", ErrStorageMetadataCorrupt, current, err)
			}
			var hdr headerRecord
			if err := strictJSON(raw, &hdr); err != nil {
				return fmt.Errorf("%w: Mainnet DAA header %s: %v", ErrStorageMetadataCorrupt, current, err)
			}
			target, err := consensus.ParseTargetHexV2(hdr.Target)
			if err != nil {
				return fmt.Errorf("%w: Mainnet DAA target at height %d", ErrStorageMetadataCorrupt, hdr.Height)
			}

			rawBlock, err := getBytes(txn, blockKey(current))
			if err != nil {
				return fmt.Errorf("%w: Mainnet DAA block %s: %v", ErrStorageMetadataCorrupt, current, err)
			}
			var b block.Block
			if err := strictJSON(rawBlock, &b); err != nil {
				return fmt.Errorf("%w: Mainnet DAA block %s: %v", ErrStorageMetadataCorrupt, current, err)
			}
			if b.BlockHash != current || b.Height != hdr.Height || b.Target != hdr.Target {
				return fmt.Errorf("%w: Mainnet DAA header/block mismatch %s", ErrStorageMetadataCorrupt, current)
			}

			reversed = append(reversed, consensus.MainnetDAAHeader{
				Height:    hdr.Height,
				Timestamp: b.Timestamp,
				Target:    target,
			})

			if hdr.Height == 0 {
				break
			}
			if hdr.Parent == "" {
				return fmt.Errorf("%w: broken Mainnet DAA ancestry at height %d", ErrStorageMetadataCorrupt, hdr.Height)
			}
			current = hdr.Parent
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	history := make([]consensus.MainnetDAAHeader, len(reversed))
	for i := range reversed {
		history[len(reversed)-1-i] = reversed[i]
	}
	return history, nil
}

// ValidateStoredMainnetConsensusCandidate is the persisted candidate path for
// M3+M4. M5 supplies exact InitialTarget/PowLimit later.
func (s *BadgerStore) ValidateStoredMainnetConsensusCandidate(
	candidate *block.Block,
	hasher consensus.RandomXHasher,
	initialTarget *big.Int,
	powLimit *big.Int,
	localSystemTime int64,
) ([]byte, error) {
	if candidate == nil || candidate.PreviousBlockHash == "" {
		return nil, consensus.ErrMainnetPoWUnsupportedBlock
	}

	history, err := s.LoadMainnetDAAHistory(
		candidate.PreviousBlockHash,
		consensus.MainnetDAAWindow+1,
	)
	if err != nil {
		return nil, err
	}
	source, err := s.NewRandomXAncestrySource(candidate.PreviousBlockHash)
	if err != nil {
		return nil, err
	}

	return consensus.ValidateMainnetConsensusCandidate(
		candidate,
		history,
		candidate.PreviousBlockHash,
		s.network,
		initialTarget,
		powLimit,
		localSystemTime,
		source,
		hasher,
	)
}
