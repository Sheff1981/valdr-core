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

const Version = uint32(1)

var (
	ErrInvalidVersion       = errors.New("invalid transaction version")
	ErrNoInputs             = errors.New("transaction has no inputs")
	ErrNoOutputs            = errors.New("transaction has no outputs")
	ErrInvalidInput         = errors.New("invalid transaction input")
	ErrDuplicateInput       = errors.New("duplicate transaction input")
	ErrInvalidOutput        = errors.New("invalid transaction output")
	ErrAmountOverflow       = errors.New("transaction output amount overflow")
	ErrInvalidTimestamp     = errors.New("invalid transaction timestamp")
	ErrInvalidPublicKey     = errors.New("invalid transaction public key")
	ErrInvalidSignature     = errors.New("invalid transaction signature")
	ErrInvalidTransactionID = errors.New("invalid transaction id")
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
	Inputs        []Input  `json:"inputs"`
	Outputs       []Output `json:"outputs"`
	Timestamp     int64    `json:"timestamp"`
	PublicKey     string   `json:"public_key"`
	Signature     string   `json:"signature"`
	TransactionID string   `json:"transaction_id"`
}

func New(inputs []Input, outputs []Output, timestamp int64) *Transaction {
	return &Transaction{
		Version:   Version,
		Inputs:    append([]Input(nil), inputs...),
		Outputs:   append([]Output(nil), outputs...),
		Timestamp: timestamp,
	}
}

func (tx *Transaction) Sign(key *ecdsa.PrivateKey) error {
	if tx == nil {
		return ErrInvalidTransactionID
	}
	if key == nil {
		return valdrcrypto.ErrInvalidPrivateKey
	}

	publicKey, err := valdrcrypto.EncodePublicKey(&key.PublicKey)
	if err != nil {
		return err
	}

	tx.PublicKey = publicKey
	tx.Signature = ""
	tx.TransactionID = ""

	if err := tx.validateUnsigned(); err != nil {
		return err
	}

	signature, err := valdrcrypto.Sign(key, tx.SigningBytes())
	if err != nil {
		return err
	}

	tx.Signature = hex.EncodeToString(signature)
	tx.TransactionID = tx.CalculateID()
	return nil
}

func (tx *Transaction) VerifySignature() bool {
	if tx == nil {
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

	return valdrcrypto.Verify(publicKey, tx.SigningBytes(), signature)
}

func (tx *Transaction) Validate() error {
	if tx == nil {
		return ErrInvalidTransactionID
	}
	if err := tx.validateUnsigned(); err != nil {
		return err
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

func (tx *Transaction) SigningBytes() []byte {
	var buf bytes.Buffer
	writeString(&buf, config.ChainID)
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
	var buf bytes.Buffer
	_, _ = buf.Write(tx.SigningBytes())
	writeString(&buf, tx.Signature)
	return buf.Bytes()
}

func (tx *Transaction) CalculateID() string {
	digest := sha256.Sum256(tx.IDBytes())
	return hex.EncodeToString(digest[:])
}

func (tx *Transaction) validateUnsigned() error {
	if tx.Version != Version {
		return fmt.Errorf("%w: got %d want %d", ErrInvalidVersion, tx.Version, Version)
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
		raw, err := hex.DecodeString(input.PreviousTransactionID)
		if err != nil || len(raw) != 32 {
			return fmt.Errorf("%w at index %d", ErrInvalidInput, i)
		}
		ref := input.PreviousTransactionID + ":" + strconv.FormatUint(uint64(input.OutputIndex), 10)
		if _, exists := seen[ref]; exists {
			return fmt.Errorf("%w: %s", ErrDuplicateInput, ref)
		}
		seen[ref] = struct{}{}
	}

	var total uint64
	for i, output := range tx.Outputs {
		if output.Amount == 0 || !valdrcrypto.ValidateAddress(output.Recipient) {
			return fmt.Errorf("%w at index %d", ErrInvalidOutput, i)
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
