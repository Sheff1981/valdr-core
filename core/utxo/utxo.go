package utxo

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/transaction"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

var (
	ErrInvalidUTXO        = errors.New("invalid UTXO")
	ErrDuplicateUTXO      = errors.New("duplicate UTXO")
	ErrUTXONotFound       = errors.New("UTXO not found or already spent")
	ErrUTXOAlreadyExists  = errors.New("UTXO already exists")
	ErrNotOwner           = errors.New("transaction signer does not own UTXO")
	ErrInsufficientFunds  = errors.New("insufficient input value")
	ErrValueMismatch      = errors.New("input and output values do not match")
	ErrBalanceOverflow    = errors.New("balance overflow")
	ErrInvalidTransaction = errors.New("invalid transaction")
	ErrMissingCoinbase    = errors.New("block is missing coinbase transaction")
	ErrInvalidCoinbase    = errors.New("invalid coinbase transaction")
	ErrUnexpectedCoinbase = errors.New("unexpected coinbase transaction")
)

type UTXO struct {
	TransactionID string `json:"transaction_id"`
	OutputIndex   uint32 `json:"output_index"`
	Amount        uint64 `json:"amount"`
	Recipient     string `json:"recipient"`
}

type Outpoint struct {
	TransactionID string `json:"transaction_id"`
	OutputIndex   uint32 `json:"output_index"`
}

type Set struct {
	entries map[string]UTXO
}

// New reconstructs a UTXO set from already validated chain state.
func New(initial []UTXO) (*Set, error) {
	s := &Set{entries: make(map[string]UTXO, len(initial))}
	for _, item := range initial {
		if err := validateUTXO(item); err != nil {
			return nil, err
		}
		key := outpointKey(item.TransactionID, item.OutputIndex)
		if _, exists := s.entries[key]; exists {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateUTXO, key)
		}
		s.entries[key] = item
	}
	return s, nil
}

func NewEmpty() *Set {
	return &Set{entries: make(map[string]UTXO)}
}

func (s *Set) Len() int {
	if s == nil {
		return 0
	}
	return len(s.entries)
}

func (s *Set) Get(transactionID string, outputIndex uint32) (UTXO, bool) {
	if s == nil {
		return UTXO{}, false
	}
	item, ok := s.entries[outpointKey(transactionID, outputIndex)]
	return item, ok
}

func (s *Set) Balance(address string) (uint64, error) {
	if s == nil {
		return 0, ErrInvalidUTXO
	}
	if !valdrcrypto.ValidateAddress(address) {
		return 0, valdrcrypto.ErrInvalidAddress
	}

	var total uint64
	for _, item := range s.entries {
		if item.Recipient != address {
			continue
		}
		if math.MaxUint64-total < item.Amount {
			return 0, ErrBalanceOverflow
		}
		total += item.Amount
	}
	return total, nil
}

func (s *Set) List(address string) ([]UTXO, error) {
	if s == nil {
		return nil, ErrInvalidUTXO
	}
	if !valdrcrypto.ValidateAddress(address) {
		return nil, valdrcrypto.ErrInvalidAddress
	}

	items := make([]UTXO, 0)
	for _, item := range s.entries {
		if item.Recipient == address {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].TransactionID == items[j].TransactionID {
			return items[i].OutputIndex < items[j].OutputIndex
		}
		return items[i].TransactionID < items[j].TransactionID
	})
	return items, nil
}

// ApplyTransaction atomically spends referenced UTXOs and creates normal
// transaction outputs. Any positive input/output difference is an implicit fee.
func (s *Set) ApplyTransaction(tx *transaction.Transaction) error {
	_, err := s.ApplyTransactionWithFee(tx)
	return err
}

// ApplyTransactionWithFee applies a normal transaction and returns its implicit
// fee in val. Coinbase is rejected here and only accepted with block context.
func (s *Set) ApplyTransactionWithFee(tx *transaction.Transaction) (uint64, error) {
	return s.ApplyTransactionWithFeeForChain(tx, config.ChainID)
}

