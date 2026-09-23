package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"

	"github.com/Sheff1981/valdr-core/config"
)

const blockVersion = uint32(1)

// Block is the minimal VALDR v0.1 block representation defined by the master specification.
// Transactions are opaque strings during Day 2 and will be replaced by the transaction model
// when the Transaction Engine is implemented on Day 5.
type Block struct {
	Version           uint32   `json:"version"`
	Height            uint64   `json:"height"`
	PreviousBlockHash string   `json:"previous_block_hash"`
	MerkleRoot        string   `json:"merkle_root"`
	Timestamp         int64    `json:"timestamp"`
	Difficulty        uint64   `json:"difficulty"`
	Nonce             uint64   `json:"nonce"`
	Transactions      []string `json:"transactions"`
	BlockHash         string   `json:"block_hash"`
	ChainID           string   `json:"chain_id"`
	ExtraData         string   `json:"extra_data,omitempty"`
}

// New creates a VALDR block and calculates its SHA-256 block hash from the header.
func New(height uint64, previousBlockHash string, timestamp int64, difficulty uint64, nonce uint64, transactions []string, extraData string) *Block {
	copiedTransactions := append([]string(nil), transactions...)
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

// NewGenesis returns the fixed VALDR devnet genesis block.
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

// HeaderBytes returns the canonical bytes used to calculate a block hash.
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

// CalculateHash returns a single SHA-256 hash of the canonical block header.
func (b *Block) CalculateHash() string {
	digest := sha256.Sum256(b.HeaderBytes())
	return hex.EncodeToString(digest[:])
}

// CalculateMerkleRoot returns a deterministic SHA-256 Merkle root for opaque transaction data.
// An empty block uses SHA-256 of the empty byte sequence as its root.
func CalculateMerkleRoot(transactions []string) string {
	if len(transactions) == 0 {
		digest := sha256.Sum256(nil)
		return hex.EncodeToString(digest[:])
	}

	level := make([][sha256.Size]byte, len(transactions))
	for i, tx := range transactions {
		level[i] = sha256.Sum256([]byte(tx))
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
