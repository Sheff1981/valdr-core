package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/transaction"
)

const (
	VersionLegacy     = uint32(1)
	VersionV2         = uint32(2)
	blockVersion      = VersionLegacy
	MaxSerializedSize = 1_000_000
	TargetHexLength   = 64
)

var (
	ErrInvalidTargetEncoding = errors.New("invalid v2 block target encoding")
	ErrGenesisNotFrozen      = errors.New("network Genesis is not frozen")
)

type Block struct {
	Version           uint32                     `json:"version"`
	Height            uint64                     `json:"height"`
	PreviousBlockHash string                     `json:"previous_block_hash"`
	MerkleRoot        string                     `json:"merkle_root"`
	Timestamp         int64                      `json:"timestamp"`
	Difficulty        uint64                     `json:"difficulty"`
	Target            string                     `json:"target,omitempty"`
	Nonce             uint64                     `json:"nonce"`
	Transactions      []*transaction.Transaction `json:"transactions"`
	BlockHash         string                     `json:"block_hash"`
	ChainID           string                     `json:"chain_id"`
	ExtraData         string                     `json:"extra_data,omitempty"`
}

type Header struct {
	Version           uint32 `json:"version"`
	Height            uint64 `json:"height"`
	PreviousBlockHash string `json:"previous_block_hash"`
	MerkleRoot        string `json:"merkle_root"`
	Timestamp         int64  `json:"timestamp"`
	Difficulty        uint64 `json:"difficulty"`
	Target            string `json:"target,omitempty"`
	Nonce             uint64 `json:"nonce"`
	BlockHash         string `json:"block_hash"`
	ChainID           string `json:"chain_id"`
	ExtraData         string `json:"extra_data,omitempty"`
}

func New(
	height uint64,
	previousBlockHash string,
	timestamp int64,
	difficulty uint64,
	nonce uint64,
	transactions []*transaction.Transaction,
	extraData string,
) *Block {
	copiedTransactions := append([]*transaction.Transaction(nil), transactions...)
	b := &Block{
		Version:           blockVersion,
		Height:            height,
		PreviousBlockHash: previousBlockHash,
		MerkleRoot:        CalculateMerkleRoot(copiedTransactions),
		Timestamp:         timestamp,
		Difficulty:        difficulty,
		Nonce:             nonce,
		Transactions:      copiedTransactions,
		ChainID:           config.ChainID,
		ExtraData:         extraData,
	}
	b.BlockHash = b.CalculateHash()
	return b
}

func NewV2(
	height uint64,
	previousBlockHash string,
	timestamp int64,
	target string,
	nonce uint64,
	transactions []*transaction.Transaction,
	chainID string,
	extraData string,
) (*Block, error) {
	if _, err := DecodeTarget(target); err != nil {
		return nil, err
	}
	copiedTransactions := append([]*transaction.Transaction(nil), transactions...)
	b := &Block{
		Version:           VersionV2,
		Height:            height,
		PreviousBlockHash: previousBlockHash,
		MerkleRoot:        CalculateMerkleRoot(copiedTransactions),
		Timestamp:         timestamp,
		Difficulty:        0,
		Target:            target,
		Nonce:             nonce,
		Transactions:      copiedTransactions,
		ChainID:           chainID,
		ExtraData:         extraData,
	}
	b.BlockHash = b.CalculateHash()
	if b.BlockHash == "" {
		return nil, ErrInvalidTargetEncoding
	}
	return b, nil
}

func NewGenesis() *Block {
	b := New(
		0,
		"",
		config.GenesisTimestamp,
		config.GenesisDifficulty,
		config.GenesisNonce,
		nil,
		config.GenesisMessage,
	)
	b.Version = config.GenesisVersion
	b.BlockHash = b.CalculateHash()
	return b
}

func NewGenesisForProfile(profile config.NetworkProfile) (*Block, error) {
	switch profile.BlockVersion {
	case VersionLegacy:
		if profile.ChainID != config.ChainID ||
			profile.GenesisHash != config.GenesisBlockHash {
			return nil, ErrGenesisNotFrozen
		}
		return NewGenesis(), nil
	case VersionV2:
		if profile.GenesisTarget == "" ||
			profile.GenesisHash == "" ||
			profile.GenesisTimestamp <= 0 ||
			profile.GenesisMessage == "" {
			return nil, ErrGenesisNotFrozen
		}
		genesis, err := NewV2(
			0,
			"",
			profile.GenesisTimestamp,
			profile.GenesisTarget,
			profile.GenesisNonce,
			nil,
			profile.ChainID,
			profile.GenesisMessage,
		)
		if err != nil {
			return nil, err
		}
		if genesis.BlockHash != profile.GenesisHash {
			return nil, fmt.Errorf(
				"%w: computed=%s frozen=%s",
				ErrGenesisNotFrozen,
				genesis.BlockHash,
				profile.GenesisHash,
			)
		}
		return genesis, nil
	default:
		return nil, ErrGenesisNotFrozen
	}
}