func (s *Set) ApplyTransactionWithFeeForChain(
	tx *transaction.Transaction,
	chainID string,
) (uint64, error) {
	if s == nil {
		return 0, ErrInvalidUTXO
	}
	if err := tx.ValidateForChain(chainID); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidTransaction, err)
	}

	publicKey, err := valdrcrypto.DecodePublicKey(tx.PublicKey)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidTransaction, err)
	}
	owner, err := valdrcrypto.AddressFromPublicKey(publicKey)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidTransaction, err)
	}

	spentKeys := make([]string, 0, len(tx.Inputs))
	var inputTotal uint64
	for _, input := range tx.Inputs {
		key := outpointKey(input.PreviousTransactionID, input.OutputIndex)
		item, exists := s.entries[key]
		if !exists {
			return 0, fmt.Errorf("%w: %s", ErrUTXONotFound, key)
		}
		if item.Recipient != owner {
			return 0, fmt.Errorf("%w: %s", ErrNotOwner, key)
		}
		if math.MaxUint64-inputTotal < item.Amount {
			return 0, ErrBalanceOverflow
		}
		inputTotal += item.Amount
		spentKeys = append(spentKeys, key)
	}

	var outputTotal uint64
	created := make([]UTXO, len(tx.Outputs))
	for i, output := range tx.Outputs {
		if math.MaxUint64-outputTotal < output.Amount {
			return 0, ErrBalanceOverflow
		}
		outputTotal += output.Amount

		item := UTXO{
			TransactionID: tx.TransactionID,
			OutputIndex:   uint32(i),
			Amount:        output.Amount,
			Recipient:     output.Recipient,
		}
		key := outpointKey(item.TransactionID, item.OutputIndex)
		if _, exists := s.entries[key]; exists {
			return 0, fmt.Errorf("%w: %s", ErrUTXOAlreadyExists, key)
		}
		created[i] = item
	}

	if inputTotal < outputTotal {
		return 0, fmt.Errorf(
			"%w: inputs=%d outputs=%d",
			ErrInsufficientFunds,
			inputTotal,
			outputTotal,
		)
	}
	fee := inputTotal - outputTotal

	for _, key := range spentKeys {
		delete(s.entries, key)
	}
	for _, item := range created {
		s.entries[outpointKey(item.TransactionID, item.OutputIndex)] = item
	}
	return fee, nil
}

// ApplyTransactions applies a batch of normal transactions atomically.
func (s *Set) ApplyTransactions(transactions []*transaction.Transaction) error {
	_, err := s.ApplyTransactionsWithFees(transactions)
	return err
}

// ApplyTransactionsWithFees applies a batch atomically and returns the total
// implicit fee. Transactions are evaluated sequentially against the working
// UTXO view.
func (s *Set) ApplyTransactionsWithFees(transactions []*transaction.Transaction) (uint64, error) {
	return s.ApplyTransactionsWithFeesForChain(
		transactions,
		config.ChainID,
	)
}

func (s *Set) ApplyTransactionsWithFeesForChain(
	transactions []*transaction.Transaction,
	chainID string,
) (uint64, error) {
	if s == nil {
		return 0, ErrInvalidUTXO
	}
	working := s.clone()
	var totalFees uint64
	for i, tx := range transactions {
		fee, err := working.ApplyTransactionWithFeeForChain(
			tx,
			chainID,
		)
		if err != nil {
			return 0, fmt.Errorf("transaction %d: %w", i, err)
		}
		if math.MaxUint64-totalFees < fee {
			return 0, ErrBalanceOverflow
		}
		totalFees += fee
	}
	s.entries = working.entries
	return totalFees, nil
}

// ApplyBlockTransactions validates exactly one coinbase at index zero,
// applies normal transactions atomically, sums their implicit fees, then
// creates the coinbase UTXO. The coinbase claim may be below, but never above,
// subsidy + total block fees. Applying coinbase last prevents same-block spend.
func (s *Set) ApplyBlockTransactions(
	blockHeight uint64,
	transactions []*transaction.Transaction,
	expectedReward uint64,
) error {
	return s.ApplyBlockTransactionsForChain(
		blockHeight,
		transactions,
		expectedReward,
		config.ChainID,
	)
}

