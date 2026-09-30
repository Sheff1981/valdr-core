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
	// Desktop history must not let frequent mining rewards hide ordinary
	// wallet transfers from the Transactions view. 500 items is still small
	// enough for the current local Testnet explorer path while covering
	// substantially more than the previous 100-entry window.
	DefaultHistoryLimit = 500
	MaxHistoryLimit     = 2000
)

var ErrInvalidHistoryAddress = errors.New("invalid history address")

type TransactionHistoryItem struct {
	Status        string   `json:"status"`
	Direction     string   `json:"direction"`
	Type          string   `json:"type"`
	Addresses     []string `json:"addresses,omitempty"`
	TransactionID string   `json:"transaction_id"`
	Timestamp     int64    `json:"timestamp"`
	AmountVal     uint64   `json:"amount_val"`
	AmountVDR     string   `json:"amount_vdr"`
	FeeVal        uint64   `json:"fee_val"`
	FeeVDR        string   `json:"fee_vdr"`
	BlockHeight   uint64   `json:"block_height,omitempty"`
	BlockHash     string   `json:"block_hash,omitempty"`
	Confirmations uint64   `json:"confirmations"`
}

type HistoryService struct {
	client  RPCClient
	index   *explorer.Index
	journal *historyJournal
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
	journal, err := newHistoryJournal(indexPath + ".wallet-history.json")
	if err != nil {
		return nil, err
	}
	return &HistoryService{
		client:  client,
		index:   index,
		journal: journal,
	}, nil
}

func (s *HistoryService) Rescan(ctx context.Context, address string) error {
	if !valdrcrypto.ValidateAddress(address) {
		return ErrInvalidHistoryAddress
	}
	return s.index.Rebuild(ctx)
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

	confirmed, err := s.confirmedHistory(
		ctx,
		address,
		status.Height,
	)
	if err != nil {
		return nil, err
	}
	pending, err := s.pendingHistory(ctx, address)
	if err != nil {
		return nil, err
	}

	result := append(pending, confirmed...)
	if s.journal != nil {
		result, err = s.journal.Merge(address, result)
		if err != nil {
			return nil, err
		}
	}
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
		// Mining rewards can be frequent on Testnet. Never let coinbase rows
		// crowd ordinary wallet transfers out of the visible history window.
		// Keep the newest non-mined activity first, then use the remaining
		// budget for mined rewards while preserving the already-sorted order.
		nonMined := 0
		for _, item := range result {
			if item.Type != "mined" {
				nonMined++
			}
		}
		if nonMined > limit {
			nonMined = limit
		}
		minedBudget := limit - nonMined
		keptNonMined := 0
		keptMined := 0
		trimmed := make([]TransactionHistoryItem, 0, limit)
		for _, item := range result {
			if item.Type == "mined" {
				if keptMined >= minedBudget {
					continue
				}
				keptMined++
			} else {
				if keptNonMined >= nonMined {
					continue
				}
				keptNonMined++
			}
			trimmed = append(trimmed, item)
			if len(trimmed) == limit {
				break
			}
		}
		result = trimmed
	}
	return result, nil
}

func (s *HistoryService) confirmedHistory(
	ctx context.Context,
	address string,
	tipHeight uint64,
) ([]TransactionHistoryItem, error) {
	activities := s.index.Activities(address)
	result := make([]TransactionHistoryItem, 0, len(activities))
	for _, activity := range activities {
		var transactionResult rpc.TransactionResult
		if err := s.client.Call(
			ctx,
			rpc.MethodGetTransaction,
			rpc.TransactionParams{TransactionID: activity.TransactionID},
			&transactionResult,
		); err != nil {
			return nil, err
		}
		itemType := "transfer"
		addresses := transactionAddresses(transactionResult.Transaction)
		if transactionResult.Transaction != nil && transactionResult.Transaction.IsCoinbase() {
			itemType = "mined"
		}
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
			Type:          itemType,
			Addresses:     addresses,
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
	return result, nil
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
		Type:          "transfer",
		Addresses:     transactionAddresses(tx),
		TransactionID: tx.TransactionID,
		Timestamp:     tx.Timestamp,
		AmountVal:     amount,
		AmountVDR:     FormatVDR(amount),
		FeeVal:        fee,
		FeeVDR:        FormatVDR(fee),
		Confirmations: 0,
	}, true
}

func transactionAddresses(tx *transaction.Transaction) []string {
	if tx == nil {
		return nil
	}
	seen := make(map[string]struct{})
	addresses := make([]string, 0, len(tx.Outputs)+1)
	add := func(address string) {
		if address == "" {
			return
		}
		if _, exists := seen[address]; exists {
			return
		}
		seen[address] = struct{}{}
		addresses = append(addresses, address)
	}
	if !tx.IsCoinbase() {
		add(transactionSenderAddress(tx))
	}
	for _, output := range tx.Outputs {
		add(output.Recipient)
	}
	sort.Strings(addresses)
	return addresses
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

