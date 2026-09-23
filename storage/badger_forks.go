package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus"
	"github.com/Sheff1981/valdr-core/core/utxo"
	badger "github.com/dgraph-io/badger/v4"
)

// LoadAllBlocks returns every validated block retained by the v2 store,
// including side branches. Active-chain order remains defined by height/*.
func (s *BadgerStore) LoadAllBlocks() ([]*block.Block, error) {
	if s == nil || s.db == nil {
		return nil, ErrBadgerStoreClosed
	}
	var blocks []*block.Block
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("block/")
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			raw, err := it.Item().ValueCopy(nil)
			if err != nil {
				return err
			}
			var candidate block.Block
			if err := strictJSON(raw, &candidate); err != nil {
				return fmt.Errorf("%w: branch block: %v", ErrStorageMetadataCorrupt, err)
			}
			blocks = append(blocks, &candidate)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].Height == blocks[j].Height {
			return blocks[i].BlockHash < blocks[j].BlockHash
		}
		return blocks[i].Height < blocks[j].Height
	})
	return blocks, nil
}

// CommitCandidate persists one validated branch block. If activeChain is nil,
// the block remains a side branch. Otherwise all active-chain changes are
// committed atomically in the same Badger transaction.
func (s *BadgerStore) CommitCandidate(
	candidate *block.Block,
	before []utxo.UTXO,
	after []utxo.UTXO,
	chainwork string,
	activeChain []*block.Block,
	activeUTXO []utxo.UTXO,
) error {
	if s == nil || s.db == nil {
		return ErrBadgerStoreClosed
	}
	if candidate == nil || candidate.Height == 0 || chainwork == "" {
		return ErrInvalidDiskChain
	}

	rawBlock, err := json.Marshal(candidate)
	if err != nil {
		return err
	}
	target, _, err := blockTargetAndWork(candidate)
	if err != nil {
		return err
	}
	status := "side"
	if activeChain != nil {
		status = "active"
	}
	rawHeader, err := json.Marshal(headerRecord{
		Parent: candidate.PreviousBlockHash,
		Height: candidate.Height,
		Difficulty: candidate.Difficulty,
		Status: status,
		Target: target,
		Chainwork: chainwork,
	})
	if err != nil {
		return err
	}
	spent, created := diffUTXO(before, after)
	rawUndo, err := json.Marshal(undoRecord{Spent: spent, Created: created})
	if err != nil {
		return err
	}

	return s.db.Update(func(txn *badger.Txn) error {
		if err := s.validateIdentityTxn(txn); err != nil {
			return err
		}
		if _, err := txn.Get(headerKey(candidate.PreviousBlockHash)); err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return fmt.Errorf("%w: parent header %s", ErrStorageStateMismatch, candidate.PreviousBlockHash)
			}
			return err
		}
		if _, err := txn.Get(blockKey(candidate.BlockHash)); err == nil {
			return fmt.Errorf("%w: duplicate block %s", ErrStorageStateMismatch, candidate.BlockHash)
		} else if !errors.Is(err, badger.ErrKeyNotFound) {
			return err
		}

		if err := txn.Set(blockKey(candidate.BlockHash), rawBlock); err != nil {
			return err
		}
		if err := txn.Set(headerKey(candidate.BlockHash), rawHeader); err != nil {
			return err
		}
		if err := txn.Set(undoKey(candidate.BlockHash), rawUndo); err != nil {
			return err
		}
		if activeChain == nil {
			return nil
		}
		if len(activeChain) == 0 ||
			activeChain[len(activeChain)-1] == nil ||
			activeChain[len(activeChain)-1].BlockHash != candidate.BlockHash {
			return ErrStorageStateMismatch
		}

		currentTip, err := getString(txn, keyActiveTip)
		if err != nil {
			return err
		}
		currentHeight, err := getUint64(txn, keyActiveHeight)
		if err != nil {
			return err
		}

		if currentTip == candidate.PreviousBlockHash &&
			currentHeight+1 == candidate.Height {
			return s.commitActiveExtensionTxn(
				txn,
				candidate,
				before,
				after,
				chainwork,
				activeChain,
			)
		}
		return s.commitReorgTxn(
			txn,
			activeChain,
			activeUTXO,
			chainwork,
		)
	})
}

