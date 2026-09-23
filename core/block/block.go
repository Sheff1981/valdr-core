package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/transaction"
)

const (
	blockVersion      = uint32(1)
	MaxSerializedSize = 1_000_000
)

type Block struct {
	Version           uint32                     `json:"version"`
	Height            uint64                     `json:"height"`
	PreviousBlockHash string                     `json:"previous_block_hash"`
	MerkleRoot        string                     `json:"merkle_root"`
	Timestamp         int64                      `json:"timestamp"`
	Difficulty        uint64                     `json:"difficulty"`
	Nonce             uint64                     `json:"nonce"`
	Transactions      []*transaction.Transaction `json:"transactions"`
	BlockHash         string                     `json:"block_hash"`
	ChainID           string                     `json:"chain_id"`
	ExtraData         string                     `json:"extra_data,omitempty"`
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

func (b *Block) HeaderBytes() []byte {
	var buf bytes.Buffer
	writeUint32(&buf, b.Version)
	writeUint64(&buf, b.Height)
	writeString(&buf, b.PreviousBlockHash)
	writeString(&buf, b.MerkleRoot)
	writeUint64(&buf, uint64(b.Timestamp))
	writeUint64(&buf, b.Difficulty)
	writeUint64(&buf, b.Nonce)
	writeString(&buf, b.ChainID)
	writeString(&buf, b.ExtraData)
	return buf.Bytes()
}

func (b *Block) CalculateHash() string {
	digest := sha256.Sum256(b.HeaderBytes())
	return hex.EncodeToString(digest[:])
}

// CanonicalBytes is the consensus serialization used only for block-size
// accounting. BlockHash is derived from HeaderBytes and is not serialized.
// Each transaction is length-prefixed so the stream is unambiguous.
func (b *Block) CanonicalBytes() []byte {
	if b == nil {
		return nil
	}
	var buf bytes.Buffer
	_, _ = buf.Write(b.HeaderBytes())
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
