package wallet

import (
	"crypto/ecdsa"
	"errors"
	"math"
	"sort"
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

var (
	ErrWalletIntegrity    = errors.New("wallet integrity check failed")
	ErrInvalidAmount      = errors.New("invalid transaction amount")
	ErrInsufficientFunds  = errors.New("wallet has insufficient funds")
	ErrWalletValueOverflow = errors.New("wallet UTXO value overflow")
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

// CreateTransaction selects this wallet's UTXOs, creates change when needed,
// and signs a normal zero-fee v0.1 transaction.
func (w *Wallet) CreateTransaction(
	available []utxo.UTXO,
	recipient string,
	amount uint64,
	timestamp int64,
) (*transaction.Transaction, error) {
	if amount == 0 {
		return nil, ErrInvalidAmount
	}
	if !valdrcrypto.ValidateAddress(recipient) {
		return nil, valdrcrypto.ErrInvalidAddress
	}
	if _, err := w.Private(); err != nil {
		return nil, err
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

	inputs := make([]transaction.Input, 0)
	var selected uint64
	for _, item := range candidates {
		if math.MaxUint64-selected < item.Amount {
			return nil, ErrWalletValueOverflow
		}
		selected += item.Amount
		inputs = append(inputs, transaction.Input{
			PreviousTransactionID: item.TransactionID,
			OutputIndex:           item.OutputIndex,
		})
		if selected >= amount {
			break
		}
	}

	if selected < amount {
		return nil, ErrInsufficientFunds
	}

	outputs := []transaction.Output{{
		Amount:    amount,
		Recipient: recipient,
	}}
	if selected > amount {
		outputs = append(outputs, transaction.Output{
			Amount:    selected - amount,
			Recipient: w.Address,
		})
	}

	tx := transaction.New(inputs, outputs, timestamp)
	key, err := w.Private()
	if err != nil {
		return nil, err
	}
	if err := tx.Sign(key); err != nil {
		return nil, err
	}
	return tx, nil
}
