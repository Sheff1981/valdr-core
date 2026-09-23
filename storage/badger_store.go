package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"sort"

	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/core/utxo"
	badger "github.com/dgraph-io/badger/v4"
)

const (
	StorageSchemaVersion = uint32(2)
	badgerDirName         = "chain-v2"
)

var (
	ErrBadgerStoreClosed      = errors.New("Badger store is closed")
	ErrStorageSchemaMismatch  = errors.New("storage schema mismatch")
	ErrStorageNetworkMismatch = errors.New("storage network mismatch")
	ErrStorageGenesisMismatch = errors.New("storage genesis mismatch")
	ErrStorageMetadataCorrupt = errors.New("storage metadata is corrupt")
	ErrStorageStateMismatch   = errors.New("storage state mismatch")
	ErrMigrationIncomplete    = errors.New("storage migration is incomplete")
	ErrMigrationTargetExists  = errors.New("migration target already exists")
)

var (
	keySchemaVersion      = []byte("meta/schema-version")
	keyNetwork            = []byte("meta/network")
	keyGenesis            = []byte("meta/genesis")
	keyActiveTip          = []byte("meta/active-tip")
	keyActiveHeight       = []byte("meta/height")
	keyActiveChainwork    = []byte("meta/chainwork")
	keyUTXOHash           = []byte("meta/utxo-hash")
	keyMigrationStatus    = []byte("migration/status")
	keyMigrationSource    = []byte("migration/source")
	keyMigrationSourceTip = []byte("migration/source-tip")
)

type BadgerStore struct {
	db          *badger.DB
	path        string
	network     string
	genesisHash string
}

type StorageInfo struct {
	SchemaVersion   uint32 `json:"schema_version"`
	Network         string `json:"network"`
	GenesisHash     string `json:"genesis_hash"`
	ActiveTip       string `json:"active_tip"`
	Height          uint64 `json:"height"`
	Chainwork       string `json:"chainwork,omitempty"`
	UTXOHash        string `json:"utxo_hash"`
	MigrationStatus string `json:"migration_status,omitempty"`
}

type headerRecord struct {
	Parent      string `json:"parent"`
	Height      uint64 `json:"height"`
	Difficulty  uint64 `json:"difficulty"`
	Status      string `json:"status"`
	Target      string `json:"target,omitempty"`
	Chainwork   string `json:"chainwork,omitempty"`
}

type transactionRecord struct {
	BlockHeight uint64                   `json:"block_height"`
	BlockHash   string                   `json:"block_hash"`
	Index       uint32                   `json:"index"`
	Transaction *transaction.Transaction `json:"transaction"`
}

type undoRecord struct {
	Spent   []utxo.UTXO `json:"spent"`
	Created []Outpoint  `json:"created"`
}

type Outpoint struct {
	TransactionID string `json:"transaction_id"`
	OutputIndex   uint32 `json:"output_index"`
}

func BadgerPath(dataDir string) string {
	return filepath.Join(dataDir, badgerDirName)
}

func NewBadgerStore(dataDir, network, genesisHash string) (*BadgerStore, error) {
	return OpenBadgerStore(BadgerPath(dataDir), network, genesisHash)
}

