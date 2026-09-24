package transaction

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/Sheff1981/valdr-core/config"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

const (
	VersionLegacy                 = uint32(1)
	VersionV2                     = uint32(2)
	Version                       = VersionLegacy
	MaxSerializedSize             = 100_000
	CoinbasePreviousTransactionID = "0000000000000000000000000000000000000000000000000000000000000000"
)

var (
	ErrInvalidVersion          = errors.New("invalid transaction version")
	ErrInvalidChainID          = errors.New("invalid transaction chain id")
	ErrNoInputs                = errors.New("transaction has no inputs")
	ErrNoOutputs               = errors.New("transaction has no outputs")
	ErrInvalidInput            = errors.New("invalid transaction input")
	ErrDuplicateInput          = errors.New("duplicate transaction input")
	ErrInvalidOutput           = errors.New("invalid transaction output")
	ErrAmountOverflow          = errors.New("transaction output amount overflow")
	ErrInvalidTimestamp        = errors.New("invalid transaction timestamp")
	ErrInvalidPublicKey        = errors.New("invalid transaction public key")
	ErrInvalidSignature        = errors.New("invalid transaction signature")
	ErrInvalidTransactionID    = errors.New("invalid transaction id")
	ErrTransactionTooLarge     = errors.New("transaction exceeds consensus serialized-size limit")
	ErrCoinbaseRequiresBlock   = errors.New("coinbase transaction requires block context")
	ErrInvalidCoinbase         = errors.New("invalid coinbase transaction")
	ErrInvalidCoinbaseHeight   = errors.New("invalid coinbase block height")
	ErrInvalidCoinbaseReward   = errors.New("invalid coinbase reward")
	ErrCoinbaseHeightOverflow  = errors.New("coinbase block height exceeds input encoding")
)

type Input struct {
	PreviousTransactionID string `json:"transaction_id"`
	OutputIndex           uint32 `json:"output_index"`
}

type Output struct {
	Amount    uint64 `json:"amount"`
	Recipient string `json:"recipient"`
}

type Transaction struct {
	Version       uint32   `json:"version"`
	ChainID       string   `json:"chain_id,omitempty"`
	Inputs        []Input  `json:"inputs"`
	Outputs       []Output `json:"outputs"`
	Timestamp     int64    `json:"timestamp"`
	PublicKey     string   `json:"public_key"`
	Signature     string   `json:"signature"`
	TransactionID string   `json:"transaction_id"`
}

func New(inputs []Input, outputs []Output, timestamp int64) *Transaction {
	return &Transaction{
		Version:   VersionLegacy,
		Inputs:    append([]Input(nil), inputs...),
		Outputs:   append([]Output(nil), outputs...),
		Timestamp: timestamp,
	}
}

func NewForChain(
	chainID string,
	inputs []Input,
	outputs []Output,
	timestamp int64,
) *Transaction {
	if chainID == config.ChainID {
		return New(inputs, outputs, timestamp)
	}
	return &Transaction{
		Version:   VersionV2,
		ChainID:   chainID,
		Inputs:    append([]Input(nil), inputs...),
		Outputs:   append([]Output(nil), outputs...),
		Timestamp: timestamp,
	}
}

// NewCoinbase preserves the frozen legacy v0.1 coinbase format.
func NewCoinbase(
	blockHeight uint64,
	recipient string,
	reward uint64,
	timestamp int64,
) (*Transaction, error) {
	return newCoinbaseForChain(
		config.ChainID,
		blockHeight,
		recipient,
		reward,
		timestamp,
	)
}

func NewCoinbaseForChain(
	chainID string,
	blockHeight uint64,
	recipient string,
	reward uint64,
	timestamp int64,
) (*Transaction, error) {
	return newCoinbaseForChain(
		chainID,
		blockHeight,
		recipient,
		reward,
		timestamp,
	)
}

