package explorer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
	blocks   map[uint64]rpc.BlockResult
	byHash   map[string]rpc.BlockResult
	txs      map[string]rpc.TransactionResult
	balances map[string]rpc.BalanceResult
}

func (m *mockClient) Call(_ context.Context, method string, params any, result any) error {
	switch method {
	case rpc.MethodGetStatus:
		return assign(result, m.status)
	case rpc.MethodGetBlock:
		p := params.(rpc.HeightParams)
		value, ok := m.blocks[p.Height]
		if !ok {
			return errors.New("not found")
		}
		return assign(result, value)
	case rpc.MethodGetBlockByHash:
		p := params.(rpc.HashParams)
		value, ok := m.byHash[p.Hash]
		if !ok {
			return errors.New("not found")
		}
		return assign(result, value)
	case rpc.MethodGetTransaction:
		p := params.(rpc.TransactionParams)
		value, ok := m.txs[p.TransactionID]
		if !ok {
			return errors.New("not found")
		}
		return assign(result, value)
	case rpc.MethodGetBalance:
		p := params.(rpc.AddressParams)
		value, ok := m.balances[p.Address]
		if !ok {
			return errors.New("not found")
		}
		return assign(result, value)
	default:
		return errors.New("unsupported method")
	}
}

func assign(target, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func TestOverviewAndBlockPages(t *testing.T) {
	const blockHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const txid = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	candidate := block.Block{
		Height:       1,
		BlockHash:    blockHash,
		Timestamp:    config.GenesisTimestamp + 60,
		Difficulty:   1,
		Nonce:        42,
		MerkleRoot:   "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		Transactions: []*transaction.Transaction{{TransactionID: txid}},
	}
	client := &mockClient{
		status:   rpc.StatusResult{ChainID: config.ChainID, Height: 1, TipHash: blockHash},
		blocks:   map[uint64]rpc.BlockResult{0: *block.NewGenesis(), 1: candidate},
		byHash:   map[string]rpc.BlockResult{blockHash: candidate},
		txs:      map[string]rpc.TransactionResult{},
		balances: map[string]rpc.BalanceResult{},
	}
	server, err := New(client)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("overview status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), blockHash) || !strings.Contains(recorder.Body.String(), "Network") {
		t.Fatalf("overview body=%s", recorder.Body.String())
	}
	if recorder.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("missing Content-Security-Policy")
	}

	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/block/1", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("block status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), txid) || !strings.Contains(recorder.Body.String(), "Block 1") {
		t.Fatalf("block body=%s", recorder.Body.String())
	}
}

func TestTransactionAddressSearchAndHealth(t *testing.T) {
	const block1Hash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const txid = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	const block2Hash = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"

	owner, err := wallet.New("explorer-owner")
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := wallet.New("explorer-recipient")
	if err != nil {
		t.Fatal(err)
	}
	coinbase, err := transaction.NewCoinbase(
		1,
		owner.Address,
		config.InitialMiningReward,
		config.GenesisTimestamp+60,
	)
	if err != nil {
		t.Fatal(err)
	}
	payment := &transaction.Transaction{
		Version: transaction.Version,
		Inputs: []transaction.Input{{
			PreviousTransactionID: coinbase.TransactionID,
			OutputIndex:           0,
		}},
		Outputs: []transaction.Output{
			{Amount: 10 * config.AtomicUnitsPerVDR, Recipient: recipient.Address},
			{Amount: 40 * config.AtomicUnitsPerVDR, Recipient: owner.Address},
		},
		Timestamp:     config.GenesisTimestamp + 90,
		TransactionID: txid,
	}
	block1 := block.Block{
		Height:            1,
		PreviousBlockHash: config.GenesisBlockHash,
		BlockHash:         block1Hash,
		Timestamp:         config.GenesisTimestamp + 60,
		Transactions:      []*transaction.Transaction{coinbase},
	}
	block2 := block.Block{
		Height:            2,
		PreviousBlockHash: block1Hash,
		BlockHash:         block2Hash,
		Timestamp:         config.GenesisTimestamp + 120,
		Transactions:      []*transaction.Transaction{payment},
	}
	client := &mockClient{
		status: rpc.StatusResult{ChainID: config.ChainID, Height: 2, TipHash: block2Hash},
		blocks: map[uint64]rpc.BlockResult{
			0: *block.NewGenesis(),
			1: block1,
			2: block2,
		},
		byHash: map[string]rpc.BlockResult{
			block1Hash: block1,
			block2Hash: block2,
		},
		txs: map[string]rpc.TransactionResult{
			txid: {Transaction: payment, Confirmed: true, BlockHeight: 2, BlockHash: block2Hash},
		},
		balances: map[string]rpc.BalanceResult{
			owner.Address:     {Address: owner.Address, BalanceVal: 40 * config.AtomicUnitsPerVDR},
			recipient.Address: {Address: recipient.Address, BalanceVal: 10 * config.AtomicUnitsPerVDR},
		},
	}
	server, err := New(client)
	if err != nil {
		t.Fatal(err)
	}
	handler := server.Handler()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/tx/"+txid, nil))
	if recorder.Code != http.StatusOK ||
		!strings.Contains(recorder.Body.String(), "10 VDR") ||
		!strings.Contains(recorder.Body.String(), "40 VDR") {
		t.Fatalf("tx status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/address/"+owner.Address, nil))
	if recorder.Code != http.StatusOK ||
		!strings.Contains(recorder.Body.String(), "Confirmed activity") ||
		!strings.Contains(recorder.Body.String(), "50 VDR") ||
		!strings.Contains(recorder.Body.String(), "40 VDR") {
		t.Fatalf("owner address status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/address/"+recipient.Address, nil))
	if recorder.Code != http.StatusOK ||
		!strings.Contains(recorder.Body.String(), "10 VDR") ||
		!strings.Contains(recorder.Body.String(), txid) {
		t.Fatalf("recipient address status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/search?q=2", nil))
	if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != "/block/2" {
		t.Fatalf("height search status=%d location=%s", recorder.Code, recorder.Header().Get("Location"))
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/search?q="+block2Hash, nil))
	if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != "/block/"+block2Hash {
		t.Fatalf("hash search status=%d location=%s", recorder.Code, recorder.Header().Get("Location"))
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "\"status\":\"ok\"") {
		t.Fatalf("health status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
