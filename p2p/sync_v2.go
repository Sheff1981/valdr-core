package p2p

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/transaction"
)

const (
	V2MaxLocatorEntries = 128
	V2MaxHeaders         = 2000
	V2MaxInventoryItems  = 32
	V2InventoryBlock     = "block"
)

var (
	ErrV2InvalidLocator   = errors.New("invalid P2P v2 block locator")
	ErrV2InvalidHeaders   = errors.New("invalid P2P v2 headers payload")
	ErrV2InvalidInventory = errors.New("invalid P2P v2 inventory payload")
)

type V2LocatorRequest struct {
	Locator  []string \`json:"locator"\`
	StopHash string   \`json:"stop_hash"\`
}

type V2HeadersPayload struct {
	Headers []block.Header \`json:"headers"\`
}

type V2InventoryItem struct {
	Kind string \`json:"kind"\`
	Hash string \`json:"hash"\`
}

type V2GetDataPayload struct {
	Items []V2InventoryItem \`json:"items"\`
}

type V2InvPayload struct {
	Items []V2InventoryItem \`json:"items"\`
}

type V2BlockPayload struct {
	Block *block.Block \`json:"block"\`
}

type V2TxPayload struct {
	Transaction *transaction.Transaction \`json:"transaction"\`
}

func ValidateV2LocatorRequest(request V2LocatorRequest) error {
	if len(request.Locator) == 0 || len(request.Locator) > V2MaxLocatorEntries {
		return fmt.Errorf(
			"%w: locator count=%d",
			ErrV2InvalidLocator,
			len(request.Locator),
		)
	}
	seen := make(map[string]struct{}, len(request.Locator))
	for index, hash := range request.Locator {
		if !isV2Hash(hash) {
			return fmt.Errorf("%w: locator[%d]", ErrV2InvalidLocator, index)
		}
		if _, exists := seen[hash]; exists {
			return fmt.Errorf("%w: duplicate locator hash", ErrV2InvalidLocator)
		}
		seen[hash] = struct{}{}
	}
	if request.StopHash != "" && !isV2Hash(request.StopHash) {
		return fmt.Errorf("%w: stop_hash", ErrV2InvalidLocator)
	}
	return nil
}

func ValidateV2HeadersPayload(payload V2HeadersPayload) error {
	if len(payload.Headers) > V2MaxHeaders {
		return fmt.Errorf(
			"%w: header count=%d",
			ErrV2InvalidHeaders,
			len(payload.Headers),
		)
	}
	for index, header := range payload.Headers {
		if !isV2Hash(header.BlockHash) {
			return fmt.Errorf("%w: header[%d] block_hash", ErrV2InvalidHeaders, index)
		}
		if header.Height > 0 && !isV2Hash(header.PreviousBlockHash) {
			return fmt.Errorf("%w: header[%d] previous hash", ErrV2InvalidHeaders, index)
		}
		if header.BlockHash != header.CalculateHash() {
			return fmt.Errorf("%w: header[%d] hash mismatch", ErrV2InvalidHeaders, index)
		}
		if index > 0 {
			parent := payload.Headers[index-1]
			if header.Height != parent.Height+1 ||
				header.PreviousBlockHash != parent.BlockHash {
				return fmt.Errorf("%w: discontinuity at %d", ErrV2InvalidHeaders, index)
			}
		}
	}
	return nil
}

func ValidateV2GetDataPayload(payload V2GetDataPayload) error {
	return validateV2Inventory(payload.Items, false)
}

func ValidateV2InvPayload(payload V2InvPayload) error {
	return validateV2Inventory(payload.Items, true)
}

func validateV2Inventory(items []V2InventoryItem, allowEmpty bool) error {
	if (!allowEmpty && len(items) == 0) || len(items) > V2MaxInventoryItems {
		return fmt.Errorf(
			"%w: item count=%d",
			ErrV2InvalidInventory,
			len(items),
		)
	}
	seen := make(map[string]struct{}, len(items))
	for index, item := range items {
		if item.Kind != V2InventoryBlock {
			return fmt.Errorf(
				"%w: item[%d] kind=%q",
				ErrV2InvalidInventory,
				index,
				item.Kind,
			)
		}
		if !isV2Hash(item.Hash) {
			return fmt.Errorf("%w: item[%d] hash", ErrV2InvalidInventory, index)
		}
		if _, exists := seen[item.Hash]; exists {
			return fmt.Errorf("%w: duplicate hash", ErrV2InvalidInventory)
		}
		seen[item.Hash] = struct{}{}
	}
	return nil
}

func isV2Hash(value string) bool {
	if len(value) != 64 {
		return false
	}
	raw, err := hex.DecodeString(value)
	if err != nil || len(raw) != 32 {
		return false
	}
	for _, ch := range value {
		if ch >= 'A' && ch <= 'F' {
			return false
		}
	}
	return true
}
