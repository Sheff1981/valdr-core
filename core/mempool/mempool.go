package mempool

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"

	"github.com/Sheff1981/valdr-core/core/transaction"
)

var (
	ErrInvalidTransaction   = errors.New("invalid mempool transaction")
	ErrDuplicateTransaction = errors.New("transaction already in mempool")
	ErrInputConflict        = errors.New("mempool input already spent")
)

type Pool struct {
	mu    sync.RWMutex
	txs   map[string]*transaction.Transaction
	spent map[string]string
}

func New() *Pool {
	return &Pool{
		txs:   make(map[string]*transaction.Transaction),
		spent: make(map[string]string),
	}
}

func (p *Pool) Add(tx *transaction.Transaction) error {
	if tx == nil {
		return ErrInvalidTransaction
	}
	if err := tx.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTransaction, err)
	}

	copyTx := cloneTransaction(tx)

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.txs[copyTx.TransactionID]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateTransaction, copyTx.TransactionID)
	}

	for _, input := range copyTx.Inputs {
		key := inputKey(input)
		if existing, exists := p.spent[key]; exists {
			return fmt.Errorf(
				"%w: %s already used by %s",
				ErrInputConflict,
				key,
				existing,
			)
		}
	}

	p.txs[copyTx.TransactionID] = copyTx
	for _, input := range copyTx.Inputs {
		p.spent[inputKey(input)] = copyTx.TransactionID
	}
	return nil
}

func (p *Pool) Remove(transactionID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	tx, exists := p.txs[transactionID]
	if !exists {
		return false
	}
	delete(p.txs, transactionID)
	for _, input := range tx.Inputs {
		delete(p.spent, inputKey(input))
	}
	return true
}

func (p *Pool) RemoveBlockTransactions(txs []*transaction.Transaction) {
	for _, tx := range txs {
		if tx == nil || tx.IsCoinbase() {
			continue
		}
		p.Remove(tx.TransactionID)
	}
}

func (p *Pool) Contains(transactionID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, exists := p.txs[transactionID]
	return exists
}

func (p *Pool) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.txs)
}

func (p *Pool) Transactions() []*transaction.Transaction {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ids := make([]string, 0, len(p.txs))
	for id := range p.txs {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	txs := make([]*transaction.Transaction, 0, len(ids))
	for _, id := range ids {
		txs = append(txs, cloneTransaction(p.txs[id]))
	}
	return txs
}

func inputKey(input transaction.Input) string {
	return input.PreviousTransactionID + ":" + strconv.FormatUint(uint64(input.OutputIndex), 10)
}

func cloneTransaction(tx *transaction.Transaction) *transaction.Transaction {
	if tx == nil {
		return nil
	}
	copyTx := *tx
	copyTx.Inputs = append([]transaction.Input(nil), tx.Inputs...)
	copyTx.Outputs = append([]transaction.Output(nil), tx.Outputs...)
	return &copyTx
}
