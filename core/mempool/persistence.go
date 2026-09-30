package mempool

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sheff1981/valdr-core/core/transaction"
)

const diskSnapshotVersion = 1

type diskSnapshot struct {
	Version      int                        `json:"version"`
	ChainID      string                     `json:"chain_id"`
	SavedAt      int64                      `json:"saved_at"`
	Transactions []*transaction.Transaction `json:"transactions"`
}

// SaveFile persists a bounded mempool snapshot atomically. Only public
// transaction data is written; wallet secrets never enter the node mempool.
func SaveFile(path, chainID string, transactions []*transaction.Transaction) error {
	path = strings.TrimSpace(path)
	chainID = strings.TrimSpace(chainID)
	if path == "" || chainID == "" {
		return errors.New("invalid mempool persistence path or chain id")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	snapshot := diskSnapshot{
		Version:      diskSnapshotVersion,
		ChainID:      chainID,
		SavedAt:      time.Now().UTC().Unix(),
		Transactions: transactions,
	}
	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(path), ".mempool-v2-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}
	if err := tmp.Chmod(0o600); err != nil {
		cleanup()
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Chmod(path, 0o600)
}

// LoadFile loads a persisted snapshot. Callers must revalidate every returned
// transaction against the current active-chain UTXO set before admitting it.
func LoadFile(path, chainID string) ([]*transaction.Transaction, error) {
	path = strings.TrimSpace(path)
	chainID = strings.TrimSpace(chainID)
	if path == "" || chainID == "" {
		return nil, errors.New("invalid mempool persistence path or chain id")
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var snapshot diskSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, err
	}
	if snapshot.Version != diskSnapshotVersion {
		return nil, errors.New("unsupported mempool snapshot version")
	}
	if snapshot.ChainID != chainID {
		return nil, errors.New("mempool snapshot chain id mismatch")
	}

	result := make([]*transaction.Transaction, 0, len(snapshot.Transactions))
	for _, tx := range snapshot.Transactions {
		if tx == nil || tx.IsCoinbase() {
			continue
		}
		result = append(result, cloneTransaction(tx))
	}
	return result, nil
}
