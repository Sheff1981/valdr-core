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

	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/core/utxo"
	"github.com/Sheff1981/valdr-core/rpc"
)

const explorerIndexVersion = 1

var (
	ErrInvalidClient       = errors.New("explorer RPC client is nil")
	ErrIndexChainMismatch  = errors.New("explorer index chain identity mismatch")
	ErrIndexLinkMismatch   = errors.New("explorer index block linkage mismatch")
	ErrIndexMissingOutpoint = errors.New("explorer index missing spent outpoint")
)

type RPCClient interface {
	Call(context.Context, string, any, any) error
}

type AddressActivity struct {
	TransactionID string `json:"transaction_id"`
	BlockHeight   uint64 `json:"block_height"`
	BlockHash     string `json:"block_hash"`
	Timestamp     int64  `json:"timestamp"`
	ReceivedVal   uint64 `json:"received_val"`
	SpentVal      uint64 `json:"spent_val"`
}

type indexedOutput struct {
	TransactionID string `json:"transaction_id"`
	OutputIndex   uint32 `json:"output_index"`
	Address       string `json:"address"`
	AmountVal     uint64 `json:"amount_val"`
}

type indexState struct {
	Version    int                          `json:"version"`
	ChainID    string                       `json:"chain_id"`
	GenesisHash string                      `json:"genesis_hash"`
	Height     uint64                       `json:"height"`
	TipHash    string                       `json:"tip_hash"`
	Blocks     map[uint64]string            `json:"blocks"`
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
	if path != "" {
		if err := idx.load(); err != nil {
			return nil, err
		}
	}
	return idx, nil
}

func (i *Index) Refresh(ctx context.Context) error {
	var status rpc.StatusResult
	if err := i.client.Call(ctx, rpc.MethodGetStatus, nil, &status); err != nil {
		return err
	}
	var genesis rpc.BlockResult
	if err := i.client.Call(
		ctx,
		rpc.MethodGetBlock,
		rpc.HeightParams{Height: 0},
		&genesis,
	); err != nil {
		return err
	}
	if status.ChainID == "" || genesis.BlockHash == "" ||
		genesis.ChainID != status.ChainID {
		return ErrIndexChainMismatch
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	if i.state.Version == 0 {
		i.resetLocked(status.ChainID, genesis.BlockHash)
	} else if i.state.Version != explorerIndexVersion ||
		i.state.ChainID != status.ChainID ||
		i.state.GenesisHash != genesis.BlockHash {
		return fmt.Errorf(
			"%w: index=%s/%s node=%s/%s",
			ErrIndexChainMismatch,
			i.state.ChainID,
			i.state.GenesisHash,
			status.ChainID,
			genesis.BlockHash,
		)
	}

	ancestor, err := i.commonAncestorLocked(ctx, status.Height)
	if err != nil {
		return err
	}
	if ancestor < i.state.Height {
		if err := i.rebuildLocked(ctx, ancestor); err != nil {
			return err
		}
	}
	for height := i.state.Height + 1; height <= status.Height; height++ {
		var candidate rpc.BlockResult
		if err := i.client.Call(
			ctx,
			rpc.MethodGetBlock,
			rpc.HeightParams{Height: height},
			&candidate,
		); err != nil {
			return err
		}
		if err := i.applyBlockLocked(&candidate); err != nil {
			return err
		}
	}
	if i.state.Height != status.Height || i.state.TipHash != status.TipHash {
		return fmt.Errorf(
			"%w: indexed=%d/%s node=%d/%s",
			ErrIndexLinkMismatch,
			i.state.Height,
			i.state.TipHash,
			status.Height,
			status.TipHash,
		)
	}
	return i.saveLocked()
}

func (i *Index) Activities(address string) []AddressActivity {
	i.mu.RLock()
	defer i.mu.RUnlock()
	result := append([]AddressActivity(nil), i.state.Activities[address]...)
	sort.SliceStable(result, func(a, b int) bool {
		if result[a].BlockHeight == result[b].BlockHeight {
			return result[a].TransactionID > result[b].TransactionID
		}
		return result[a].BlockHeight > result[b].BlockHeight
	})
	return result
}

func (i *Index) UTXOs(address string) []utxo.UTXO {
	i.mu.RLock()
	defer i.mu.RUnlock()
	result := make([]utxo.UTXO, 0)
	for _, item := range i.state.Outpoints {
		if item.Address != address {
			continue
		}
		result = append(result, utxo.UTXO{
			TransactionID: item.TransactionID,
			OutputIndex:   item.OutputIndex,
			Amount:        item.AmountVal,
			Recipient:     item.Address,
		})
	}
	sort.Slice(result, func(a, b int) bool {
		if result[a].TransactionID == result[b].TransactionID {
			return result[a].OutputIndex < result[b].OutputIndex
		}
		return result[a].TransactionID < result[b].TransactionID
	})
	return result
}

func (i *Index) State() (height uint64, tipHash string) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.state.Height, i.state.TipHash
}

