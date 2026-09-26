package rpc

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/mempool"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/core/utxo"
	"github.com/Sheff1981/valdr-core/mining"
	"github.com/Sheff1981/valdr-core/p2p"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestRPCReadSendConfirmFlow(t *testing.T) {
	chain := blockchain.New()
	pool := mempool.New()

	minerWallet, err := wallet.New("miner")
	if err != nil {
		t.Fatal(err)
	}
	recipientWallet, err := wallet.New("recipient")
	if err != nil {
		t.Fatal(err)
	}

	block1, err := mining.MineBlock(
		chain,
		minerWallet.Address,
		config.GenesisTimestamp+60,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	node, err := p2p.NewNode(p2p.NodeConfig{
		NodeID:        "rpc-test-node",
		ListenAddress: "127.0.0.1:0",
		Blockchain:    chain,
		Mempool:       pool,
	})
	if err != nil {
		t.Fatal(err)
	}

	server, err := NewServer(chain, node)
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	client := NewClient(httpServer.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var status StatusResult
	if err := client.Call(ctx, MethodGetStatus, nil, &status); err != nil {
		t.Fatal(err)
	}
	if status.Height != 1 || status.TipHash != block1.BlockHash {
		t.Fatalf("status = %+v", status)
	}
	if status.ChainID != config.ChainID ||
		status.MempoolCount != 0 ||
		status.MempoolSizeBytes != 0 ||
		status.ProtocolVersion == 0 {
		t.Fatalf("unexpected status = %+v", status)
	}

	var byHeight block.Block
	if err := client.Call(
		ctx,
		MethodGetBlock,
		HeightParams{Height: 1},
		&byHeight,
	); err != nil {
		t.Fatal(err)
	}
	if byHeight.BlockHash != block1.BlockHash {
		t.Fatalf("getBlock hash = %s, want %s", byHeight.BlockHash, block1.BlockHash)
	}

	var byHash block.Block
	if err := client.Call(
		ctx,
		MethodGetBlockByHash,
		HashParams{Hash: block1.BlockHash},
		&byHash,
	); err != nil {
		t.Fatal(err)
	}
	if byHash.Height != 1 {
		t.Fatalf("getBlockByHash height = %d, want 1", byHash.Height)
	}

	var miningInfo MiningInfoResult
	if err := client.Call(ctx, MethodGetMiningInfo, nil, &miningInfo); err != nil {
		t.Fatal(err)
	}
	if miningInfo.NextHeight != 2 ||
		miningInfo.BlockRewardVal != config.InitialMiningReward ||
		miningInfo.TargetBlockTimeSeconds != config.TargetBlockTimeSeconds {
		t.Fatalf("mining info = %+v", miningInfo)
	}

	var sourceBalance BalanceResult
	if err := client.Call(
		ctx,
		MethodGetBalance,
		AddressParams{Address: minerWallet.Address},
		&sourceBalance,
	); err != nil {
		t.Fatal(err)
	}
	if sourceBalance.BalanceVal != config.InitialMiningReward {
		t.Fatalf("miner balance = %d, want %d", sourceBalance.BalanceVal, config.InitialMiningReward)
	}

	var available []utxo.UTXO
	if err := client.Call(
		ctx,
		MethodGetUTXOs,
		AddressParams{Address: minerWallet.Address},
		&available,
	); err != nil {
		t.Fatal(err)
	}
	if len(available) != 1 {
		t.Fatalf("UTXO count = %d, want 1", len(available))
	}

	payment, err := minerWallet.CreateTransaction(
		available,
		recipientWallet.Address,
		10*config.AtomicUnitsPerVDR,
		config.GenesisTimestamp+90,
	)
	if err != nil {
		t.Fatal(err)
	}

	var sent SendTransactionResult
	if err := client.Call(
		ctx,
		MethodSendTransaction,
		SendTransactionParams{Transaction: payment},
		&sent,
	); err != nil {
		t.Fatal(err)
	}
	if sent.TransactionID != payment.TransactionID {
		t.Fatalf("send txid = %s, want %s", sent.TransactionID, payment.TransactionID)
	}
	if pool.Len() != 1 {
		t.Fatalf("mempool length = %d, want 1", pool.Len())
	}
	if err := client.Call(ctx, MethodGetStatus, nil, &status); err != nil {
		t.Fatal(err)
	}
	if status.MempoolCount != 1 ||
		status.MempoolSizeBytes != uint64(payment.SerializedSize()) {
		t.Fatalf("mempool diagnostics = %+v payment_size=%d", status, payment.SerializedSize())
	}

	var mempoolTxs []*transaction.Transaction
	if err := client.Call(ctx, MethodGetMempool, nil, &mempoolTxs); err != nil {
		t.Fatal(err)
	}
	if len(mempoolTxs) != 1 || mempoolTxs[0].TransactionID != payment.TransactionID {
		t.Fatalf("mempool = %+v", mempoolTxs)
	}

	var pending TransactionResult
	if err := client.Call(
		ctx,
		MethodGetTransaction,
		TransactionParams{TransactionID: payment.TransactionID},
		&pending,
	); err != nil {
		t.Fatal(err)
	}
	if pending.Confirmed {
		t.Fatal("mempool transaction reported as confirmed")
	}

	block2, err := mining.MineBlock(
		chain,
		minerWallet.Address,
		config.GenesisTimestamp+120,
		[]*transaction.Transaction{payment},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := node.BroadcastBlock(block2); err != nil {
		t.Fatal(err)
	}

	var confirmed TransactionResult
	if err := client.Call(
		ctx,
		MethodGetTransaction,
		TransactionParams{TransactionID: payment.TransactionID},
		&confirmed,
	); err != nil {
		t.Fatal(err)
	}
	if !confirmed.Confirmed || confirmed.BlockHeight != 2 || confirmed.BlockHash != block2.BlockHash {
		t.Fatalf("confirmed transaction = %+v", confirmed)
	}

	var recipientBalance BalanceResult
	if err := client.Call(
		ctx,
		MethodGetBalance,
		AddressParams{Address: recipientWallet.Address},
		&recipientBalance,
	); err != nil {
		t.Fatal(err)
	}
	if recipientBalance.BalanceVal != 10*config.AtomicUnitsPerVDR {
		t.Fatalf("recipient balance = %d, want 10 VDR", recipientBalance.BalanceVal)
	}

	var peers []p2p.Peer
	if err := client.Call(ctx, MethodGetPeers, nil, &peers); err != nil {
		t.Fatal(err)
	}
	if len(peers) != 0 {
		t.Fatalf("peers = %+v, want none", peers)
	}
}

func TestRPCMethodAndNotFoundErrors(t *testing.T) {
	chain := blockchain.New()
	node, err := p2p.NewNode(p2p.NodeConfig{
		NodeID:        "rpc-error-node",
		ListenAddress: "127.0.0.1:0",
		Blockchain:    chain,
		Mempool:       mempool.New(),
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(chain, node)
	if err != nil {
		t.Fatal(err)
	}

	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	client := NewClient(httpServer.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var candidate block.Block
	err = client.Call(ctx, MethodGetBlock, HeightParams{Height: 99}, &candidate)
	if !errors.Is(err, ErrRPCResponse) {
		t.Fatalf("missing block error = %v, want ErrRPCResponse", err)
	}

	err = client.Call(ctx, "notAMethod", nil, &candidate)
	if !errors.Is(err, ErrRPCResponse) {
		t.Fatalf("unknown method error = %v, want ErrRPCResponse", err)
	}
}


func TestBlocksUntilRetarget(t *testing.T) {
	tests := []struct {
		nextHeight uint64
		interval   uint64
		want       uint64
	}{
		{nextHeight: 1, interval: 60, want: 59},
		{nextHeight: 59, interval: 60, want: 1},
		{nextHeight: 60, interval: 60, want: 0},
		{nextHeight: 61, interval: 60, want: 59},
		{nextHeight: 180, interval: 60, want: 0},
		{nextHeight: 10, interval: 0, want: 0},
	}
	for _, tc := range tests {
		if got := blocksUntilRetarget(tc.nextHeight, tc.interval); got != tc.want {
			t.Fatalf(
				"blocksUntilRetarget(%d, %d)=%d want=%d",
				tc.nextHeight,
				tc.interval,
				got,
				tc.want,
			)
		}
	}
}