func OpenBadgerStore(path, network, genesisHash string) (*BadgerStore, error) {
	if path == "" || network == "" || genesisHash == "" {
		return nil, ErrInvalidStorePath
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return nil, err
	}
	db, err := badger.Open(badger.DefaultOptions(path).WithLogger(nil))
	if err != nil {
		return nil, err
	}
	store := &BadgerStore{db: db, path: path, network: network, genesisHash: genesisHash}
	if _, _, err := store.info(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *BadgerStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *BadgerStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

func (s *BadgerStore) Load() ([]*block.Block, error) {
	if s == nil || s.db == nil {
		return nil, ErrBadgerStoreClosed
	}
	info, initialized, err := s.info()
	if err != nil {
		return nil, err
	}
	if !initialized {
		return nil, nil
	}
	if info.MigrationStatus != "" && info.MigrationStatus != "complete" {
		return nil, fmt.Errorf("%w: %s", ErrMigrationIncomplete, info.MigrationStatus)
	}
	blocks := make([]*block.Block, 0, info.Height+1)
	err = s.db.View(func(txn *badger.Txn) error {
		for height := uint64(0); height <= info.Height; height++ {
			hash, err := getString(txn, heightKey(height))
			if err != nil {
				return fmt.Errorf("%w: missing height %d: %v", ErrStorageMetadataCorrupt, height, err)
			}
			raw, err := getBytes(txn, blockKey(hash))
			if err != nil {
				return fmt.Errorf("%w: missing block %s: %v", ErrStorageMetadataCorrupt, hash, err)
			}
			var candidate block.Block
			if err := strictJSON(raw, &candidate); err != nil {
				return fmt.Errorf("%w: block %d: %v", ErrStorageMetadataCorrupt, height, err)
			}
			if candidate.Height != height || candidate.BlockHash != hash {
				return fmt.Errorf("%w: active mapping at height %d", ErrStorageMetadataCorrupt, height)
			}
			blocks = append(blocks, &candidate)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(blocks) == 0 || blocks[len(blocks)-1].BlockHash != info.ActiveTip {
		return nil, ErrStorageMetadataCorrupt
	}
	return blocks, nil
}

// Save initializes a fresh v2 database with exactly the canonical Genesis.
// Later blocks are committed incrementally through SaveBlock.
func (s *BadgerStore) Save(blocks []*block.Block) error {
	if s == nil || s.db == nil {
		return ErrBadgerStoreClosed
	}
	if len(blocks) != 1 || blocks[0] == nil || blocks[0].Height != 0 {
		return fmt.Errorf("%w: Badger initialization requires exactly Genesis", ErrInvalidDiskChain)
	}
	genesis := blocks[0]
	if genesis.BlockHash != s.genesisHash {
		return ErrStorageGenesisMismatch
	}

	return s.db.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(keySchemaVersion); err == nil {
			return fmt.Errorf("%w: database already initialized", ErrStorageStateMismatch)
		} else if !errors.Is(err, badger.ErrKeyNotFound) {
			return err
		}
		rawBlock, err := json.Marshal(genesis)
		if err != nil {
			return err
		}
		target, targetValue, err := blockTargetAndWork(genesis)
		if err != nil {
			return err
		}
		var work *big.Int
		if genesis.Version == block.VersionV2 {
			work, err = consensus.BlockWorkTarget(targetValue)
		} else {
			work, err = consensus.BlockWork(genesis.Difficulty)
		}
		if err != nil {
			return err
		}
		genesisChainwork := consensus.ChainworkHex(work)
		rawHeader, err := json.Marshal(headerRecord{
			Parent: genesis.PreviousBlockHash, Height: genesis.Height,
			Difficulty: genesis.Difficulty, Status: "active",
			Target: target, Chainwork: genesisChainwork,
		})
		if err != nil {
			return err
		}
		sets := []struct{ key, value []byte }{
			{keySchemaVersion, encodeUint32(StorageSchemaVersion)},
			{keyNetwork, []byte(s.network)},
			{keyGenesis, []byte(s.genesisHash)},
			{keyActiveTip, []byte(genesis.BlockHash)},
			{keyActiveHeight, encodeUint64(0)},
			{keyActiveChainwork, []byte(genesisChainwork)},
			{keyUTXOHash, []byte(HashUTXOSet(nil))},
			{blockKey(genesis.BlockHash), rawBlock},
			{heightKey(0), []byte(genesis.BlockHash)},
			{headerKey(genesis.BlockHash), rawHeader},
		}
		for _, item := range sets {
			if err := txn.Set(item.key, item.value); err != nil {
				return err
			}
		}
		return nil
	})
}

// SaveBlock atomically updates all active-chain indexes for one validated block.
func (s *BadgerStore) SaveBlock(candidate *block.Block, before, after []utxo.UTXO) error {
	if s == nil || s.db == nil {
		return ErrBadgerStoreClosed
	}
	if candidate == nil || candidate.Height == 0 {
		return ErrInvalidDiskChain
	}

	beforeHash := HashUTXOSet(before)
	afterHash := HashUTXOSet(after)
	spent, created := diffUTXO(before, after)

	rawBlock, err := json.Marshal(candidate)
	if err != nil {
		return err
	}
	rawHeader, err := json.Marshal(headerRecord{
		Parent: candidate.PreviousBlockHash, Height: candidate.Height,
		Difficulty: candidate.Difficulty, Status: "active",
	})
	if err != nil {
		return err
	}
	rawUndo, err := json.Marshal(undoRecord{Spent: spent, Created: created})
	if err != nil {
		return err
	}

	return s.db.Update(func(txn *badger.Txn) error {
		if err := s.validateIdentityTxn(txn); err != nil {
			return err
		}
		currentHeight, err := getUint64(txn, keyActiveHeight)
		if err != nil {
			return err
		}
		currentTip, err := getString(txn, keyActiveTip)
		if err != nil {
			return err
		}
		currentUTXOHash, err := getString(txn, keyUTXOHash)
		if err != nil {
			return err
		}
		if candidate.Height != currentHeight+1 ||
			candidate.PreviousBlockHash != currentTip ||
			currentUTXOHash != beforeHash {
			return fmt.Errorf("%w: height/tip/utxo mismatch", ErrStorageStateMismatch)
		}

		if err := txn.Set(blockKey(candidate.BlockHash), rawBlock); err != nil {
			return err
		}
		if err := txn.Set(heightKey(candidate.Height), []byte(candidate.BlockHash)); err != nil {
			return err
		}
		if err := txn.Set(headerKey(candidate.BlockHash), rawHeader); err != nil {
			return err
		}
		for index, tx := range candidate.Transactions {
			if tx == nil {
				continue
			}
			rawTx, err := json.Marshal(transactionRecord{
				BlockHeight: candidate.Height, BlockHash: candidate.BlockHash,
				Index: uint32(index), Transaction: tx,
			})
			if err != nil {
				return err
			}
			if err := txn.Set(txKey(tx.TransactionID), rawTx); err != nil {
				return err
			}
		}
		for _, item := range spent {
			if err := txn.Delete(utxoKey(item.TransactionID, item.OutputIndex)); err != nil {
				return err
			}
		}
		afterMap := utxoMap(after)
		for _, outpoint := range created {
			item, ok := afterMap[outpointKeyString(outpoint.TransactionID, outpoint.OutputIndex)]
			if !ok {
				return ErrStorageStateMismatch
			}
			raw, err := json.Marshal(item)
			if err != nil {
				return err
			}
			if err := txn.Set(utxoKey(item.TransactionID, item.OutputIndex), raw); err != nil {
				return err
			}
		}
		if err := txn.Set(undoKey(candidate.BlockHash), rawUndo); err != nil {
			return err
		}
		if err := txn.Set(keyActiveTip, []byte(candidate.BlockHash)); err != nil {
			return err
		}
		if err := txn.Set(keyActiveHeight, encodeUint64(candidate.Height)); err != nil {
			return err
		}
		return txn.Set(keyUTXOHash, []byte(afterHash))
	})
}

func (s *BadgerStore) Info() (StorageInfo, error) {
	info, initialized, err := s.info()
	if err != nil {
		return StorageInfo{}, err
	}
	if !initialized {
		return StorageInfo{}, ErrStorageMetadataCorrupt
	}
	return info, nil
}

func (s *BadgerStore) MarkMigration(status, source, sourceTip string) error {
	if s == nil || s.db == nil {
		return ErrBadgerStoreClosed
	}
	if status != "running" && status != "complete" {
		return fmt.Errorf("invalid migration status %q", status)
	}
	return s.db.Update(func(txn *badger.Txn) error {
		if err := s.validateIdentityTxn(txn); err != nil {
			return err
		}
		if err := txn.Set(keyMigrationStatus, []byte(status)); err != nil {
			return err
		}
		if err := txn.Set(keyMigrationSource, []byte(source)); err != nil {
			return err
		}
		return txn.Set(keyMigrationSourceTip, []byte(sourceTip))
	})
}

func (s *BadgerStore) KeyCount(prefix string) (int, error) {
	if s == nil || s.db == nil {
		return 0, ErrBadgerStoreClosed
	}
	count := 0
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(prefix)
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}
		return nil
	})
	return count, err
}

