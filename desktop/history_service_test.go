package desktop

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/core/utxo"
	"github.com/Sheff1981/valdr-core/rpc"
	"github.com/Sheff1981/valdr-core/wallet"
)

type historyRPCMock struct {
	status  rpc.StatusResult
	blocks  map[uint64]rpc.BlockResult
	mempool []*transaction.Transaction
	utxos   []utxo.UTXO
}

func (m *historyRPCMock) Call(
	_ context.Context,
	method string,
	params any,
	result any,
) error {
	switch method {
	case rpc.MethodGetStatus:
		return assignHistoryRPC(result, m.status)
	case rpc.MethodGetBlock:
		p := params.(rpc.HeightParams)
		return assignHistoryRPC(result, m.blocks[p.Height])
	case rpc.MethodGetMempool:
		return assignHistoryRPC(result, m.mempool)
	case rpc.MethodGetUTXOs:
		return assignHistoryRPC(result, m.utxos)
	default:
		panic("unexpected history RPC method: " + method)
	}
}

func assignHistoryRPC(target, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func TestHistoryServicePendingThenConfirmed(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	genesis, err := block.NewGenesisForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	source, err := wallet.New("history-source")
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := wallet.New("history-recipient")
	if err != nil {
		t.Fatal(err)
	}

	coinbase, err := transaction.NewCoinbaseForChain(
		profile.ChainID,
		1,
		source.Address,
		profile.InitialSubsidyVDR*config.AtomicUnitsPerVDR,
		profile.GenesisTimestamp+60,
	)
	if err != nil {
		t.Fatal(err)
	}
	h1 := strings.Repeat("a", 64)
	block1 := rpc.BlockResult{
		Version:           2,
		Height:            1,
		PreviousBlockHash: genesis.BlockHash,
		BlockHash:         h1,
		ChainID:           profile.ChainID,
		Timestamp:         profile.GenesisTimestamp + 60,
		Transactions:      []*transaction.Transaction{coinbase},
	}

	available := []utxo.UTXO{{
		TransactionID: coinbase.TransactionID,
		OutputIndex:   0,
		Amount:        profile.InitialSubsidyVDR * config.AtomicUnitsPerVDR,
		Recipient:     source.Address,
	}}
	spend, fee, err := source.CreateTransactionForChain(
		profile.ChainID,
		available,
		recipient.Address,
		config.AtomicUnitsPerVDR/2,
		profile.MinRelayFeePerByte,
		profile.GenesisTimestamp+120,
	)
	if err != nil {
		t.Fatal(err)
	}
	if fee == 0 {
		t.Fatal("expected non-zero Testnet fee")
	}

	mock := &historyRPCMock{
		status: rpc.StatusResult{
			Network:   profile.Name,
			ChainID:   profile.ChainID,
			Height:    1,
			TipHash:   h1,
			Chainwork: "01",
		},
		blocks: map[uint64]rpc.BlockResult{
			0: *genesis,
			1: block1,
		},
		mempool: []*transaction.Transaction{spend},
		utxos:   available,
	}
	service, err := NewHistoryService(
		mock,
		filepath.Join(t.TempDir(), "desktop-history.json"),
	)
	if err != nil {
		t.Fatal(err)
	}

	history, err := service.History(
		context.Background(),
		source.Address,
		100,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 {
		t.Fatalf("history=%+v want pending spend + confirmed receive", history)
	}
	if history[0].Status != "pending" ||
		history[0].Direction != "sent" ||
		history[0].TransactionID != spend.TransactionID ||
		history[0].AmountVal != config.AtomicUnitsPerVDR/2 ||
		history[0].FeeVal != fee {
		t.Fatalf("unexpected pending item: %+v", history[0])
	}
	if history[1].Status != "confirmed" ||
		history[1].Direction != "received" ||
		history[1].TransactionID != coinbase.TransactionID ||
		history[1].Confirmations != 1 {
		t.Fatalf("unexpected confirmed receive: %+v", history[1])
	}

	h2 := strings.Repeat("b", 64)
	mock.blocks[2] = rpc.BlockResult{
		Version:           2,
		Height:            2,
		PreviousBlockHash: h1,
		BlockHash:         h2,
		ChainID:           profile.ChainID,
		Timestamp:         profile.GenesisTimestamp + 120,
		Transactions:      []*transaction.Transaction{spend},
	}
	mock.status.Height = 2
	mock.status.TipHash = h2
	mock.mempool = nil
	mock.utxos = nil

	history, err = service.History(
		context.Background(),
		source.Address,
		100,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 {
		t.Fatalf("confirmed history=%+v", history)
	}
	if history[0].Status != "confirmed" ||
		history[0].Direction != "sent" ||
		history[0].TransactionID != spend.TransactionID ||
		history[0].FeeVal != fee ||
		history[0].Confirmations != 1 {
		t.Fatalf("unexpected confirmed spend: %+v", history[0])
	}
	if history[1].Confirmations != 2 {
		t.Fatalf("coinbase confirmations=%d want=2", history[1].Confirmations)
	}
}