func (s *BadgerStore) commitActiveExtensionTxn(
	txn *badger.Txn,
	candidate *block.Block,
	before []utxo.UTXO,
	after []utxo.UTXO,
	chainwork string,
	activeChain []*block.Block,
) error {
	currentUTXOHash, err := getString(txn, keyUTXOHash)
	if err != nil {
		return err
	}
	if currentUTXOHash != HashUTXOSet(before) {
		return fmt.Errorf("%w: active extension UTXO mismatch", ErrStorageStateMismatch)
	}

	spent, created := diffUTXO(before, after)
	for index, tx := range candidate.Transactions {
		if tx == nil {
			continue
		}
		rawTx, err := json.Marshal(transactionRecord{
			BlockHeight: candidate.Height,
			BlockHash: candidate.BlockHash,
			Index: uint32(index),
			Transaction: tx,
		})
		if err != nil {
			return err
		}
		if err := txn.Set(txKey(tx.TransactionID), rawTx); err != nil {
			return err
		}
	}
	for _, item := range spent {
		if err := txn.Delete(utxoKey(item.TransactionID, item.OutputIndex)); err != nil {
			return err
		}
	}
	afterMap := utxoMap(after)
	for _, outpoint := range created {
		item, ok := afterMap[outpointKeyString(outpoint.TransactionID, outpoint.OutputIndex)]
		if !ok {
			return ErrStorageStateMismatch
		}
		raw, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if err := txn.Set(utxoKey(item.TransactionID, item.OutputIndex), raw); err != nil {
			return err
		}
	}
	if err := txn.Set(heightKey(candidate.Height), []byte(candidate.BlockHash)); err != nil {
		return err
	}
	if err := writeActiveHeaderMetadataTxn(txn, activeChain); err != nil {
		return err
	}
	if err := txn.Set(keyActiveTip, []byte(candidate.BlockHash)); err != nil {
		return err
	}
	if err := txn.Set(keyActiveHeight, encodeUint64(candidate.Height)); err != nil {
		return err
	}
	if err := txn.Set(keyActiveChainwork, []byte(chainwork)); err != nil {
		return err
	}
	return txn.Set(keyUTXOHash, []byte(HashUTXOSet(after)))
}

func (s *BadgerStore) commitReorgTxn(
	txn *badger.Txn,
	activeChain []*block.Block,
	activeUTXO []utxo.UTXO,
	chainwork string,
) error {
	if len(activeChain) == 0 || activeChain[0] == nil ||
		activeChain[0].BlockHash != s.genesisHash {
		return ErrStorageStateMismatch
	}

	if err := backfillCurrentActiveHeadersTxn(txn); err != nil {
		return err
	}
	if err := markAllHeadersSideTxn(txn); err != nil {
		return err
	}
	for _, prefix := range []string{"height/", "tx/", "utxo/"} {
		if err := deletePrefixTxn(txn, []byte(prefix)); err != nil {
			return err
		}
	}

	finalWork, err := writeActiveChainTxn(txn, activeChain)
	if err != nil {
		return err
	}
	if consensus.ChainworkHex(finalWork) != chainwork {
		return fmt.Errorf("%w: chainwork mismatch", ErrStorageStateMismatch)
	}

	for _, item := range activeUTXO {
		raw, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if err := txn.Set(utxoKey(item.TransactionID, item.OutputIndex), raw); err != nil {
			return err
		}
	}

	tip := activeChain[len(activeChain)-1]
	if err := txn.Set(keyActiveTip, []byte(tip.BlockHash)); err != nil {
		return err
	}
	if err := txn.Set(keyActiveHeight, encodeUint64(tip.Height)); err != nil {
		return err
	}
	if err := txn.Set(keyActiveChainwork, []byte(chainwork)); err != nil {
		return err
	}
	return txn.Set(keyUTXOHash, []byte(HashUTXOSet(activeUTXO)))
}

func writeActiveChainTxn(txn *badger.Txn, activeChain []*block.Block) (*big.Int, error) {
	var cumulative *big.Int
	for _, candidate := range activeChain {
		if candidate == nil {
			return nil, ErrStorageStateMismatch
		}
		target, nextWork, err := blockTargetAndCumulativeWork(candidate, cumulative)
		if err != nil {
			return nil, err
		}
		cumulative = nextWork
		rawHeader, err := json.Marshal(headerRecord{
			Parent: candidate.PreviousBlockHash,
			Height: candidate.Height,
			Difficulty: candidate.Difficulty,
			Status: "active",
			Target: target,
			Chainwork: consensus.ChainworkHex(cumulative),
		})
		if err != nil {
			return nil, err
		}
		if err := txn.Set(headerKey(candidate.BlockHash), rawHeader); err != nil {
			return nil, err
		}
		if err := txn.Set(heightKey(candidate.Height), []byte(candidate.BlockHash)); err != nil {
			return nil, err
		}
		for index, tx := range candidate.Transactions {
			if tx == nil {
				continue
			}
			rawTx, err := json.Marshal(transactionRecord{
				BlockHeight: candidate.Height,
				BlockHash: candidate.BlockHash,
				Index: uint32(index),
				Transaction: tx,
			})
			if err != nil {
				return nil, err
			}
			if err := txn.Set(txKey(tx.TransactionID), rawTx); err != nil {
				return nil, err
			}
		}
	}
	return cumulative, nil
}

