package wallet

import (
	"crypto/ecdsa"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/core/utxo"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

type Wallet struct {
	Name       string    `json:"name"`
	Address    string    `json:"address"`
	PublicKey  string    `json:"public_key"`
	PrivateKey string    `json:"private_key"`
	CreatedAt  time.Time `json:"created_at"`
}

type Metadata struct {
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	PublicKey string    `json:"public_key"`
	CreatedAt time.Time `json:"created_at"`
}

const maxP256DERSignatureHexLength = 144

var (
	ErrWalletIntegrity     = errors.New("wallet integrity check failed")
	ErrInvalidAmount       = errors.New("invalid transaction amount")
	ErrInsufficientFunds   = errors.New("wallet has insufficient funds")
	ErrWalletValueOverflow = errors.New("wallet UTXO value overflow")
	ErrWalletFeeOverflow   = errors.New("wallet transaction fee overflow")
)

func New(name string) (*Wallet, error) {
	privateKey, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		return nil, err
	}
	privateHex, err := valdrcrypto.EncodePrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	publicHex, err := valdrcrypto.EncodePublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}
	address, err := valdrcrypto.AddressFromPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = "wallet-" + address[4:12]
	}
	return &Wallet{
		Name:       name,
		Address:    address,
		PublicKey:  publicHex,
		PrivateKey: privateHex,
		CreatedAt:  time.Now().UTC(),
	}, nil
}

func (w *Wallet) Metadata() Metadata {
	return Metadata{
		Name:      w.Name,
		Address:   w.Address,
		PublicKey: w.PublicKey,
		CreatedAt: w.CreatedAt,
	}
}

func (w *Wallet) Private() (*ecdsa.PrivateKey, error) {
	key, err := valdrcrypto.DecodePrivateKey(w.PrivateKey)
	if err != nil {
		return nil, err
	}

	publicHex, err := valdrcrypto.EncodePublicKey(&key.PublicKey)
	if err != nil {
		return nil, err
	}
	address, err := valdrcrypto.AddressFromPublicKey(&key.PublicKey)
	if err != nil {
		return nil, err
	}
	if publicHex != w.PublicKey || address != w.Address || !valdrcrypto.ValidateAddress(w.Address) {
		return nil, ErrWalletIntegrity
	}
	return key, nil
}

func (w *Wallet) Sign(message []byte) ([]byte, error) {
	key, err := w.Private()
	if err != nil {
		return nil, err
	}
	return valdrcrypto.Sign(key, message)
}

// CreateTransaction preserves the legacy zero-fee wallet API. Testnet and
// other policy-aware callers must use CreateTransactionWithFeeRate.
func (w *Wallet) CreateTransaction(
	available []utxo.UTXO,
	recipient string,
	amount uint64,
	timestamp int64,
) (*transaction.Transaction, error) {
	tx, _, err := w.CreateTransactionWithFeeRate(
		available,
		recipient,
		amount,
		0,
		timestamp,
	)
	return tx, err
}

