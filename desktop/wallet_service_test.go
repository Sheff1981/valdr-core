package desktop

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/core/utxo"
	"github.com/Sheff1981/valdr-core/rpc"
	"github.com/Sheff1981/valdr-core/wallet"
)

type walletServiceRPCMock struct {
	status rpc.StatusResult
	utxos  []utxo.UTXO
	balance rpc.BalanceResult
	sent   *transaction.Transaction
}

func (m *walletServiceRPCMock) Call(
	_ context.Context,
	method string,
	params any,
	result any,
) error {
	switch method {
	case rpc.MethodGetStatus:
		return assignDesktopRPC(result, m.status)
	case rpc.MethodGetUTXOs:
		return assignDesktopRPC(result, m.utxos)
	case rpc.MethodGetBalance:
		return assignDesktopRPC(result, m.balance)
	case rpc.MethodSendTransaction:
		p := params.(rpc.SendTransactionParams)
		m.sent = p.Transaction
		return assignDesktopRPC(result, rpc.SendTransactionResult{
			TransactionID: p.Transaction.TransactionID,
		})
	default:
		panic("unexpected RPC method: " + method)
	}
}

func assignDesktopRPC(target, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func TestWalletServiceSendsNetworkBoundTestnetTransaction(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV02)
	if err != nil {
		t.Fatal(err)
	}
	store := wallet.NewStore(filepath.Join(t.TempDir(), "wallets"))
	passphrase := []byte("desktop-wallet-service-passphrase")
	source, err := store.CreateEncrypted("source", passphrase)
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := wallet.New("recipient")
	if err != nil {
		t.Fatal(err)
	}

	mock := &walletServiceRPCMock{
		status: rpc.StatusResult{
			Network: profile.Name,
			ChainID: profile.ChainID,
		},
		utxos: []utxo.UTXO{{
			TransactionID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			OutputIndex:   0,
			Amount:        2_000_000,
			Recipient:     source.Address,
		}},
		balance: rpc.BalanceResult{
			Address:    source.Address,
			BalanceVal: 2_000_000,
		},
	}
	service, err := NewWalletService(store, mock)
	if err != nil {
		t.Fatal(err)
	}

	balance, err := service.Balance(context.Background(), source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if balance.BalanceVal != 2_000_000 ||
		balance.BalanceVDR != "0.02000000" {
		t.Fatalf("unexpected balance: %+v", balance)
	}

	preview, err := service.PreviewSend(
		context.Background(),
		"source",
		passphrase,
		recipient.Address,
		1_000_000,
	)
	if err != nil {
		t.Fatal(err)
	}
	if mock.sent != nil {
		t.Fatal("preview unexpectedly broadcast a transaction")
	}
	if preview.AmountVal != 1_000_000 ||
		preview.FeeVal == 0 ||
		preview.TotalVal != preview.AmountVal+preview.FeeVal {
		t.Fatalf("unexpected send preview: %+v", preview)
	}

	result, err := service.Send(
		context.Background(),
		"source",
		passphrase,
		recipient.Address,
		1_000_000,
	)
	if err != nil {
		t.Fatal(err)
	}
	if mock.sent == nil {
		t.Fatal("transaction was not broadcast")
	}
	if mock.sent.Version != transaction.VersionV2 ||
		mock.sent.ChainID != profile.ChainID {
		t.Fatalf("wrong transaction network identity: %+v", mock.sent)
	}
	if err := mock.sent.ValidateForChain(profile.ChainID); err != nil {
		t.Fatal(err)
	}
	if result.TransactionID != mock.sent.TransactionID {
		t.Fatalf("send result txid=%s want=%s", result.TransactionID, mock.sent.TransactionID)
	}
	if result.FeeVal < uint64(mock.sent.SerializedSize())*profile.MinRelayFeePerByte {
		t.Fatalf(
			"fee=%d below relay minimum size=%d rate=%d",
			result.FeeVal,
			mock.sent.SerializedSize(),
			profile.MinRelayFeePerByte,
		)
	}
	if result.TotalVal != result.AmountVal+result.FeeVal {
		t.Fatalf("total mismatch: %+v", result)
	}
}

func TestParseAndFormatVDRDesktop(t *testing.T) {
	tests := []struct {
		text string
		val  uint64
	}{
		{"1", 100_000_000},
		{"0.00000001", 1},
		{"12.34000000", 1_234_000_000},
	}
	for _, tc := range tests {
		got, err := ParseVDR(tc.text)
		if err != nil {
			t.Fatalf("ParseVDR(%q): %v", tc.text, err)
		}
		if got != tc.val {
			t.Fatalf("ParseVDR(%q)=%d want=%d", tc.text, got, tc.val)
		}
	}
	if got := FormatVDR(1_234_000_000); got != "12.34000000" {
		t.Fatalf("FormatVDR=%q", got)
	}
	for _, bad := range []string{"", "-1", "+1", ".1", "1.", "1.000000000"} {
		if _, err := ParseVDR(bad); err == nil {
			t.Fatalf("invalid amount %q accepted", bad)
		}
	}
}
