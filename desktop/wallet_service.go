package desktop

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/core/utxo"
	"github.com/Sheff1981/valdr-core/rpc"
	"github.com/Sheff1981/valdr-core/wallet"
)

var ErrWalletServiceConfig = errors.New("invalid Desktop wallet service configuration")

type RPCClient interface {
	Call(context.Context, string, any, any) error
}

type WalletService struct {
	store  *wallet.Store
	client RPCClient
}

type WalletBalance struct {
	Address    string `json:"address"`
	BalanceVal uint64 `json:"balance_val"`
	BalanceVDR string `json:"balance_vdr"`
}

type SendPreview struct {
	AmountVal uint64 `json:"amount_val"`
	AmountVDR string `json:"amount_vdr"`
	FeeVal    uint64 `json:"fee_val"`
	FeeVDR    string `json:"fee_vdr"`
	TotalVal  uint64 `json:"total_val"`
	TotalVDR  string `json:"total_vdr"`
}

type SendResult struct {
	TransactionID string `json:"transaction_id"`
	SendPreview
}

func NewWalletService(
	store *wallet.Store,
	client RPCClient,
) (*WalletService, error) {
	if store == nil || client == nil {
		return nil, ErrWalletServiceConfig
	}
	return &WalletService{store: store, client: client}, nil
}

func (s *WalletService) Balance(
	ctx context.Context,
	address string,
) (WalletBalance, error) {
	var result rpc.BalanceResult
	if err := s.client.Call(
		ctx,
		rpc.MethodGetBalance,
		rpc.AddressParams{Address: address},
		&result,
	); err != nil {
		return WalletBalance{}, err
	}
	return WalletBalance{
		Address:    result.Address,
		BalanceVal: result.BalanceVal,
		BalanceVDR: FormatVDR(result.BalanceVal),
	}, nil
}

func (s *WalletService) PreviewSend(
	ctx context.Context,
	selector string,
	passphrase []byte,
	recipient string,
	amount uint64,
) (SendPreview, error) {
	_, fee, err := s.buildTransaction(
		ctx,
		selector,
		passphrase,
		recipient,
		amount,
	)
	if err != nil {
		return SendPreview{}, err
	}
	return sendPreview(amount, fee)
}

func (s *WalletService) Send(
	ctx context.Context,
	selector string,
	passphrase []byte,
	recipient string,
	amount uint64,
) (SendResult, error) {
	tx, fee, err := s.buildTransaction(
		ctx,
		selector,
		passphrase,
		recipient,
		amount,
	)
	if err != nil {
		return SendResult{}, err
	}

	var sent rpc.SendTransactionResult
	if err := s.client.Call(
		ctx,
		rpc.MethodSendTransaction,
		rpc.SendTransactionParams{Transaction: tx},
		&sent,
	); err != nil {
		return SendResult{}, err
	}

	preview, err := sendPreview(amount, fee)
	if err != nil {
		return SendResult{}, err
	}
	return SendResult{
		TransactionID: sent.TransactionID,
		SendPreview:   preview,
	}, nil
}

func (s *WalletService) buildTransaction(
	ctx context.Context,
	selector string,
	passphrase []byte,
	recipient string,
	amount uint64,
) (*transaction.Transaction, uint64, error) {
	secret := append([]byte(nil), passphrase...)
	defer clearBytes(secret)

	source, err := s.store.Unlock(selector, secret)
	if err != nil {
		return nil, 0, err
	}

	var status rpc.StatusResult
	if err := s.client.Call(
		ctx,
		rpc.MethodGetStatus,
		nil,
		&status,
	); err != nil {
		return nil, 0, err
	}
	profile, err := config.ResolveNetworkProfile(status.Network)
	if err != nil {
		return nil, 0, err
	}
	if profile.Name != config.NetworkTestnetV029 {
		return nil, 0, ErrDesktopMainnet
	}

	var available []utxo.UTXO
	if err := s.client.Call(
		ctx,
		rpc.MethodGetUTXOs,
		rpc.AddressParams{Address: source.Address},
		&available,
	); err != nil {
		return nil, 0, err
	}

	tx, fee, err := source.CreateTransactionForChain(
		profile.ChainID,
		available,
		recipient,
		amount,
		profile.MinRelayFeePerByte,
		time.Now().UTC().Unix(),
	)
	if err != nil {
		return nil, 0, err
	}

	return tx, fee, nil
}

func sendPreview(amount, fee uint64) (SendPreview, error) {
	if math.MaxUint64-amount < fee {
		return SendPreview{}, wallet.ErrWalletFeeOverflow
	}
	total := amount + fee
	return SendPreview{
		AmountVal: amount,
		AmountVDR: FormatVDR(amount),
		FeeVal:    fee,
		FeeVDR:    FormatVDR(fee),
		TotalVal:  total,
		TotalVDR:  FormatVDR(total),
	}, nil
}

func clearBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