func (b *Block) Header() Header {
	if b == nil {
		return Header{}
	}
	return Header{
		Version:           b.Version,
		Height:            b.Height,
		PreviousBlockHash: b.PreviousBlockHash,
		MerkleRoot:        b.MerkleRoot,
		Timestamp:         b.Timestamp,
		Difficulty:        b.Difficulty,
		Target:            b.Target,
		Nonce:             b.Nonce,
		BlockHash:         b.BlockHash,
		ChainID:           b.ChainID,
		ExtraData:         b.ExtraData,
	}
}

func (h Header) HeaderBytesChecked() ([]byte, error) {
	candidate := &Block{
		Version: h.Version, Height: h.Height,
		PreviousBlockHash: h.PreviousBlockHash, MerkleRoot: h.MerkleRoot,
		Timestamp: h.Timestamp, Difficulty: h.Difficulty, Target: h.Target,
		Nonce: h.Nonce, ChainID: h.ChainID, ExtraData: h.ExtraData,
	}
	return candidate.HeaderBytesChecked()
}

func (h Header) CalculateHash() string {
	raw, err := h.HeaderBytesChecked()
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func (b *Block) HeaderBytes() []byte {
	raw, _ := b.HeaderBytesChecked()
	return raw
}

func (b *Block) HeaderBytesChecked() ([]byte, error) {
	if b == nil {
		return nil, errors.New("nil block")
	}
	var buf bytes.Buffer
	writeUint32(&buf, b.Version)
	writeUint64(&buf, b.Height)
	writeString(&buf, b.PreviousBlockHash)
	writeString(&buf, b.MerkleRoot)
	writeUint64(&buf, uint64(b.Timestamp))

	switch b.Version {
	case VersionLegacy:
		writeUint64(&buf, b.Difficulty)
	case VersionV2:
		if b.Difficulty != 0 {
			return nil, fmt.Errorf("%w: v2 legacy difficulty must be zero", ErrInvalidTargetEncoding)
		}
		target, err := DecodeTarget(b.Target)
		if err != nil {
			return nil, err
		}
		_, _ = buf.Write(target)
	default:
		return nil, fmt.Errorf("unsupported block version %d", b.Version)
	}

	writeUint64(&buf, b.Nonce)
	writeString(&buf, b.ChainID)
	writeString(&buf, b.ExtraData)
	return buf.Bytes(), nil
}

func DecodeTarget(value string) ([]byte, error) {
	if len(value) != TargetHexLength {
		return nil, fmt.Errorf("%w: length=%d want=%d", ErrInvalidTargetEncoding, len(value), TargetHexLength)
	}
	raw, err := hex.DecodeString(value)
	if err != nil || len(raw) != sha256.Size {
		return nil, ErrInvalidTargetEncoding
	}
	if hex.EncodeToString(raw) != value {
		return nil, ErrInvalidTargetEncoding
	}
	return raw, nil
}

func (b *Block) CalculateHash() string {
	raw, err := b.HeaderBytesChecked()
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

// CanonicalBytes is the consensus serialization used only for block-size
// accounting. BlockHash is derived from HeaderBytes and is not serialized.
// Each transaction is length-prefixed so the stream is unambiguous.
func (b *Block) CanonicalBytes() []byte {
	if b == nil {
		return nil
	}
	header, err := b.HeaderBytesChecked()
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	_, _ = buf.Write(header)
	writeUint64(&buf, uint64(len(b.Transactions)))
	for _, tx := range b.Transactions {
		if tx == nil {
			writeUint64(&buf, 0)
			continue
		}
		raw := tx.CanonicalBytes()
		writeUint64(&buf, uint64(len(raw)))
		_, _ = buf.Write(raw)
	}
	return buf.Bytes()
}

func (b *Block) SerializedSize() int {
	if b == nil {
		return 0
	}
	return len(b.CanonicalBytes())
}

func CalculateMerkleRoot(transactions []*transaction.Transaction) string {
	if len(transactions) == 0 {
		digest := sha256.Sum256(nil)
		return hex.EncodeToString(digest[:])
	}

	level := make([][sha256.Size]byte, len(transactions))
	for i, tx := range transactions {
		level[i] = transactionLeaf(tx)
	}

	for len(level) > 1 {
		if len(level)%2 == 1 {
			level = append(level, level[len(level)-1])
		}
		next := make([][sha256.Size]byte, 0, len(level)/2)
		for i := 0; i < len(level); i += 2 {
			combined := make([]byte, 0, sha256.Size*2)
			combined = append(combined, level[i][:]...)
			combined = append(combined, level[i+1][:]...)
			next = append(next, sha256.Sum256(combined))
		}
		level = next
	}

	return hex.EncodeToString(level[0][:])
}

func transactionLeaf(tx *transaction.Transaction) [sha256.Size]byte {
	if tx == nil {
		return sha256.Sum256(nil)
	}

	raw, err := hex.DecodeString(tx.TransactionID)
	if err == nil && len(raw) == sha256.Size {
		var leaf [sha256.Size]byte
		copy(leaf[:], raw)
		return leaf
	}

	return sha256.Sum256([]byte(tx.TransactionID))
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
