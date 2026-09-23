package mempool

import (
	"errors"
	"fmt"
	"math/bits"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/Sheff1981/valdr-core/core/transaction"
)

const (
	DefaultMaxBytes = 64 * 1024 * 1024
	DefaultExpiry   = 72 * time.Hour
)

var (
	ErrInvalidTransaction   = errors.New("invalid mempool transaction")
	ErrDuplicateTransaction = errors.New("transaction already in mempool")
	ErrInputConflict        = errors.New("mempool input already spent")
	ErrUnconfirmedParent    = errors.New("unconfirmed-parent spending is disabled")
	ErrMinRelayFee          = errors.New("transaction fee rate below relay minimum")
	ErrMempoolFull          = errors.New("transaction evicted by mempool size limit")
)

type Config struct {
	MaxBytes           int
	Expiry             time.Duration
	MinRelayFeePerByte uint64
	Now                func() time.Time
}

type entry struct {
	tx      *transaction.Transaction
	fee     uint64
	size    int
	addedAt time.Time
}

type Pool struct {
	mu    sync.RWMutex
	txs   map[string]entry
	spent map[string]string

	maxBytes           int
	expiry             time.Duration
	minRelayFeePerByte uint64
	now                func() time.Time
	totalBytes         int
}

func New() *Pool {
	return NewWithConfig(Config{})
}

func NewWithConfig(cfg Config) *Pool {
	maxBytes := cfg.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	expiry := cfg.Expiry
	if expiry <= 0 {
		expiry = DefaultExpiry
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &Pool{
		txs:                make(map[string]entry),
		spent:              make(map[string]string),
		maxBytes:           maxBytes,
		expiry:             expiry,
		minRelayFeePerByte: cfg.MinRelayFeePerByte,
		now:                now,
	}
}

// Add preserves the v0.1 zero-fee API. Policy-aware callers should use
// AddWithFee so relay minimums and fee-rate eviction can be enforced.
func (p *Pool) Add(tx *transaction.Transaction) error {
	return p.AddWithFee(tx, 0)
}

func (p *Pool) AddWithFee(tx *transaction.Transaction, fee uint64) error {
	if tx == nil {
		return ErrInvalidTransaction
	}
	if err := tx.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTransaction, err)
	}
	size := tx.SerializedSize()
	if size <= 0 || size > transaction.MaxSerializedSize {
		return fmt.Errorf("%w: serialized size %d", ErrInvalidTransaction, size)
	}
	if !meetsRelayFee(fee, size, p.minRelayFeePerByte) {
		return fmt.Errorf(
			"%w: fee=%d size=%d minimum=%d val/byte",
			ErrMinRelayFee,
			fee,
			size,
			p.minRelayFeePerByte,
		)
	}

	copyTx := cloneTransaction(tx)
	now := p.now()

	p.mu.Lock()
	defer p.mu.Unlock()

	p.expireLocked(now)

	if _, exists := p.txs[copyTx.TransactionID]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateTransaction, copyTx.TransactionID)
	}
	for _, input := range copyTx.Inputs {
		if _, exists := p.txs[input.PreviousTransactionID]; exists {
			return fmt.Errorf(
				"%w: parent=%s",
				ErrUnconfirmedParent,
				input.PreviousTransactionID,
			)
		}
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

	p.txs[copyTx.TransactionID] = entry{
		tx:      copyTx,
		fee:     fee,
		size:    size,
		addedAt: now,
	}
	p.totalBytes += size
	for _, input := range copyTx.Inputs {
		p.spent[inputKey(input)] = copyTx.TransactionID
	}

	for p.totalBytes > p.maxBytes && len(p.txs) > 0 {
		victimID := p.lowestPriorityLocked()
		if victimID == "" {
			break
		}
		p.removeLocked(victimID)
	}
	if _, exists := p.txs[copyTx.TransactionID]; !exists {
		return fmt.Errorf("%w: txid=%s", ErrMempoolFull, copyTx.TransactionID)
	}
	return nil
}

func (p *Pool) Remove(transactionID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.removeLocked(transactionID)
}

func (p *Pool) RemoveBlockTransactions(txs []*transaction.Transaction) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, tx := range txs {
		if tx == nil || tx.IsCoinbase() {
			continue
		}
		p.removeLocked(tx.TransactionID)
	}
}

func (p *Pool) Contains(transactionID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.expireLocked(p.now())
	_, exists := p.txs[transactionID]
	return exists
}

func (p *Pool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.expireLocked(p.now())
	return len(p.txs)
}

func (p *Pool) Bytes() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.expireLocked(p.now())
	return p.totalBytes
}