func writeActiveHeaderMetadataTxn(txn *badger.Txn, activeChain []*block.Block) error {
	_, err := writeActiveChainHeadersOnlyTxn(txn, activeChain)
	return err
}

func writeActiveChainHeadersOnlyTxn(
	txn *badger.Txn,
	activeChain []*block.Block,
) (*big.Int, error) {
	var cumulative *big.Int
	for _, candidate := range activeChain {
		if candidate == nil {
			return nil, ErrStorageStateMismatch
		}
		target, nextWork, err := blockTargetAndCumulativeWork(candidate, cumulative)
		if err != nil {
			return nil, err
		}
		cumulative = nextWork
		rawHeader, err := json.Marshal(headerRecord{
			Parent: candidate.PreviousBlockHash,
			Height: candidate.Height,
			Difficulty: candidate.Difficulty,
			Status: "active",
			Target: target,
			Chainwork: consensus.ChainworkHex(cumulative),
		})
		if err != nil {
			return nil, err
		}
		if err := txn.Set(headerKey(candidate.BlockHash), rawHeader); err != nil {
			return nil, err
		}
	}
	return cumulative, nil
}

func backfillCurrentActiveHeadersTxn(txn *badger.Txn) error {
	height, err := getUint64(txn, keyActiveHeight)
	if err != nil {
		return err
	}
	active := make([]*block.Block, 0, height+1)
	for h := uint64(0); h <= height; h++ {
		hash, err := getString(txn, heightKey(h))
		if err != nil {
			return err
		}
		raw, err := getBytes(txn, blockKey(hash))
		if err != nil {
			return err
		}
		var candidate block.Block
		if err := strictJSON(raw, &candidate); err != nil {
			return err
		}
		active = append(active, &candidate)
	}
	_, err = writeActiveChainHeadersOnlyTxn(txn, active)
	return err
}

func markAllHeadersSideTxn(txn *badger.Txn) error {
	type item struct {
		key []byte
		rec headerRecord
	}
	var items []item
	opts := badger.DefaultIteratorOptions
	opts.Prefix = []byte("header/")
	it := txn.NewIterator(opts)
	for it.Rewind(); it.Valid(); it.Next() {
		key := it.Item().KeyCopy(nil)
		raw, err := it.Item().ValueCopy(nil)
		if err != nil {
			it.Close()
			return err
		}
		var rec headerRecord
		if err := strictJSON(raw, &rec); err != nil {
			it.Close()
			return err
		}
		rec.Status = "side"
		items = append(items, item{key: key, rec: rec})
	}
	it.Close()
	for _, item := range items {
		raw, err := json.Marshal(item.rec)
		if err != nil {
			return err
		}
		if err := txn.Set(item.key, raw); err != nil {
			return err
		}
	}
	return nil
}

func blockTargetAndWork(candidate *block.Block) (string, *big.Int, error) {
	if candidate == nil {
		return "", nil, ErrStorageStateMismatch
	}
	if candidate.Version == block.VersionV2 {
		target, err := consensus.ParseTargetHexV2(candidate.Target)
		if err != nil {
			return "", nil, err
		}
		return candidate.Target, target, nil
	}
	targetHex, err := consensus.TargetHex(candidate.Difficulty)
	if err != nil {
		return "", nil, err
	}
	target, err := consensus.TargetForDifficulty(candidate.Difficulty)
	if err != nil {
		return "", nil, err
	}
	return targetHex, target, nil
}

func blockTargetAndCumulativeWork(
	candidate *block.Block,
	parentWork *big.Int,
) (string, *big.Int, error) {
	targetHex, target, err := blockTargetAndWork(candidate)
	if err != nil {
		return "", nil, err
	}
	var cumulative *big.Int
	if candidate.Version == block.VersionV2 {
		cumulative, err = consensus.AddTargetWork(parentWork, target)
	} else {
		cumulative, err = consensus.AddWork(parentWork, candidate.Difficulty)
	}
	return targetHex, cumulative, err
}

func deletePrefixTxn(txn *badger.Txn, prefix []byte) error {
	var keys [][]byte
	opts := badger.DefaultIteratorOptions
	opts.Prefix = prefix
	it := txn.NewIterator(opts)
	for it.Rewind(); it.Valid(); it.Next() {
		keys = append(keys, it.Item().KeyCopy(nil))
	}
	it.Close()
	for _, key := range keys {
		if err := txn.Delete(key); err != nil {
			return err
		}
	}
	return nil
}