func newCoinbaseForChain(
	chainID string,
	blockHeight uint64,
	recipient string,
	reward uint64,
	timestamp int64,
) (*Transaction, error) {
	if blockHeight == 0 {
		return nil, ErrInvalidCoinbaseHeight
	}
	if blockHeight > uint64(^uint32(0)) {
		return nil, ErrCoinbaseHeightOverflow
	}

	version := VersionV2
	explicitChainID := chainID
	if chainID == config.ChainID {
		version = VersionLegacy
		explicitChainID = ""
	}
	tx := &Transaction{
		Version: version,
		ChainID: explicitChainID,
		Inputs: []Input{{
			PreviousTransactionID: CoinbasePreviousTransactionID,
			OutputIndex:           uint32(blockHeight),
		}},
		Outputs: []Output{{
			Amount:    reward,
			Recipient: recipient,
		}},
		Timestamp: timestamp,
	}
	tx.TransactionID = tx.CalculateID()

	if err := tx.ValidateCoinbaseForChain(
		blockHeight,
		reward,
		chainID,
	); err != nil {
		return nil, err
	}
	return tx, nil
}

func (tx *Transaction) Sign(key *ecdsa.PrivateKey) error {
	if tx == nil {
		return ErrInvalidTransactionID
	}
	if key == nil {
		return valdrcrypto.ErrInvalidPrivateKey
	}
	if tx.HasCoinbaseMarker() {
		return ErrCoinbaseRequiresBlock
	}

	publicKey, err := valdrcrypto.EncodePublicKey(&key.PublicKey)
	if err != nil {
		return err
	}

	tx.PublicKey = publicKey
	tx.Signature = ""
	tx.TransactionID = ""

	if err := tx.validateUnsignedSelf(); err != nil {
		return err
	}

	signature, err := valdrcrypto.Sign(key, tx.SigningBytes())
	if err != nil {
		return err
	}

	tx.Signature = hex.EncodeToString(signature)
	if tx.SerializedSize() > MaxSerializedSize {
		tx.Signature = ""
		return fmt.Errorf(
			"%w: got %d max %d",
			ErrTransactionTooLarge,
			tx.SerializedSize(),
			MaxSerializedSize,
		)
	}
	tx.TransactionID = tx.CalculateID()
	return nil
}

func (tx *Transaction) VerifySignature() bool {
	if tx == nil || tx.HasCoinbaseMarker() {
		return false
	}
	if err := tx.validateNetworkSelf(); err != nil {
		return false
	}

	publicKey, err := valdrcrypto.DecodePublicKey(tx.PublicKey)
	if err != nil {
		return false
	}

	signature, err := hex.DecodeString(tx.Signature)
	if err != nil || len(signature) == 0 {
		return false
	}

	return valdrcrypto.Verify(
		publicKey,
		tx.SigningBytes(),
		signature,
	)
}

// Validate preserves frozen legacy v0.1 behavior.
func (tx *Transaction) Validate() error {
	return tx.ValidateForChain(config.ChainID)
}

func (tx *Transaction) ValidateForChain(expectedChainID string) error {
	if tx == nil {
		return ErrInvalidTransactionID
	}
	if tx.HasCoinbaseMarker() {
		return ErrCoinbaseRequiresBlock
	}
	if err := tx.validateUnsignedSelf(); err != nil {
		return err
	}
	if err := tx.validateExpectedChain(expectedChainID); err != nil {
		return err
	}
	if tx.SerializedSize() > MaxSerializedSize {
		return fmt.Errorf(
			"%w: got %d max %d",
			ErrTransactionTooLarge,
			tx.SerializedSize(),
			MaxSerializedSize,
		)
	}
	if tx.Signature == "" || !tx.VerifySignature() {
		return ErrInvalidSignature
	}

	expectedID := tx.CalculateID()
	if tx.TransactionID == "" || tx.TransactionID != expectedID {
		return fmt.Errorf(
			"%w: got %q want %q",
			ErrInvalidTransactionID,
			tx.TransactionID,
			expectedID,
		)
	}
	return nil
}

func (tx *Transaction) ValidateCoinbase(
	blockHeight,
	expectedReward uint64,
) error {
	return tx.ValidateCoinbaseForChain(
		blockHeight,
		expectedReward,
		config.ChainID,
	)
}