// Transactions returns a stable txid-sorted snapshot for RPC/status surfaces.
func (p *Pool) Transactions() []*transaction.Transaction {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.expireLocked(p.now())

	ids := make([]string, 0, len(p.txs))
	for id := range p.txs {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	txs := make([]*transaction.Transaction, 0, len(ids))
	for _, id := range ids {
		txs = append(txs, cloneTransaction(p.txs[id].tx))
	}
	return txs
}

// MiningTransactions returns a deterministic miner template order:
// highest fee-rate first, then txid ascending.
func (p *Pool) MiningTransactions() []*transaction.Transaction {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.expireLocked(p.now())

	entries := make([]entry, 0, len(p.txs))
	for _, candidate := range p.txs {
		entries = append(entries, candidate)
	}
	sort.Slice(entries, func(i, j int) bool {
		cmp := compareFeeRate(entries[i], entries[j])
		if cmp != 0 {
			return cmp > 0
		}
		return entries[i].tx.TransactionID < entries[j].tx.TransactionID
	})

	result := make([]*transaction.Transaction, 0, len(entries))
	for _, candidate := range entries {
		result = append(result, cloneTransaction(candidate.tx))
	}
	return result
}

// Revalidate checks the entire pool against the current active-chain UTXO
// state. Invalid/under-minimum transactions are dropped and valid fees are
// refreshed. Validator must return the transaction's current implicit fee.
func (p *Pool) Revalidate(
	validator func(*transaction.Transaction) (uint64, error),
) int {
	if validator == nil {
		return 0
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.expireLocked(p.now())

	ids := make([]string, 0, len(p.txs))
	for id := range p.txs {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	removed := 0
	for _, id := range ids {
		candidate, exists := p.txs[id]
		if !exists {
			continue
		}
		fee, err := validator(cloneTransaction(candidate.tx))
		if err != nil || !meetsRelayFee(fee, candidate.size, p.minRelayFeePerByte) {
			if p.removeLocked(id) {
				removed++
			}
			continue
		}
		candidate.fee = fee
		p.txs[id] = candidate
	}
	return removed
}

func (p *Pool) Reconsider(
	tx *transaction.Transaction,
	validator func(*transaction.Transaction) (uint64, error),
) error {
	if tx == nil || tx.IsCoinbase() {
		return ErrInvalidTransaction
	}
	if validator == nil {
		return ErrInvalidTransaction
	}
	fee, err := validator(tx)
	if err != nil {
		return err
	}
	return p.AddWithFee(tx, fee)
}

func (p *Pool) Expire() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	before := len(p.txs)
	p.expireLocked(p.now())
	return before - len(p.txs)
}

func (p *Pool) removeLocked(transactionID string) bool {
	candidate, exists := p.txs[transactionID]
	if !exists {
		return false
	}
	delete(p.txs, transactionID)
	p.totalBytes -= candidate.size
	if p.totalBytes < 0 {
		p.totalBytes = 0
	}
	for _, input := range candidate.tx.Inputs {
		delete(p.spent, inputKey(input))
	}
	return true
}

func (p *Pool) expireLocked(now time.Time) {
	for id, candidate := range p.txs {
		if now.Sub(candidate.addedAt) >= p.expiry {
			p.removeLocked(id)
		}
	}
}

func (p *Pool) lowestPriorityLocked() string {
	var (
		victimID string
		victim   entry
		found    bool
	)
	for id, candidate := range p.txs {
		if !found {
			victimID, victim, found = id, candidate, true
			continue
		}
		cmp := compareFeeRate(candidate, victim)
		if cmp < 0 ||
			(cmp == 0 && candidate.addedAt.Before(victim.addedAt)) ||
			(cmp == 0 && candidate.addedAt.Equal(victim.addedAt) && id < victimID) {
			victimID, victim = id, candidate
		}
	}
	return victimID
}

// compareFeeRate compares a.fee/a.size with b.fee/b.size exactly without
// floating-point arithmetic.
func compareFeeRate(a, b entry) int {
	aHi, aLo := bits.Mul64(a.fee, uint64(b.size))
	bHi, bLo := bits.Mul64(b.fee, uint64(a.size))
	if aHi < bHi {
		return -1
	}
	if aHi > bHi {
		return 1
	}
	if aLo < bLo {
		return -1
	}
	if aLo > bLo {
		return 1
	}
	return 0
}

func meetsRelayFee(fee uint64, size int, minimum uint64) bool {
	if minimum == 0 {
		return true
	}
	hi, lo := bits.Mul64(minimum, uint64(size))
	if hi != 0 {
		return false
	}
	return fee >= lo
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
