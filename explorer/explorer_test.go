package explorer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/rpc"
	"github.com/Sheff1981/valdr-core/wallet"
)

type mockClient struct {
	status   rpc.StatusResult
	mining   rpc.MiningInfoResult
	peers    []rpc.PeerResult
	blocks   map[uint64]rpc.BlockResult
	byHash   map[string]rpc.BlockResult
	txs      map[string]rpc.TransactionResult
	balances map[string]rpc.BalanceResult
	mempool  []*transaction.Transaction
}

func (m *mockClient) Call(_ context.Context, method string, params any, result any) error {
	switch method {
	case rpc.MethodGetStatus:
		return assign(result, m.status)
	case rpc.MethodGetMiningInfo:
		return assign(result, m.mining)
	case rpc.MethodGetPeers:
		return assign(result, m.peers)
	case rpc.MethodGetMempool:
		return assign(result, m.mempool)
	case rpc.MethodGetBlock:
		p := params.(rpc.HeightParams)
		value, ok := m.blocks[p.Height]
		if !ok { return errors.New("block not found") }
		return assign(result, value)
	case rpc.MethodGetBlockByHash:
		p := params.(rpc.HashParams)
		value, ok := m.byHash[p.Hash]
		if !ok { return errors.New("block not found") }
		return assign(result, value)
	case rpc.MethodGetTransaction:
		p := params.(rpc.TransactionParams)
		value, ok := m.txs[p.TransactionID]
		if !ok { return errors.New("transaction not found") }
		return assign(result, value)
	case rpc.MethodGetBalance:
		p := params.(rpc.AddressParams)
		value, ok := m.balances[p.Address]
		if !ok { value = rpc.BalanceResult{Address:p.Address} }
		return assign(result, value)
	default:
		return errors.New("unsupported read method")
	}
}

func assign(target, value any) error {
	raw, err := json.Marshal(value)
	if err != nil { return err }
	return json.Unmarshal(raw, target)
}

func explorerFixture(t *testing.T) (*mockClient, string, string, string) {
	t.Helper()
	owner, err := wallet.New("owner")
	if err != nil { t.Fatal(err) }
	recipient, err := wallet.New("recipient")
	if err != nil { t.Fatal(err) }

	genesis := *block.NewGenesis()
	coinbase, err := transaction.NewCoinbase(
		1, owner.Address, config.InitialMiningReward, config.GenesisTimestamp+60,
	)
	if err != nil { t.Fatal(err) }
	h1 := strings.Repeat("a", 64)
	h2a := strings.Repeat("b", 64)
	h2b := strings.Repeat("c", 64)
	txid := strings.Repeat("d", 64)

	block1 := block.Block{
		Version:1, Height:1, PreviousBlockHash:genesis.BlockHash,
		BlockHash:h1, ChainID:config.ChainID,
		Timestamp:config.GenesisTimestamp+60,
		Transactions:[]*transaction.Transaction{coinbase},
	}
	payment := &transaction.Transaction{
		Version: transaction.Version,
		Inputs: []transaction.Input{{
			PreviousTransactionID: coinbase.TransactionID, OutputIndex:0,
		}},
		Outputs: []transaction.Output{{
			Amount:config.InitialMiningReward, Recipient:recipient.Address,
		}},
		Timestamp:config.GenesisTimestamp+120, TransactionID:txid,
	}
	block2A := block.Block{
		Version:1, Height:2, PreviousBlockHash:h1, BlockHash:h2a,
		ChainID:config.ChainID, Timestamp:config.GenesisTimestamp+120,
		Transactions:[]*transaction.Transaction{payment},
	}
	reorgCoinbase, err := transaction.NewCoinbase(
		2, owner.Address, config.InitialMiningReward, config.GenesisTimestamp+121,
	)
	if err != nil { t.Fatal(err) }
	block2B := block.Block{
		Version:1, Height:2, PreviousBlockHash:h1, BlockHash:h2b,
		ChainID:config.ChainID, Timestamp:config.GenesisTimestamp+121,
		Transactions:[]*transaction.Transaction{reorgCoinbase},
	}

	client := &mockClient{
		status: rpc.StatusResult{
			Project:config.ProjectName, Ticker:config.Ticker, Version:config.Version,
			Network:config.NetworkLegacyV01, ChainID:config.ChainID,
			Height:2, TipHash:h2a, Chainwork:"01", BlockVersion:1,
			TargetBlockTimeSeconds:60,
		},
		mining: rpc.MiningInfoResult{
			Height:2, NextHeight:3, CurrentDifficulty:1,
			BlockRewardVal:config.InitialMiningReward, TargetBlockTimeSeconds:60,
		},
		blocks: map[uint64]rpc.BlockResult{0:genesis,1:block1,2:block2A},
		byHash: map[string]rpc.BlockResult{genesis.BlockHash:genesis,h1:block1,h2a:block2A,h2b:block2B},
		txs: map[string]rpc.TransactionResult{
			txid:{Transaction:payment,Confirmed:true,BlockHeight:2,BlockHash:h2a},
		},
		balances: map[string]rpc.BalanceResult{
			recipient.Address:{Address:recipient.Address,BalanceVal:config.InitialMiningReward},
		},
		mempool: []*transaction.Transaction{},
	}
	return client, recipient.Address, h2a, h2b
}

