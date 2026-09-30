package desktop

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

const historyJournalVersion = 1

type historyJournalState struct {
	Version int                                                   `json:"version"`
	Items   map[string]map[string]TransactionHistoryItem          `json:"items"`
}

type historyJournal struct {
	mu    sync.Mutex
	path  string
	state historyJournalState
}

func newHistoryJournal(path string) (*historyJournal, error) {
	j := &historyJournal{
		path: path,
		state: historyJournalState{
			Version: historyJournalVersion,
			Items:   make(map[string]map[string]TransactionHistoryItem),
		},
	}
	if path == "" {
		return j, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return j, nil
	}
	if err != nil {
		return nil, err
	}
	var loaded historyJournalState
	if err := json.Unmarshal(raw, &loaded); err != nil {
		// The journal is derived wallet metadata, not consensus or key
		// material. A corrupt journal must not prevent Desktop startup.
		return j, nil
	}
	if loaded.Version != historyJournalVersion || loaded.Items == nil {
		return j, nil
	}
	j.state = loaded
	return j, nil
}

// Merge preserves wallet-visible knowledge when a transaction leaves the
// active mempool/chain. A formerly pending transaction that disappears becomes
// conflicted; a formerly confirmed transaction removed by a reorg becomes
// reorged. If it later reappears, the live node state wins.
func (j *historyJournal) Merge(
	address string,
	current []TransactionHistoryItem,
) ([]TransactionHistoryItem, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	previous := j.state.Items[address]
	if previous == nil {
		previous = make(map[string]TransactionHistoryItem)
	}
	live := make(map[string]struct{}, len(current))
	next := make(map[string]TransactionHistoryItem, len(previous)+len(current))
	result := append([]TransactionHistoryItem(nil), current...)

	for _, item := range current {
		if item.TransactionID == "" {
			continue
		}
		live[item.TransactionID] = struct{}{}
		next[item.TransactionID] = item
	}

	for txid, item := range previous {
		if _, ok := live[txid]; ok {
			continue
		}
		switch item.Status {
		case "pending":
			item.Status = "conflicted"
			item.Confirmations = 0
		case "confirmed":
			item.Status = "reorged"
			item.Confirmations = 0
			item.BlockHeight = 0
			item.BlockHash = ""
		case "conflicted", "reorged":
			// Preserve terminal wallet-visible state until live chain/mempool
			// knowledge for the same txid supersedes it.
		default:
			continue
		}
		next[txid] = item
		result = append(result, item)
	}

	j.state.Items[address] = next
	return result, j.saveLocked()
}

func (j *historyJournal) saveLocked() error {
	if j.path == "" {
		return nil
	}
	dir := filepath.Dir(j.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(j.state, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp, err := os.CreateTemp(dir, ".wallet-history-*.tmp")
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
	if err := os.Rename(tmpName, j.path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Chmod(j.path, 0o600)
}