func (s *BadgerStore) info() (StorageInfo, bool, error) {
	if s == nil || s.db == nil {
		return StorageInfo{}, false, ErrBadgerStoreClosed
	}
	var info StorageInfo
	initialized := false
	err := s.db.View(func(txn *badger.Txn) error {
		rawSchema, err := getBytes(txn, keySchemaVersion)
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		initialized = true
		if len(rawSchema) != 4 {
			return ErrStorageMetadataCorrupt
		}
		info.SchemaVersion = binary.BigEndian.Uint32(rawSchema)
		if info.SchemaVersion != StorageSchemaVersion {
			return fmt.Errorf("%w: got %d want %d", ErrStorageSchemaMismatch, info.SchemaVersion, StorageSchemaVersion)
		}
		if err := s.validateIdentityTxn(txn); err != nil {
			return err
		}
		info.Network, err = getString(txn, keyNetwork)
		if err != nil { return err }
		info.GenesisHash, err = getString(txn, keyGenesis)
		if err != nil { return err }
		info.ActiveTip, err = getString(txn, keyActiveTip)
		if err != nil { return err }
		info.Height, err = getUint64(txn, keyActiveHeight)
		if err != nil { return err }
		if chainwork, chainworkErr := getString(txn, keyActiveChainwork); chainworkErr == nil {
			info.Chainwork = chainwork
		} else if !errors.Is(chainworkErr, badger.ErrKeyNotFound) {
			return chainworkErr
		}
		info.UTXOHash, err = getString(txn, keyUTXOHash)
		if err != nil { return err }
		if status, err := getString(txn, keyMigrationStatus); err == nil {
			info.MigrationStatus = status
		} else if !errors.Is(err, badger.ErrKeyNotFound) {
			return err
		}
		return nil
	})
	return info, initialized, err
}