func (i *Index) commonAncestorLocked(
	ctx context.Context,
	nodeHeight uint64,
) (uint64, error) {
	if len(i.state.Blocks) == 0 {
		return 0, nil
	}
	height := i.state.Height
	if nodeHeight < height {
		height = nodeHeight
	}
	for {
		var candidate rpc.BlockResult
		if err := i.client.Call(
			ctx,
			rpc.MethodGetBlock,
			rpc.HeightParams{Height: height},
			&candidate,
		); err != nil {
			return 0, err
		}
		if expected, ok := i.state.Blocks[height]; ok &&
			expected == candidate.BlockHash {
			return height, nil
		}
		if height == 0 {
			return 0, ErrIndexChainMismatch
		}
		height--
	}
}

func (i *Index) rebuildLocked(
	ctx context.Context,
	through uint64,
) error {
	chainID := i.state.ChainID
	genesisHash := i.state.GenesisHash
	i.resetLocked(chainID, genesisHash)
	for height := uint64(1); height <= through; height++ {
		var candidate rpc.BlockResult
		if err := i.client.Call(
			ctx,
			rpc.MethodGetBlock,
			rpc.HeightParams{Height: height},
			&candidate,
		); err != nil {
			return err
		}
		if err := i.applyBlockLocked(&candidate); err != nil {
			return err
		}
	}
	return nil
}

func (i *Index) applyBlockLocked(candidate *rpc.BlockResult) error {
	if candidate == nil {
		return errors.New("nil explorer block")
	}
	expectedHeight := i.state.Height + 1
	if candidate.Height != expectedHeight ||
		candidate.PreviousBlockHash != i.state.TipHash {
		return fmt.Errorf(
			"%w: height=%d previous=%s expected_height=%d expected_previous=%s",
			ErrIndexLinkMismatch,
			candidate.Height,
			candidate.PreviousBlockHash,
			expectedHeight,
			i.state.TipHash,
		)
	}

	for _, tx := range candidate.Transactions {
		if tx == nil {
			continue
		}
		if err := i.applyTransactionLocked(candidate, tx); err != nil {
			return err
		}
	}
	i.state.Height = candidate.Height
	i.state.TipHash = candidate.BlockHash
	i.state.Blocks[candidate.Height] = candidate.BlockHash
	return nil
}

func (i *Index) applyTransactionLocked(
	candidate *rpc.BlockResult,
	tx *transaction.Transaction,
) error {
	activity := make(map[string]*AddressActivity)
	entryFor := func(address string) *AddressActivity {
		entry := activity[address]
		if entry == nil {
			entry = &AddressActivity{
				TransactionID: tx.TransactionID,
				BlockHeight:   candidate.Height,
				BlockHash:     candidate.BlockHash,
				Timestamp:     tx.Timestamp,
			}
			if entry.Timestamp == 0 {
				entry.Timestamp = candidate.Timestamp
			}
			activity[address] = entry
		}
		return entry
	}

	if !tx.IsCoinbase() {
		for _, input := range tx.Inputs {
			key := outpointKey(input.PreviousTransactionID, input.OutputIndex)
			previous, ok := i.state.Outpoints[key]
			if !ok {
				return fmt.Errorf("%w: %s", ErrIndexMissingOutpoint, key)
			}
			entryFor(previous.Address).SpentVal += previous.AmountVal
			delete(i.state.Outpoints, key)
		}
	}
	for outputIndex, output := range tx.Outputs {
		index := uint32(outputIndex)
		entryFor(output.Recipient).ReceivedVal += output.Amount
		i.state.Outpoints[outpointKey(tx.TransactionID, index)] = indexedOutput{
			TransactionID: tx.TransactionID,
			OutputIndex:   index,
			Address:       output.Recipient,
			AmountVal:     output.Amount,
		}
	}
	for address, item := range activity {
		i.state.Activities[address] = append(i.state.Activities[address], *item)
	}
	return nil
}

func outpointKey(transactionID string, outputIndex uint32) string {
	return fmt.Sprintf("%s:%d", transactionID, outputIndex)
}

func (i *Index) resetLocked(chainID, genesisHash string) {
	i.state = indexState{
		Version:     explorerIndexVersion,
		ChainID:     chainID,
		GenesisHash: genesisHash,
		Height:      0,
		TipHash:     genesisHash,
		Blocks:      map[uint64]string{0: genesisHash},
		Activities:  make(map[string][]AddressActivity),
		Outpoints:   make(map[string]indexedOutput),
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
	if loaded.Version != explorerIndexVersion ||
		loaded.ChainID == "" ||
		loaded.GenesisHash == "" ||
		loaded.Blocks == nil ||
		loaded.Activities == nil ||
		loaded.Outpoints == nil {
		return ErrIndexChainMismatch
	}
	i.state = loaded
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
