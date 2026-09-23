package utxo

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"

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
// transaction outputs. Coinbase is rejected here and only accepted in
// ApplyBlockTransactions with block context.
func (s *Set) ApplyTransaction(tx *transaction.Transaction) error {
	if s == nil {
		return ErrInvalidUTXO
	}
	if err := tx.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTransaction, err)
	}

	publicKey, err := valdrcrypto.DecodePublicKey(tx.PublicKey)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTransaction, err)
	}
	owner, err := valdrcrypto.AddressFromPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTransaction, err)
	}

	spentKeys := make([]string, 0, len(tx.Inputs))
	var inputTotal uint64
	for _, input := range tx.Inputs {
		key := outpointKey(input.PreviousTransactionID, input.OutputIndex)
		item, exists := s.entries[key]
		if !exists {
			return fmt.Errorf("%w: %s", ErrUTXONotFound, key)
		}
		if item.Recipient != owner {
			return fmt.Errorf("%w: %s", ErrNotOwner, key)
		}
		if math.MaxUint64-inputTotal < item.Amount {
			return ErrBalanceOverflow
		}
		inputTotal += item.Amount
		spentKeys = append(spentKeys, key)
	}

	var outputTotal uint64
	created := make([]UTXO, len(tx.Outputs))
	for i, output := range tx.Outputs {
		if math.MaxUint64-outputTotal < output.Amount {
			return ErrBalanceOverflow
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
			return fmt.Errorf("%w: %s", ErrUTXOAlreadyExists, key)
		}
		created[i] = item
	}

	if inputTotal < outputTotal {
		return fmt.Errorf(
			"%w: inputs=%d outputs=%d",
			ErrInsufficientFunds,
			inputTotal,
			outputTotal,
		)
	}
	if inputTotal != outputTotal {
		return fmt.Errorf(
			"%w: inputs=%d outputs=%d",
			ErrValueMismatch,
			inputTotal,
			outputTotal,
		)
	}

	for _, key := range spentKeys {
		delete(s.entries, key)
	}
	for _, item := range created {
		s.entries[outpointKey(item.TransactionID, item.OutputIndex)] = item
	}
	return nil
}

// ApplyTransactions applies a batch of normal transactions atomically.
func (s *Set) ApplyTransactions(transactions []*transaction.Transaction) error {
	if s == nil {
		return ErrInvalidUTXO
	}
	working := s.clone()
	for i, tx := range transactions {
		if err := working.ApplyTransaction(tx); err != nil {
			return fmt.Errorf("transaction %d: %w", i, err)
		}
	}
	s.entries = working.entries
	return nil
}

// ApplyBlockTransactions validates exactly one coinbase at index zero,
// applies all normal transactions atomically, then creates the subsidy UTXO.
// Applying coinbase last prevents spending the new subsidy in the same block.
func (s *Set) ApplyBlockTransactions(
	blockHeight uint64,
	transactions []*transaction.Transaction,
	expectedReward uint64,
) error {
	if s == nil {
		return ErrInvalidUTXO
	}
	if len(transactions) == 0 {
		return ErrMissingCoinbase
	}

	coinbase := transactions[0]
	if err := coinbase.ValidateCoinbase(blockHeight, expectedReward); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidCoinbase, err)
	}

	working := s.clone()
	for i, tx := range transactions[1:] {
		if tx != nil && tx.HasCoinbaseMarker() {
			return fmt.Errorf("%w at transaction %d", ErrUnexpectedCoinbase, i+1)
		}
		if err := working.ApplyTransaction(tx); err != nil {
			return fmt.Errorf("transaction %d: %w", i+1, err)
		}
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