func (s *BadgerStore) validateIdentityTxn(txn *badger.Txn) error {
	rawSchema, err := getBytes(txn, keySchemaVersion)
	if err != nil {
		return err
	}
	if len(rawSchema) != 4 || binary.BigEndian.Uint32(rawSchema) != StorageSchemaVersion {
		return ErrStorageSchemaMismatch
	}
	network, err := getString(txn, keyNetwork)
	if err != nil {
		return err
	}
	if network != s.network {
		return fmt.Errorf("%w: got %q want %q", ErrStorageNetworkMismatch, network, s.network)
	}
	genesis, err := getString(txn, keyGenesis)
	if err != nil {
		return err
	}
	if genesis != s.genesisHash {
		return fmt.Errorf("%w: got %q want %q", ErrStorageGenesisMismatch, genesis, s.genesisHash)
	}
	return nil
}

func HashUTXOSet(items []utxo.UTXO) string {
	sorted := append([]utxo.UTXO(nil), items...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].TransactionID == sorted[j].TransactionID {
			return sorted[i].OutputIndex < sorted[j].OutputIndex
		}
		return sorted[i].TransactionID < sorted[j].TransactionID
	})
	h := sha256.New()
	var scratch [8]byte
	for _, item := range sorted {
		rawID, err := hex.DecodeString(item.TransactionID)
		if err != nil || len(rawID) != sha256.Size {
			rawID = []byte(item.TransactionID)
		}
		binary.BigEndian.PutUint64(scratch[:], uint64(len(rawID)))
		_, _ = h.Write(scratch[:])
		_, _ = h.Write(rawID)
		var idx [4]byte
		binary.BigEndian.PutUint32(idx[:], item.OutputIndex)
		_, _ = h.Write(idx[:])
		binary.BigEndian.PutUint64(scratch[:], item.Amount)
		_, _ = h.Write(scratch[:])
		binary.BigEndian.PutUint64(scratch[:], uint64(len(item.Recipient)))
		_, _ = h.Write(scratch[:])
		_, _ = h.Write([]byte(item.Recipient))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func diffUTXO(before, after []utxo.UTXO) ([]utxo.UTXO, []Outpoint) {
	beforeMap := utxoMap(before)
	afterMap := utxoMap(after)
	spent := make([]utxo.UTXO, 0)
	for key, item := range beforeMap {
		if _, exists := afterMap[key]; !exists {
			spent = append(spent, item)
		}
	}
	sort.Slice(spent, func(i, j int) bool {
		if spent[i].TransactionID == spent[j].TransactionID {
			return spent[i].OutputIndex < spent[j].OutputIndex
		}
		return spent[i].TransactionID < spent[j].TransactionID
	})
	created := make([]Outpoint, 0)
	for key, item := range afterMap {
		if _, exists := beforeMap[key]; !exists {
			created = append(created, Outpoint{TransactionID: item.TransactionID, OutputIndex: item.OutputIndex})
		}
	}
	sort.Slice(created, func(i, j int) bool {
		if created[i].TransactionID == created[j].TransactionID {
			return created[i].OutputIndex < created[j].OutputIndex
		}
		return created[i].TransactionID < created[j].TransactionID
	})
	return spent, created
}

func utxoMap(items []utxo.UTXO) map[string]utxo.UTXO {
	result := make(map[string]utxo.UTXO, len(items))
	for _, item := range items {
		result[outpointKeyString(item.TransactionID, item.OutputIndex)] = item
	}
	return result
}

func blockKey(hash string) []byte  { return []byte("block/" + hash) }
func headerKey(hash string) []byte { return []byte("header/" + hash) }
func txKey(txid string) []byte     { return []byte("tx/" + txid) }
func undoKey(hash string) []byte   { return []byte("undo/" + hash) }
func heightKey(height uint64) []byte { return []byte(fmt.Sprintf("height/%020d", height)) }
func utxoKey(txid string, index uint32) []byte {
	return []byte("utxo/" + outpointKeyString(txid, index))
}
func outpointKeyString(txid string, index uint32) string {
	return fmt.Sprintf("%s:%010d", txid, index)
}

func encodeUint32(value uint32) []byte {
	var raw [4]byte
	binary.BigEndian.PutUint32(raw[:], value)
	return raw[:]
}
func encodeUint64(value uint64) []byte {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], value)
	return raw[:]
}
func getUint64(txn *badger.Txn, key []byte) (uint64, error) {
	raw, err := getBytes(txn, key)
	if err != nil { return 0, err }
	if len(raw) != 8 { return 0, ErrStorageMetadataCorrupt }
	return binary.BigEndian.Uint64(raw), nil
}
func getString(txn *badger.Txn, key []byte) (string, error) {
	raw, err := getBytes(txn, key)
	if err != nil { return "", err }
	return string(raw), nil
}
func getBytes(txn *badger.Txn, key []byte) ([]byte, error) {
	item, err := txn.Get(key)
	if err != nil { return nil, err }
	return item.ValueCopy(nil)
}
func strictJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}
