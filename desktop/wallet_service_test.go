package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/core/utxo"
	"github.com/Sheff1981/valdr-core/rpc"
	"github.com/Sheff1981/valdr-core/wallet"
)

type walletServiceRPCMock struct {
	status  rpc.StatusResult
	utxos   []utxo.UTXO
	mempool []*transaction.Transaction
	balance rpc.BalanceResult
	sent    *transaction.Transaction
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
	case rpc.MethodGetMempool:
		return assignDesktopRPC(result, m.mempool)
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
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
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
		balance.BalanceVDR != "0.02000000" ||
		balance.SpendableVal != 2_000_000 ||
		balance.SpendableVDR != "0.02000000" ||
		balance.PendingVal != 0 ||
		balance.PendingVDR != "0.00000000" ||
		balance.TotalVal != 2_000_000 ||
		balance.TotalVDR != "0.02000000" {
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

func TestWalletServiceBalanceAccountsForMempoolReservationsAndPendingOutputs(t *testing.T) {
	store := wallet.NewStore(filepath.Join(t.TempDir(), "wallets"))
	source, err := store.CreateEncrypted(
		"balance-source",
		[]byte("balance-source-passphrase"),
	)
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := wallet.New("balance-recipient")
	if err != nil {
		t.Fatal(err)
	}

	const confirmedTxID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	mock := &walletServiceRPCMock{
		utxos: []utxo.UTXO{{
			TransactionID: confirmedTxID,
			OutputIndex:   0,
			Amount:        2_000_000,
			Recipient:     source.Address,
		}},
		mempool: []*transaction.Transaction{
			{
				Inputs: []transaction.Input{{
					PreviousTransactionID: confirmedTxID,
					OutputIndex:           0,
				}},
				Outputs: []transaction.Output{
					{Amount: 600_000, Recipient: recipient.Address},
					{Amount: 1_300_000, Recipient: source.Address},
				},
			},
			{
				Outputs: []transaction.Output{
					{Amount: 200_000, Recipient: source.Address},
				},
			},
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
	if balance.SpendableVal != 0 {
		t.Fatalf("spendable=%d want=0", balance.SpendableVal)
	}
	if balance.PendingVal != 1_500_000 {
		t.Fatalf("pending=%d want=1500000", balance.PendingVal)
	}
	if balance.TotalVal != 1_500_000 {
		t.Fatalf("total=%d want=1500000", balance.TotalVal)
	}
	if balance.BalanceVal != balance.SpendableVal {
		t.Fatalf("compatibility balance=%d spendable=%d", balance.BalanceVal, balance.SpendableVal)
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


func TestWalletSessionAutoLocksAndClearsCachedPassphrase(t *testing.T) {
	store := wallet.NewStore(filepath.Join(t.TempDir(), "wallets"))
	passphrase := []byte("desktop-session-passphrase")
	source, err := store.CreateEncrypted("session-wallet", passphrase)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1_790_208_000, 0)
	manager, err := newWalletSessionManager(
		store,
		5*time.Minute,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Unlock(source.Address, []byte("wrong")); !errors.Is(
		err,
		wallet.ErrWalletAuthentication,
	) {
		t.Fatalf("wrong passphrase error=%v", err)
	}
	if manager.IsUnlocked(source.Address) {
		t.Fatal("wallet unlocked after wrong passphrase")
	}

	if _, err := manager.Unlock(source.Address, passphrase); err != nil {
		t.Fatal(err)
	}
	if !manager.IsUnlocked(source.Address) {
		t.Fatal("wallet is not unlocked")
	}
	backing := manager.sessions[source.Address].passphrase
	cached, err := manager.Passphrase(source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if string(cached) != string(passphrase) {
		t.Fatal("cached passphrase does not match")
	}
	clearBytes(cached)

	now = now.Add(5*time.Minute + time.Second)
	if manager.IsUnlocked(source.Address) {
		t.Fatal("wallet did not auto-lock after inactivity")
	}
	if _, err := manager.Passphrase(source.Address); !errors.Is(
		err,
		ErrWalletLocked,
	) {
		t.Fatalf("expired session error=%v", err)
	}
	for i, b := range backing {
		if b != 0 {
			t.Fatalf("cached passphrase byte %d was not cleared", i)
		}
	}
}

func TestWalletSessionTimeoutIsConfigurable(t *testing.T) {
	store := wallet.NewStore(filepath.Join(t.TempDir(), "wallets"))
	source, err := store.CreateEncrypted(
		"timeout-wallet",
		[]byte("timeout-passphrase"),
	)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1_790_208_000, 0)
	manager, err := newWalletSessionManager(
		store,
		DefaultWalletAutoLock,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Unlock(
		source.Address,
		[]byte("timeout-passphrase"),
	); err != nil {
		t.Fatal(err)
	}

	now = now.Add(6 * time.Minute)
	if err := manager.SetTimeout(5 * time.Minute); err != nil {
		t.Fatal(err)
	}
	if manager.IsUnlocked(source.Address) {
		t.Fatal("shorter timeout did not expire inactive wallet")
	}
	if err := manager.SetTimeout(0); !errors.Is(err, ErrWalletAutoLock) {
		t.Fatalf("invalid timeout error=%v", err)
	}
	if err := manager.SetTimeout(25 * time.Hour); !errors.Is(err, ErrWalletAutoLock) {
		t.Fatalf("oversized timeout error=%v", err)
	}
}


func TestWalletServiceRejectsHistoricalTestnet1ForSending(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV02)
	if err != nil {
		t.Fatal(err)
	}
	store := wallet.NewStore(filepath.Join(t.TempDir(), "wallets"))
	passphrase := []byte("desktop-old-testnet-passphrase")
	source, err := store.CreateEncrypted("source-old-testnet", passphrase)
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := wallet.New("recipient-old-testnet")
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewWalletService(store, &walletServiceRPCMock{
		status: rpc.StatusResult{
			Network: profile.Name,
			ChainID: profile.ChainID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.PreviewSend(
		context.Background(),
		source.Address,
		passphrase,
		recipient.Address,
		1,
	); !errors.Is(err, ErrDesktopMainnet) {
		t.Fatalf("historical Testnet1 send error=%v want ErrDesktopMainnet", err)
	}
}
