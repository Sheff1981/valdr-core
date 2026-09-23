package explorer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/rpc"
)

var (
	ErrIndexChainMismatch = errors.New("explorer index chain mismatch")
	ErrIndexLinkMismatch  = errors.New("explorer index block linkage mismatch")
)

type AddressActivity struct {
	TransactionID string `json:"transaction_id"`
	BlockHeight   uint64 `json:"block_height"`
	BlockHash     string `json:"block_hash"`
	Timestamp     int64  `json:"timestamp"`
	ReceivedVal   uint64 `json:"received_val"`
	SpentVal      uint64 `json:"spent_val"`
}

type indexedOutput struct {
	Address   string `json:"address"`
	AmountVal uint64 `json:"amount_val"`
}

type indexState struct {
	ChainID    string                       `json:"chain_id"`
	Height     uint64                       `json:"height"`
	TipHash    string                       `json:"tip_hash"`
	Activities map[string][]AddressActivity `json:"activities"`
	Outpoints  map[string]indexedOutput     `json:"outpoints"`
}

type Index struct {
	mu     sync.RWMutex
	client RPCClient
	path   string
	state  indexState
}

func NewIndex(client RPCClient, path string) (*Index, error) {
	if client == nil {
		return nil, ErrInvalidClient
	}
	idx := &Index{client: client, path: path}
	idx.reset()
	if path == "" {
		return idx, nil
	}
	if err := idx.load(); err != nil {
		return nil, err
	}
	return idx, nil
}

func (i *Index) Refresh(ctx context.Context) error {
	var status rpc.StatusResult
	if err := i.client.Call(ctx, rpc.MethodGetStatus, nil, &status); err != nil {
		return err
	}
	if status.ChainID != config.ChainID {
		return fmt.Errorf("%w: got %q want %q", ErrIndexChainMismatch, status.ChainID, config.ChainID)
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	if i.state.ChainID != config.ChainID {
		i.resetLocked()
	}

	if i.state.Height > status.Height {
		i.resetLocked()
	}
	if i.state.Height > 0 {
		var indexedTip rpc.BlockResult
		if err := i.client.Call(
			ctx,
			rpc.MethodGetBlock,
			rpc.HeightParams{Height: i.state.Height},
			&indexedTip,
		); err != nil {
			return err
		}
		if indexedTip.BlockHash != i.state.TipHash {
			i.resetLocked()
		}
	}

	start := uint64(1)
	if i.state.Height > 0 {
		start = i.state.Height + 1
	}

	for height := start; height <= status.Height; height++ {
		var candidate rpc.BlockResult
		if err := i.client.Call(
			ctx,
			rpc.MethodGetBlock,
			rpc.HeightParams{Height: height},
			&candidate,
		); err != nil {
			return err
		}

		expectedPrevious := config.GenesisBlockHash
		if i.state.Height > 0 {
			expectedPrevious = i.state.TipHash
		}
		if candidate.Height != height || candidate.PreviousBlockHash != expectedPrevious {
			return fmt.Errorf(
				"%w: height=%d previous=%s expected=%s",
				ErrIndexLinkMismatch,
				height,
				candidate.PreviousBlockHash,
				expectedPrevious,
			)
		}

		if err := i.applyBlockLocked(&candidate); err != nil {
			return err
		}
		i.state.Height = candidate.Height
		i.state.TipHash = candidate.BlockHash
	}

	if i.path != "" {
		return i.saveLocked()
	}
	return nil
}

func (i *Index) Activities(address string) []AddressActivity {
	i.mu.RLock()
	defer i.mu.RUnlock()

	source := i.state.Activities[address]
	result := append([]AddressActivity(nil), source...)
	sort.SliceStable(result, func(a, b int) bool {
		if result[a].BlockHeight == result[b].BlockHeight {
			return result[a].TransactionID > result[b].TransactionID
		}
		return result[a].BlockHeight > result[b].BlockHeight
	})
	return result
}

func (i *Index) applyBlockLocked(candidate *rpc.BlockResult) error {
	if candidate == nil {
		return errors.New("nil block")
	}

	for _, tx := range candidate.Transactions {
		if tx == nil {
			continue
		}

		activityByAddress := make(map[string]*AddressActivity)
		getActivity := func(address string) *AddressActivity {
			entry := activityByAddress[address]
			if entry == nil {
				entry = &AddressActivity{
					TransactionID: tx.TransactionID,
					BlockHeight:   candidate.Height,
					BlockHash:     candidate.BlockHash,
					Timestamp:     tx.Timestamp,
				}
				activityByAddress[address] = entry
			}
			return entry
		}

		if !tx.IsCoinbase() {
			for _, input := range tx.Inputs {
				outpoint := outpointKey(input.PreviousTransactionID, input.OutputIndex)
				previous, ok := i.state.Outpoints[outpoint]
				if !ok {
					return fmt.Errorf("explorer index missing outpoint %s", outpoint)
				}
				entry := getActivity(previous.Address)
				entry.SpentVal += previous.AmountVal
				delete(i.state.Outpoints, outpoint)
			}
		}

		for outputIndex, output := range tx.Outputs {
			entry := getActivity(output.Recipient)
			entry.ReceivedVal += output.Amount
			i.state.Outpoints[outpointKey(tx.TransactionID, uint32(outputIndex))] = indexedOutput{
				Address:   output.Recipient,
				AmountVal: output.Amount,
			}
		}

		for address, entry := range activityByAddress {
			i.state.Activities[address] = append(i.state.Activities[address], *entry)
		}
	}
	return nil
}

func outpointKey(transactionID string, outputIndex uint32) string {
	return fmt.Sprintf("%s:%d", transactionID, outputIndex)
}

func (i *Index) reset() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.resetLocked()
}

func (i *Index) resetLocked() {
	i.state = indexState{
		ChainID:    config.ChainID,
		Activities: make(map[string][]AddressActivity),
		Outpoints:  make(map[string]indexedOutput),
	}
}

func (i *Index) load() error {
	raw, err := os.ReadFile(i.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	var loaded indexState
	if err := json.Unmarshal(raw, &loaded); err != nil {
		return err
	}
	if loaded.ChainID != config.ChainID {
		return fmt.Errorf("%w: got %q want %q", ErrIndexChainMismatch, loaded.ChainID, config.ChainID)
	}
	if loaded.Activities == nil {
		loaded.Activities = make(map[string][]AddressActivity)
	}
	if loaded.Outpoints == nil {
		loaded.Outpoints = make(map[string]indexedOutput)
	}

	i.mu.Lock()
	i.state = loaded
	i.mu.Unlock()
	return nil
}

func (i *Index) saveLocked() error {
	if i.path == "" {
		return nil
	}
	dir := filepath.Dir(i.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(i.state, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')

	tmp, err := os.CreateTemp(dir, ".explorer-index-*.tmp")
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
	if err := os.Rename(tmpName, i.path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Chmod(i.path, 0o600)
}