func (tx *Transaction) ValidateCoinbaseForChain(
	blockHeight,
	expectedReward uint64,
	expectedChainID string,
) error {
	if tx == nil {
		return ErrInvalidCoinbase
	}
	if blockHeight == 0 || blockHeight > uint64(^uint32(0)) {
		return ErrInvalidCoinbaseHeight
	}
	if err := tx.validateNetworkSelf(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCoinbase, err)
	}
	if err := tx.validateExpectedChain(expectedChainID); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCoinbase, err)
	}
	if tx.Timestamp <= 0 {
		return fmt.Errorf(
			"%w: %v",
			ErrInvalidCoinbase,
			ErrInvalidTimestamp,
		)
	}
	if len(tx.Inputs) != 1 ||
		tx.Inputs[0].PreviousTransactionID != CoinbasePreviousTransactionID ||
		tx.Inputs[0].OutputIndex != uint32(blockHeight) {
		return fmt.Errorf(
			"%w: coinbase input marker",
			ErrInvalidCoinbase,
		)
	}
	if len(tx.Outputs) != 1 {
		return fmt.Errorf(
			"%w: coinbase must have exactly one output",
			ErrInvalidCoinbase,
		)
	}
	if expectedReward == 0 ||
		tx.Outputs[0].Amount == 0 ||
		tx.Outputs[0].Amount > expectedReward {
		return fmt.Errorf(
			"%w: got %d max %d",
			ErrInvalidCoinbaseReward,
			tx.Outputs[0].Amount,
			expectedReward,
		)
	}
	if tx.SerializedSize() > MaxSerializedSize {
		return fmt.Errorf(
			"%w: got %d max %d",
			ErrTransactionTooLarge,
			tx.SerializedSize(),
			MaxSerializedSize,
		)
	}
	if !valdrcrypto.ValidateAddress(tx.Outputs[0].Recipient) {
		return fmt.Errorf(
			"%w: recipient",
			ErrInvalidCoinbase,
		)
	}
	if tx.PublicKey != "" || tx.Signature != "" {
		return fmt.Errorf(
			"%w: coinbase must not contain public key or signature",
			ErrInvalidCoinbase,
		)
	}

	expectedID := tx.CalculateID()
	if tx.TransactionID == "" || tx.TransactionID != expectedID {
		return fmt.Errorf(
			"%w: got %q want %q",
			ErrInvalidTransactionID,
			tx.TransactionID,
			expectedID,
		)
	}
	return nil
}

func (tx *Transaction) HasCoinbaseMarker() bool {
	return tx != nil &&
		len(tx.Inputs) == 1 &&
		tx.Inputs[0].PreviousTransactionID ==
			CoinbasePreviousTransactionID
}

func (tx *Transaction) IsCoinbase() bool {
	return tx != nil &&
		tx.HasCoinbaseMarker() &&
		tx.PublicKey == "" &&
		tx.Signature == ""
}

func (tx *Transaction) SigningBytes() []byte {
	var buf bytes.Buffer
	writeString(&buf, tx.canonicalChainID())
	writeUint32(&buf, tx.Version)
	writeUint64(&buf, uint64(tx.Timestamp))
	writeUint64(&buf, uint64(len(tx.Inputs)))

	for _, input := range tx.Inputs {
		writeString(&buf, input.PreviousTransactionID)
		writeUint32(&buf, input.OutputIndex)
	}

	writeUint64(&buf, uint64(len(tx.Outputs)))
	for _, output := range tx.Outputs {
		writeUint64(&buf, output.Amount)
		writeString(&buf, output.Recipient)
	}

	writeString(&buf, tx.PublicKey)
	return buf.Bytes()
}

func (tx *Transaction) IDBytes() []byte {
	return tx.CanonicalBytes()
}

func (tx *Transaction) CanonicalBytes() []byte {
	var buf bytes.Buffer
	_, _ = buf.Write(tx.SigningBytes())
	writeString(&buf, tx.Signature)
	return buf.Bytes()
}