// CreateTransactionWithFeeRate deterministically selects this wallet's UTXOs
// and reserves at least minFeePerByte * canonical serialized bytes as the
// implicit fee. The estimate reserves the maximum P-256 DER signature length,
// so the final signed transaction cannot fall below the requested fee rate
// merely because the DER signature length varies.
func (w *Wallet) CreateTransactionWithFeeRate(
	available []utxo.UTXO,
	recipient string,
	amount uint64,
	minFeePerByte uint64,
	timestamp int64,
) (*transaction.Transaction, uint64, error) {
	if amount == 0 {
		return nil, 0, ErrInvalidAmount
	}
	if !valdrcrypto.ValidateAddress(recipient) {
		return nil, 0, valdrcrypto.ErrInvalidAddress
	}
	key, err := w.Private()
	if err != nil {
		return nil, 0, err
	}
	publicHex, err := valdrcrypto.EncodePublicKey(&key.PublicKey)
	if err != nil {
		return nil, 0, err
	}

	candidates := make([]utxo.UTXO, 0, len(available))
	for _, item := range available {
		if item.Recipient == w.Address {
			candidates = append(candidates, item)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].TransactionID == candidates[j].TransactionID {
			return candidates[i].OutputIndex < candidates[j].OutputIndex
		}
		return candidates[i].TransactionID < candidates[j].TransactionID
	})

	inputs := make([]transaction.Input, 0, len(candidates))
	var selected uint64
	for _, item := range candidates {
		if math.MaxUint64-selected < item.Amount {
			return nil, 0, ErrWalletValueOverflow
		}
		selected += item.Amount
		inputs = append(inputs, transaction.Input{
			PreviousTransactionID: item.TransactionID,
			OutputIndex:           item.OutputIndex,
		})

		outputs, fee, ok, err := feeAwareOutputs(
			inputs,
			publicHex,
			recipient,
			w.Address,
			amount,
			selected,
			minFeePerByte,
			timestamp,
		)
		if err != nil {
			return nil, 0, err
		}
		if !ok {
			continue
		}

		tx := transaction.New(inputs, outputs, timestamp)
		if err := tx.Sign(key); err != nil {
			return nil, 0, err
		}
		actualFee, err := implicitFee(selected, tx.Outputs)
		if err != nil {
			return nil, 0, err
		}
		required, err := feeForSize(minFeePerByte, tx.SerializedSize())
		if err != nil {
			return nil, 0, err
		}
		if actualFee < required {
			return nil, 0, ErrWalletFeeOverflow
		}
		if actualFee < fee {
			return nil, 0, ErrWalletFeeOverflow
		}
		return tx, actualFee, nil
	}

	return nil, 0, ErrInsufficientFunds
}

func feeAwareOutputs(
	inputs []transaction.Input,
	publicHex string,
	recipient string,
	changeAddress string,
	amount uint64,
	selected uint64,
	minFeePerByte uint64,
	timestamp int64,
) ([]transaction.Output, uint64, bool, error) {
	recipientOutput := transaction.Output{
		Amount:    amount,
		Recipient: recipient,
	}

	// Prefer change. Amount is fixed-width in canonical serialization, so a
	// one-val placeholder has exactly the same byte size as the final change.
	withChange := []transaction.Output{
		recipientOutput,
		{Amount: 1, Recipient: changeAddress},
	}
	changeFee, err := estimatedSignedFee(
		inputs,
		withChange,
		publicHex,
		minFeePerByte,
		timestamp,
	)
	if err != nil {
		return nil, 0, false, err
	}
	requiredWithChange, overflow := addUint64(amount, changeFee)
	if !overflow && selected > requiredWithChange {
		withChange[1].Amount = selected - requiredWithChange
		return withChange, changeFee, true, nil
	}

	// If a selected UTXO cannot fund a positive change output, allow a
	// one-output transaction. Any remainder becomes additional fee rather than
	// creating a zero-value output.
	oneOutput := []transaction.Output{recipientOutput}
	minimumFee, err := estimatedSignedFee(
		inputs,
		oneOutput,
		publicHex,
		minFeePerByte,
		timestamp,
	)
	if err != nil {
		return nil, 0, false, err
	}
	requiredOne, overflow := addUint64(amount, minimumFee)
	if overflow || selected < requiredOne {
		return nil, 0, false, nil
	}
	return oneOutput, selected - amount, true, nil
}

func estimatedSignedFee(
	inputs []transaction.Input,
	outputs []transaction.Output,
	publicHex string,
	minFeePerByte uint64,
	timestamp int64,
) (uint64, error) {
	tx := transaction.New(inputs, outputs, timestamp)
	tx.PublicKey = publicHex
	tx.Signature = strings.Repeat("0", maxP256DERSignatureHexLength)
	return feeForSize(minFeePerByte, tx.SerializedSize())
}

func feeForSize(rate uint64, size int) (uint64, error) {
	if rate == 0 || size == 0 {
		return 0, nil
	}
	if size < 0 || uint64(size) > math.MaxUint64/rate {
		return 0, ErrWalletFeeOverflow
	}
	return rate * uint64(size), nil
}

func implicitFee(
	selected uint64,
	outputs []transaction.Output,
) (uint64, error) {
	var total uint64
	for _, output := range outputs {
		if math.MaxUint64-total < output.Amount {
			return 0, ErrWalletValueOverflow
		}
		total += output.Amount
	}
	if total > selected {
		return 0, ErrWalletFeeOverflow
	}
	return selected - total, nil
}

func addUint64(a, b uint64) (uint64, bool) {
	if math.MaxUint64-a < b {
		return 0, true
	}
	return a + b, false
}
