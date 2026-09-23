package blockchain

import (
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/transaction"
)

type TransactionLocation struct {
	Transaction *transaction.Transaction
	BlockHeight uint64
	BlockHash   string
}

func (bc *Blockchain) BlockByHash(hash string) (*block.Block, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	node, ok := bc.nodes[hash]
	if !ok || node == nil {
		return nil, false
	}
	return node.block, true
}

func (bc *Blockchain) TransactionByID(transactionID string) (TransactionLocation, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	for _, candidate := range bc.blocks {
		for _, tx := range candidate.Transactions {
			if tx != nil && tx.TransactionID == transactionID {
				return TransactionLocation{
					Transaction: tx,
					BlockHeight: candidate.Height,
					BlockHash:   candidate.BlockHash,
				}, true
			}
		}
	}
	return TransactionLocation{}, false
}
