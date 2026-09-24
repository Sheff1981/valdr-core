package desktop

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/core/utxo"
	"github.com/Sheff1981/valdr-core/explorer"
	"github.com/Sheff1981/valdr-core/rpc"
)

const (
	DefaultHistoryLimit = 100
	MaxHistoryLimit     = 200
)

var ErrInvalidHistoryAddress = errors.New("invalid history address")

type TransactionHistoryItem struct {
	Status        string `json:"status"`
	Direction     string `json:"direction"`
	TransactionID string `json:"transaction_id"`
	Timestamp     int64  `json:"timestamp"`
	AmountVal     uint64 `json:"amount_val"`
	AmountVDR     string `json:"amount_vdr"`
	FeeVal        uint64 `json:"fee_val"`
	FeeVDR        string `json:"fee_vdr"`
	BlockHeight   uint64 `json:"block_height,omitempty"`
	BlockHash     string `json:"block_hash,omitempty"`
	Confirmations uint64 `json:"confirmations"`
}

type HistoryService struct {
	client RPCClient
	index  *explorer.Index
}

func NewHistoryService(
	client RPCClient,
	indexPath string,
) (*HistoryService, error) {
	if client == nil {
		return nil, ErrWalletServiceConfig
	}
	index, err := explorer.NewIndex(client, indexPath)
	if err != nil {
		return nil, err
	}
	return &HistoryService{
		client: client,
		index:  index,
	}, nil
}

func (s *HistoryService) History(
	ctx context.Context,
	address string,
	limit int,
) ([]TransactionHistoryItem, error) {
	if !valdrcrypto.ValidateAddress(address) {
		return nil, ErrInvalidHistoryAddress
	}
	if limit <= 0 {
		limit = DefaultHistoryLimit
	}
	if limit > MaxHistoryLimit {
		limit = MaxHistoryLimit
	}

	if err := s.index.Refresh(ctx); err != nil {
		return nil, err
	}

	var status rpc.StatusResult
	if err := s.client.Call(
		ctx,
		rpc.MethodGetStatus,
		nil,
		&status,
	); err != nil {
		return nil, err
	}

	confirmed := s.confirmedHistory(
		address,
		status.Height,
	)
	pending, err := s.pendingHistory(ctx, address)
	if err != nil {
		return nil, err
	}

	result := append(pending, confirmed...)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Status != result[j].Status {
			return result[i].Status == "pending"
		}
		if result[i].Timestamp != result[j].Timestamp {
			return result[i].Timestamp > result[j].Timestamp
		}
		return result[i].TransactionID > result[j].TransactionID
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (s *HistoryService) confirmedHistory(
	address string,
	tipHeight uint64,
) []TransactionHistoryItem {
	activities := s.index.Activities(address)
	result := make([]TransactionHistoryItem, 0, len(activities))
	for _, activity := range activities {
		direction := "received"
		amount := activity.ReceivedVal
		fee := uint64(0)
		if activity.SpentVal > 0 {
			direction = "sent"
			fee = activity.FeeVal
			if activity.SpentVal >= activity.ReceivedVal+fee {
				amount = activity.SpentVal -
					activity.ReceivedVal -
					fee
			} else {
				amount = 0
			}
			if amount == 0 {
				direction = "self"
			}
		}
		confirmations := uint64(0)
		if tipHeight >= activity.BlockHeight {
			confirmations = tipHeight -
				activity.BlockHeight +
				1
		}
		result = append(result, TransactionHistoryItem{
			Status:        "confirmed",
			Direction:     direction,
			TransactionID: activity.TransactionID,
			Timestamp:     activity.Timestamp,
			AmountVal:     amount,
			AmountVDR:     FormatVDR(amount),
			FeeVal:        fee,
			FeeVDR:        FormatVDR(fee),
			BlockHeight:   activity.BlockHeight,
			BlockHash:     activity.BlockHash,
			Confirmations: confirmations,
		})
	}
	return result
}

func (s *HistoryService) pendingHistory(
	ctx context.Context,
	address string,
) ([]TransactionHistoryItem, error) {
	var transactions []*transaction.Transaction
	if err := s.client.Call(
		ctx,
		rpc.MethodGetMempool,
		nil,
		&transactions,
	); err != nil {
		return nil, err
	}

	var available []utxo.UTXO
	if err := s.client.Call(
		ctx,
		rpc.MethodGetUTXOs,
		rpc.AddressParams{Address: address},
		&available,
	); err != nil {
		return nil, err
	}
	outpoints := make(map[string]uint64, len(available))
	for _, item := range available {
		outpoints[historyOutpointKey(
			item.TransactionID,
			item.OutputIndex,
		)] = item.Amount
	}

	result := make([]TransactionHistoryItem, 0)
	for _, tx := range transactions {
		item, ok := pendingHistoryItem(
			tx,
			address,
			outpoints,
		)
		if ok {
			result = append(result, item)
		}
	}
	return result, nil
}

func pendingHistoryItem(
	tx *transaction.Transaction,
	address string,
	outpoints map[string]uint64,
) (TransactionHistoryItem, bool) {
	if tx == nil || tx.IsCoinbase() {
		return TransactionHistoryItem{}, false
	}

	sender := transactionSenderAddress(tx)
	var received uint64
	var totalOutput uint64
	for _, output := range tx.Outputs {
		totalOutput += output.Amount
		if output.Recipient == address {
			received += output.Amount
		}
	}

	direction := ""
	amount := uint64(0)
	fee := uint64(0)

	if sender == address {
		direction = "sent"
		external := totalOutput - received
		amount = external
		if amount == 0 {
			direction = "self"
		}

		var spent uint64
		allInputsKnown := true
		for _, input := range tx.Inputs {
			value, ok := outpoints[historyOutpointKey(
				input.PreviousTransactionID,
				input.OutputIndex,
			)]
			if !ok {
				allInputsKnown = false
				break
			}
			spent += value
		}
		if allInputsKnown && spent >= totalOutput {
			fee = spent - totalOutput
		}
	} else if received > 0 {
		direction = "received"
		amount = received
	} else {
		return TransactionHistoryItem{}, false
	}

	return TransactionHistoryItem{
		Status:        "pending",
		Direction:     direction,
		TransactionID: tx.TransactionID,
		Timestamp:     tx.Timestamp,
		AmountVal:     amount,
		AmountVDR:     FormatVDR(amount),
		FeeVal:        fee,
		FeeVDR:        FormatVDR(fee),
		Confirmations: 0,
	}, true
}

func transactionSenderAddress(
	tx *transaction.Transaction,
) string {
	if tx == nil || tx.PublicKey == "" {
		return ""
	}
	publicKey, err := valdrcrypto.DecodePublicKey(
		tx.PublicKey,
	)
	if err != nil {
		return ""
	}
	address, err := valdrcrypto.AddressFromPublicKey(
		publicKey,
	)
	if err != nil {
		return ""
	}
	return address
}

func historyOutpointKey(
	transactionID string,
	outputIndex uint32,
) string {
	return fmt.Sprintf(
		"%s:%s",
		transactionID,
		strconv.FormatUint(uint64(outputIndex), 10),
	)
}

