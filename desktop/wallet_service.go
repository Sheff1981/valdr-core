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

var (
	ErrWalletServiceConfig   = errors.New("invalid Desktop wallet service configuration")
	ErrWalletBalanceOverflow = errors.New("wallet balance overflow")
)

type RPCClient interface {
	Call(context.Context, string, any, any) error
}

type WalletService struct {
	store  *wallet.Store
	client RPCClient
}

type WalletBalance struct {
	Address      string `json:"address"`
	BalanceVal   uint64 `json:"balance_val"`
	BalanceVDR   string `json:"balance_vdr"`
	SpendableVal uint64 `json:"spendable_val"`
	SpendableVDR string `json:"spendable_vdr"`
	PendingVal   uint64 `json:"pending_val"`
	PendingVDR   string `json:"pending_vdr"`
	TotalVal     uint64 `json:"total_val"`
	TotalVDR     string `json:"total_vdr"`
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
	var available []utxo.UTXO
	if err := s.client.Call(
		ctx,
		rpc.MethodGetUTXOs,
		rpc.AddressParams{Address: address},
		&available,
	); err != nil {
		return WalletBalance{}, err
	}

	type outpoint struct {
		transactionID string
		outputIndex   uint32
	}
	owned := make(map[outpoint]uint64, len(available))
	var spendable uint64
	for _, item := range available {
		if math.MaxUint64-spendable < item.Amount {
			return WalletBalance{}, ErrWalletBalanceOverflow
		}
		spendable += item.Amount
		owned[outpoint{item.TransactionID, item.OutputIndex}] = item.Amount
	}

	var mempool []*transaction.Transaction
	if err := s.client.Call(
		ctx,
		rpc.MethodGetMempool,
		nil,
		&mempool,
	); err != nil {
		return WalletBalance{}, err
	}

	spent := make(map[outpoint]struct{})
	var pending uint64
	for _, tx := range mempool {
		if tx == nil || tx.IsCoinbase() {
			continue
		}
		for _, input := range tx.Inputs {
			key := outpoint{input.PreviousTransactionID, input.OutputIndex}
			amount, ok := owned[key]
			if !ok {
				continue
			}
			if _, alreadyReserved := spent[key]; alreadyReserved {
				continue
			}
			if amount > spendable {
				return WalletBalance{}, ErrWalletBalanceOverflow
			}
			spendable -= amount
			spent[key] = struct{}{}
		}
		for _, output := range tx.Outputs {
			if output.Recipient != address {
				continue
			}
			if math.MaxUint64-pending < output.Amount {
				return WalletBalance{}, ErrWalletBalanceOverflow
			}
			pending += output.Amount
		}
	}
	if math.MaxUint64-spendable < pending {
		return WalletBalance{}, ErrWalletBalanceOverflow
	}
	total := spendable + pending

	// balance_val/balance_vdr remain compatibility aliases for spendable.
	return WalletBalance{
		Address:      address,
		BalanceVal:   spendable,
		BalanceVDR:   FormatVDR(spendable),
		SpendableVal: spendable,
		SpendableVDR: FormatVDR(spendable),
		PendingVal:   pending,
		PendingVDR:   FormatVDR(pending),
		TotalVal:     total,
		TotalVDR:     FormatVDR(total),
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
