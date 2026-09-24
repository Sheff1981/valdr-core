package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"

	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
	"golang.org/x/crypto/scrypt"
)

const (
	WalletFileVersionV2 = 2

	walletV2ScryptN   = 1 << 15
	walletV2ScryptR   = 8
	walletV2ScryptP   = 1
	walletV2KeyBytes  = 32
	walletV2SaltBytes = 16
)

var (
	ErrPassphraseRequired      = errors.New("wallet passphrase is required")
	ErrWalletAuthentication    = errors.New("wallet authentication failed")
	ErrWalletMigrationRequired = errors.New("legacy plaintext wallet must be migrated")
	ErrWalletAlreadyV2         = errors.New("wallet is already encrypted v2")
	ErrUnsupportedWalletFile   = errors.New("unsupported wallet file format")
)

type WalletKDFV2 struct {
	Name   string `json:"name"`
	N      int    `json:"n"`
	R      int    `json:"r"`
	P      int    `json:"p"`
	KeyLen int    `json:"key_len"`
	Salt   string `json:"salt"`
}

type WalletCipherV2 struct {
	Name       string `json:"name"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

type WalletFileV2 struct {
	Version   int            `json:"version"`
	Name      string         `json:"name"`
	Address   string         `json:"address"`
	PublicKey string         `json:"public_key"`
	CreatedAt time.Time      `json:"created_at"`
	KDF       WalletKDFV2    `json:"kdf"`
	Cipher    WalletCipherV2 `json:"cipher"`
}

type walletPrivatePayloadV2 struct {
	PrivateKey string `json:"private_key"`
}

type walletAADV2 struct {
	Version   int       `json:"version"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	PublicKey string    `json:"public_key"`
	CreatedAt time.Time `json:"created_at"`
}

func encryptWalletV2(w *Wallet, passphrase []byte) (*WalletFileV2, error) {
	if w == nil {
		return nil, ErrWalletIntegrity
	}
	if len(passphrase) == 0 {
		return nil, ErrPassphraseRequired
	}
	if _, err := w.Private(); err != nil {
		return nil, err
	}

	salt := make([]byte, walletV2SaltBytes)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	key, err := scrypt.Key(
		passphrase,
		salt,
		walletV2ScryptN,
		walletV2ScryptR,
		walletV2ScryptP,
		walletV2KeyBytes,
	)
	if err != nil {
		return nil, err
	}
	defer zeroBytes(key)

	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	payload, err := json.Marshal(walletPrivatePayloadV2{
		PrivateKey: w.PrivateKey,
	})
	if err != nil {
		return nil, err
	}
	defer zeroBytes(payload)

	file := &WalletFileV2{
		Version:   WalletFileVersionV2,
		Name:      w.Name,
		Address:   w.Address,
		PublicKey: w.PublicKey,
		CreatedAt: w.CreatedAt.UTC(),
		KDF: WalletKDFV2{
			Name:   "scrypt",
			N:      walletV2ScryptN,
			R:      walletV2ScryptR,
			P:      walletV2ScryptP,
			KeyLen: walletV2KeyBytes,
			Salt:   hex.EncodeToString(salt),
		},
		Cipher: WalletCipherV2{
			Name:  "aes-256-gcm",
			Nonce: hex.EncodeToString(nonce),
		},
	}
	aad, err := walletV2AAD(file)
	if err != nil {
		return nil, err
	}
	file.Cipher.Ciphertext = hex.EncodeToString(
		gcm.Seal(nil, nonce, payload, aad),
	)
	return file, nil
}

func decryptWalletV2(
	file *WalletFileV2,
	passphrase []byte,
) (*Wallet, error) {
	if file == nil || file.Version != WalletFileVersionV2 {
		return nil, ErrUnsupportedWalletFile
	}
	if len(passphrase) == 0 {
		return nil, ErrPassphraseRequired
	}
	if err := validateWalletV2Parameters(file); err != nil {
		return nil, err
	}
	if err := validateMetadata(file.Metadata()); err != nil {
		return nil, err
	}

	salt, err := hex.DecodeString(file.KDF.Salt)
	if err != nil || len(salt) != walletV2SaltBytes {
		return nil, ErrUnsupportedWalletFile
	}
	nonce, err := hex.DecodeString(file.Cipher.Nonce)
	if err != nil {
		return nil, ErrUnsupportedWalletFile
	}
	ciphertext, err := hex.DecodeString(file.Cipher.Ciphertext)
	if err != nil {
		return nil, ErrUnsupportedWalletFile
	}

	key, err := scrypt.Key(
		passphrase,
		salt,
		file.KDF.N,
		file.KDF.R,
		file.KDF.P,
		file.KDF.KeyLen,
	)
	if err != nil {
		return nil, err
	}
	defer zeroBytes(key)

	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrUnsupportedWalletFile
	}
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, ErrUnsupportedWalletFile
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, ErrUnsupportedWalletFile
	}

	aad, err := walletV2AAD(file)
	if err != nil {
		return nil, err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, ErrWalletAuthentication
	}
	defer zeroBytes(plaintext)

	var payload walletPrivatePayloadV2
	if err := json.Unmarshal(plaintext, &payload); err != nil ||
		payload.PrivateKey == "" {
		return nil, ErrWalletAuthentication
	}

	w := &Wallet{
		Name:       file.Name,
		Address:    file.Address,
		PublicKey:  file.PublicKey,
		PrivateKey: payload.PrivateKey,
		CreatedAt:  file.CreatedAt.UTC(),
	}
	if _, err := w.Private(); err != nil {
		return nil, ErrWalletAuthentication
	}
	return w, nil
}

func (f *WalletFileV2) Metadata() Metadata {
	if f == nil {
		return Metadata{}
	}
	return Metadata{
		Name:      f.Name,
		Address:   f.Address,
		PublicKey: f.PublicKey,
		CreatedAt: f.CreatedAt.UTC(),
	}
}

func validateWalletV2Parameters(file *WalletFileV2) error {
	if file.KDF.Name != "scrypt" ||
		file.KDF.N != walletV2ScryptN ||
		file.KDF.R != walletV2ScryptR ||
		file.KDF.P != walletV2ScryptP ||
		file.KDF.KeyLen != walletV2KeyBytes ||
		file.Cipher.Name != "aes-256-gcm" {
		return ErrUnsupportedWalletFile
	}
	return nil
}

func walletV2AAD(file *WalletFileV2) ([]byte, error) {
	return json.Marshal(walletAADV2{
		Version:   file.Version,
		Name:      file.Name,
		Address:   file.Address,
		PublicKey: file.PublicKey,
		CreatedAt: file.CreatedAt.UTC(),
	})
}

func validateMetadata(meta Metadata) error {
	if meta.Address == "" || meta.PublicKey == "" {
		return ErrWalletIntegrity
	}
	publicKey, err := valdrcrypto.DecodePublicKey(meta.PublicKey)
	if err != nil {
		return ErrWalletIntegrity
	}
	address, err := valdrcrypto.AddressFromPublicKey(publicKey)
	if err != nil ||
		address != meta.Address ||
		!valdrcrypto.ValidateAddress(meta.Address) {
		return ErrWalletIntegrity
	}
	return nil
}

func zeroBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
