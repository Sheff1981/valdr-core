package wallet

import (
	"crypto/ecdsa"
	"errors"
	"time"

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

var ErrWalletIntegrity = errors.New("wallet integrity check failed")

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