func (tx *Transaction) SerializedSize() int {
	if tx == nil {
		return 0
	}
	return len(tx.CanonicalBytes())
}

func (tx *Transaction) CalculateID() string {
	digest := sha256.Sum256(tx.IDBytes())
	return hex.EncodeToString(digest[:])
}

func (tx *Transaction) canonicalChainID() string {
	if tx != nil && tx.Version == VersionV2 {
		return tx.ChainID
	}
	return config.ChainID
}

func (tx *Transaction) validateNetworkSelf() error {
	switch tx.Version {
	case VersionLegacy:
		if tx.ChainID != "" {
			return fmt.Errorf(
				"%w: legacy transaction must omit chain id",
				ErrInvalidChainID,
			)
		}
	case VersionV2:
		if tx.ChainID == "" {
			return fmt.Errorf(
				"%w: v2 transaction requires chain id",
				ErrInvalidChainID,
			)
		}
	default:
		return fmt.Errorf(
			"%w: got %d",
			ErrInvalidVersion,
			tx.Version,
		)
	}
	return nil
}

func (tx *Transaction) validateExpectedChain(
	expectedChainID string,
) error {
	if expectedChainID == "" {
		return ErrInvalidChainID
	}
	if expectedChainID == config.ChainID {
		if tx.Version != VersionLegacy || tx.ChainID != "" {
			return fmt.Errorf(
				"%w: legacy chain requires version %d without explicit chain id",
				ErrInvalidChainID,
				VersionLegacy,
			)
		}
		return nil
	}
	if tx.Version != VersionV2 ||
		tx.ChainID != expectedChainID {
		return fmt.Errorf(
			"%w: got version=%d chain=%q want version=%d chain=%q",
			ErrInvalidChainID,
			tx.Version,
			tx.ChainID,
			VersionV2,
			expectedChainID,
		)
	}
	return nil
}

func (tx *Transaction) validateUnsignedSelf() error {
	if err := tx.validateNetworkSelf(); err != nil {
		return err
	}
	if tx.Timestamp <= 0 {
		return ErrInvalidTimestamp
	}
	if len(tx.Inputs) == 0 {
		return ErrNoInputs
	}
	if len(tx.Outputs) == 0 {
		return ErrNoOutputs
	}

	if _, err := valdrcrypto.DecodePublicKey(tx.PublicKey); err != nil {
		return ErrInvalidPublicKey
	}

	seen := make(map[string]struct{}, len(tx.Inputs))
	for i, input := range tx.Inputs {
		raw, err := hex.DecodeString(
			input.PreviousTransactionID,
		)
		if err != nil || len(raw) != 32 {
			return fmt.Errorf(
				"%w at index %d",
				ErrInvalidInput,
				i,
			)
		}
		ref := input.PreviousTransactionID +
			":" +
			strconv.FormatUint(
				uint64(input.OutputIndex),
				10,
			)
		if _, exists := seen[ref]; exists {
			return fmt.Errorf(
				"%w: %s",
				ErrDuplicateInput,
				ref,
			)
		}
		seen[ref] = struct{}{}
	}

	var total uint64
	for i, output := range tx.Outputs {
		if output.Amount == 0 ||
			!valdrcrypto.ValidateAddress(output.Recipient) {
			return fmt.Errorf(
				"%w at index %d",
				ErrInvalidOutput,
				i,
			)
		}
		if math.MaxUint64-total < output.Amount {
			return ErrAmountOverflow
		}
		total += output.Amount
	}

	return nil
}

func writeString(buf *bytes.Buffer, value string) {
	writeUint64(buf, uint64(len(value)))
	_, _ = buf.WriteString(value)
}

func writeUint32(buf *bytes.Buffer, value uint32) {
	_ = binary.Write(buf, binary.BigEndian, value)
}

func writeUint64(buf *bytes.Buffer, value uint64) {
	_ = binary.Write(buf, binary.BigEndian, value)
}
