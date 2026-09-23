package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"

	"github.com/Sheff1981/valdr-core/config"
)

const (
	privateKeyBytes      = 32
	addressHashBytes     = 20
	addressChecksumBytes = 4
)

var (
	ErrInvalidPrivateKey = errors.New("invalid private key")
	ErrInvalidPublicKey  = errors.New("invalid public key")
	ErrInvalidAddress    = errors.New("invalid VALDR address")
)

func GenerateKeyPair() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

func EncodePrivateKey(key *ecdsa.PrivateKey) (string, error) {
	if key == nil || key.D == nil || key.D.Sign() <= 0 {
		return "", ErrInvalidPrivateKey
	}
	buf := make([]byte, privateKeyBytes)
	key.D.FillBytes(buf)
	return hex.EncodeToString(buf), nil
}

func DecodePrivateKey(encoded string) (*ecdsa.PrivateKey, error) {
	raw, err := hex.DecodeString(encoded)
	if err != nil || len(raw) != privateKeyBytes {
		return nil, ErrInvalidPrivateKey
	}
	curve := elliptic.P256()
	d := new(big.Int).SetBytes(raw)
	if d.Sign() <= 0 || d.Cmp(curve.Params().N) >= 0 {
		return nil, ErrInvalidPrivateKey
	}
	x, y := curve.ScalarBaseMult(raw)
	return &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{Curve: curve, X: x, Y: y},
		D:         d,
	}, nil
}

func EncodePublicKey(key *ecdsa.PublicKey) (string, error) {
	if key == nil || key.X == nil || key.Y == nil || key.Curve == nil || !key.Curve.IsOnCurve(key.X, key.Y) {
		return "", ErrInvalidPublicKey
	}
	return hex.EncodeToString(elliptic.Marshal(key.Curve, key.X, key.Y)), nil
}

func DecodePublicKey(encoded string) (*ecdsa.PublicKey, error) {
	raw, err := hex.DecodeString(encoded)
	if err != nil {
		return nil, ErrInvalidPublicKey
	}
	curve := elliptic.P256()
	x, y := elliptic.Unmarshal(curve, raw)
	if x == nil || y == nil || !curve.IsOnCurve(x, y) {
		return nil, ErrInvalidPublicKey
	}
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}

func Sign(key *ecdsa.PrivateKey, message []byte) ([]byte, error) {
	if key == nil || key.D == nil {
		return nil, ErrInvalidPrivateKey
	}
	digest := sha256.Sum256(message)
	return ecdsa.SignASN1(rand.Reader, key, digest[:])
}

func Verify(key *ecdsa.PublicKey, message, signature []byte) bool {
	if key == nil || key.X == nil || key.Y == nil || key.Curve == nil || !key.Curve.IsOnCurve(key.X, key.Y) {
		return false
	}
	digest := sha256.Sum256(message)
	return ecdsa.VerifyASN1(key, digest[:], signature)
}

func AddressFromPublicKey(key *ecdsa.PublicKey) (string, error) {
	encoded, err := EncodePublicKey(key)
	if err != nil {
		return "", err
	}
	raw, _ := hex.DecodeString(encoded)
	payloadDigest := sha256.Sum256(raw)
	payload := payloadDigest[:addressHashBytes]
	checksum := addressChecksum(payload)

	data := make([]byte, 0, len(payload)+len(checksum))
	data = append(data, payload...)
	data = append(data, checksum...)

	body := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(data)
	return config.AddressPrefix + body, nil
}

func ValidateAddress(address string) bool {
	if !strings.HasPrefix(address, config.AddressPrefix) {
		return false
	}

	body := strings.TrimPrefix(address, config.AddressPrefix)
	raw, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(body))
	if err != nil || len(raw) != addressHashBytes+addressChecksumBytes {
		return false
	}

	payload := raw[:addressHashBytes]
	checksum := raw[addressHashBytes:]
	expected := addressChecksum(payload)
	for i := range checksum {
		if checksum[i] != expected[i] {
			return false
		}
	}
	return true
}

func addressChecksum(payload []byte) []byte {
	data := make([]byte, 0, len(config.ChainID)+len(payload))
	data = append(data, []byte(config.ChainID)...)
	data = append(data, payload...)
	digest := sha256.Sum256(data)
	return digest[:addressChecksumBytes]
}