func TestExplorerIndexPersistsAndReorgs(t *testing.T) {
	client, recipient, _, h2b := explorerFixture(t)
	path := filepath.Join(t.TempDir(), "index.json")
	idx, err := NewIndex(client, path)
	if err != nil { t.Fatal(err) }
	if err := idx.Refresh(context.Background()); err != nil { t.Fatal(err) }
	if len(idx.Activities(recipient)) != 1 || len(idx.UTXOs(recipient)) != 1 {
		t.Fatalf("initial recipient index history=%v utxos=%v", idx.Activities(recipient), idx.UTXOs(recipient))
	}

	restarted, err := NewIndex(client, path)
	if err != nil { t.Fatal(err) }
	if height, _ := restarted.State(); height != 2 {
		t.Fatalf("persisted height=%d want=2", height)
	}

	client.blocks[2] = client.byHash[h2b]
	client.status.TipHash = h2b
	client.balances[recipient] = rpc.BalanceResult{Address:recipient}
	if err := restarted.Refresh(context.Background()); err != nil { t.Fatal(err) }
	if history := restarted.Activities(recipient); len(history) != 0 {
		t.Fatalf("reorg left stale address history: %+v", history)
	}
	if items := restarted.UTXOs(recipient); len(items) != 0 {
		t.Fatalf("reorg left stale UTXOs: %+v", items)
	}
	if height, tip := restarted.State(); height != 2 || tip != h2b {
		t.Fatalf("reorg state=%d/%s want=2/%s", height, tip, h2b)
	}
}

func TestExplorerRESTAndViews(t *testing.T) {
	client, recipient, h2a, _ := explorerFixture(t)
	server, err := New(client)
	if err != nil { t.Fatal(err) }
	handler := server.Handler()

	check := func(path string, want int, contains ...string) {
		t.Helper()
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != want {
			t.Fatalf("%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		for _, value := range contains {
			if !strings.Contains(rec.Body.String(), value) {
				t.Fatalf("%s missing %q body=%s", path, value, rec.Body.String())
			}
		}
		if rec.Header().Get("Content-Security-Policy") == "" {
			t.Fatalf("%s missing security headers", path)
		}
	}

	check("/api/v1/status", http.StatusOK, config.ChainID, "average_block_interval_seconds")
	check("/api/v1/blocks?limit=2", http.StatusOK, h2a)
	check("/api/v1/block/2", http.StatusOK, h2a)
	check("/api/v1/tx/"+strings.Repeat("d",64), http.StatusOK, "\"confirmed\":true")
	check("/api/v1/address/"+recipient, http.StatusOK, "\"history\"", "\"utxos\"")
	check("/api/v1/mempool", http.StatusOK, "[]")
	check("/", http.StatusOK, "VALDR Explorer", "chainwork")
	check("/address/"+recipient, http.StatusOK, recipient)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/search?q=2", nil))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/block/2" {
		t.Fatalf("search status=%d location=%s", rec.Code, rec.Header().Get("Location"))
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/status", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("write-like method status=%d want=405", rec.Code)
	}
}