func (s *Set) ApplyBlockTransactionsForChain(
	blockHeight uint64,
	transactions []*transaction.Transaction,
	expectedReward uint64,
	chainID string,
) error {
	if s == nil {
		return ErrInvalidUTXO
	}
	if len(transactions) == 0 {
		return ErrMissingCoinbase
	}

	coinbase := transactions[0]
	working := s.clone()
	var totalFees uint64
	for i, tx := range transactions[1:] {
		if tx != nil && tx.HasCoinbaseMarker() {
			return fmt.Errorf("%w at transaction %d", ErrUnexpectedCoinbase, i+1)
		}
		fee, err := working.ApplyTransactionWithFeeForChain(
			tx,
			chainID,
		)
		if err != nil {
			return fmt.Errorf("transaction %d: %w", i+1, err)
		}
		if math.MaxUint64-totalFees < fee {
			return ErrBalanceOverflow
		}
		totalFees += fee
	}

	if math.MaxUint64-expectedReward < totalFees {
		return ErrBalanceOverflow
	}
	maxClaim := expectedReward + totalFees
	if err := coinbase.ValidateCoinbaseForChain(
		blockHeight,
		maxClaim,
		chainID,
	); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidCoinbase, err)
	}

	output := coinbase.Outputs[0]
	item := UTXO{
		TransactionID: coinbase.TransactionID,
		OutputIndex:   0,
		Amount:        output.Amount,
		Recipient:     output.Recipient,
	}
	key := outpointKey(item.TransactionID, item.OutputIndex)
	if _, exists := working.entries[key]; exists {
		return fmt.Errorf("%w: %s", ErrUTXOAlreadyExists, key)
	}
	working.entries[key] = item

	s.entries = working.entries
	return nil
}

// ApplyUndo disconnects one previously validated block atomically.
// created are outputs produced by that block; spent are the UTXOs it consumed.
func (s *Set) ApplyUndo(spent []UTXO, created []Outpoint) error {
	if s == nil {
		return ErrInvalidUTXO
	}
	working := s.clone()

	for _, outpoint := range created {
		key := outpointKey(outpoint.TransactionID, outpoint.OutputIndex)
		if _, exists := working.entries[key]; !exists {
			return fmt.Errorf("%w while undoing created output: %s", ErrUTXONotFound, key)
		}
		delete(working.entries, key)
	}

	for _, item := range spent {
		if err := validateUTXO(item); err != nil {
			return err
		}
		key := outpointKey(item.TransactionID, item.OutputIndex)
		if _, exists := working.entries[key]; exists {
			return fmt.Errorf("%w while undoing spent output: %s", ErrUTXOAlreadyExists, key)
		}
		working.entries[key] = item
	}

	s.entries = working.entries
	return nil
}

func (s *Set) Snapshot() []UTXO {
	if s == nil {
		return nil
	}
	items := make([]UTXO, 0, len(s.entries))
	for _, item := range s.entries {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].TransactionID == items[j].TransactionID {
			return items[i].OutputIndex < items[j].OutputIndex
		}
		return items[i].TransactionID < items[j].TransactionID
	})
	return items
}

func (s *Set) clone() *Set {
	clone := NewEmpty()
	for key, item := range s.entries {
		clone.entries[key] = item
	}
	return clone
}

func validateUTXO(item UTXO) error {
	raw, err := hex.DecodeString(item.TransactionID)
	if err != nil || len(raw) != 32 {
		return fmt.Errorf("%w: transaction id", ErrInvalidUTXO)
	}
	if item.Amount == 0 {
		return fmt.Errorf("%w: zero amount", ErrInvalidUTXO)
	}
	if !valdrcrypto.ValidateAddress(item.Recipient) {
		return fmt.Errorf("%w: recipient", ErrInvalidUTXO)
	}
	return nil
}

func outpointKey(transactionID string, outputIndex uint32) string {
	return transactionID + ":" + strconv.FormatUint(uint64(outputIndex), 10)
}
