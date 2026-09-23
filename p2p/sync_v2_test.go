package p2p

import (
	"errors"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
)

func TestV2LocatorContract(t *testing.T) {
	hash := strings.Repeat("ab", 32)
	if err := ValidateV2LocatorRequest(V2LocatorRequest{
		Locator: []string{hash},
	}); err != nil {
		t.Fatal(err)
	}

	if err := ValidateV2LocatorRequest(V2LocatorRequest{}); !errors.Is(err, ErrV2InvalidLocator) {
		t.Fatalf("empty locator error=%v", err)
	}
	if err := ValidateV2LocatorRequest(V2LocatorRequest{
		Locator: []string{strings.ToUpper(hash)},
	}); !errors.Is(err, ErrV2InvalidLocator) {
		t.Fatalf("uppercase locator error=%v", err)
	}
	if err := ValidateV2LocatorRequest(V2LocatorRequest{
		Locator: []string{hash, hash},
	}); !errors.Is(err, ErrV2InvalidLocator) {
		t.Fatalf("duplicate locator error=%v", err)
	}

	tooMany := make([]string, V2MaxLocatorEntries+1)
	for i := range tooMany {
		tooMany[i] = strings.Repeat("0", 62) + strings.ToLower(hexByte(i))
	}
	if err := ValidateV2LocatorRequest(V2LocatorRequest{
		Locator: tooMany,
	}); !errors.Is(err, ErrV2InvalidLocator) {
		t.Fatalf("oversized locator error=%v", err)
	}
}

func TestV2HeadersContract(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}
	genesis, err := block.NewGenesisForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	second, err := block.NewV2(
		1,
		genesis.BlockHash,
		genesis.Timestamp+60,
		genesis.Target,
		1,
		nil,
		profile.ChainID,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	payload := V2HeadersPayload{
		Headers: []block.Header{genesis.Header(), second.Header()},
	}
	if err := ValidateV2HeadersPayload(payload); err != nil {
		t.Fatal(err)
	}

	broken := payload
	broken.Headers = append([]block.Header(nil), payload.Headers...)
	broken.Headers[1].PreviousBlockHash = strings.Repeat("11", 32)
	broken.Headers[1].BlockHash = broken.Headers[1].CalculateHash()
	if err := ValidateV2HeadersPayload(broken); !errors.Is(err, ErrV2InvalidHeaders) {
		t.Fatalf("broken continuity error=%v", err)
	}

	tampered := payload
	tampered.Headers = append([]block.Header(nil), payload.Headers...)
	tampered.Headers[1].BlockHash = strings.Repeat("22", 32)
	if err := ValidateV2HeadersPayload(tampered); !errors.Is(err, ErrV2InvalidHeaders) {
		t.Fatalf("tampered hash error=%v", err)
	}
}

func TestV2InventoryContract(t *testing.T) {
	a := strings.Repeat("aa", 32)
	b := strings.Repeat("bb", 32)

	if err := ValidateV2GetDataPayload(V2GetDataPayload{
		Items: []V2InventoryItem{{Kind: V2InventoryBlock, Hash: a}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateV2GetDataPayload(V2GetDataPayload{}); !errors.Is(err, ErrV2InvalidInventory) {
		t.Fatalf("empty get_data error=%v", err)
	}
	if err := ValidateV2GetDataPayload(V2GetDataPayload{
		Items: []V2InventoryItem{
			{Kind: V2InventoryBlock, Hash: a},
			{Kind: V2InventoryBlock, Hash: a},
		},
	}); !errors.Is(err, ErrV2InvalidInventory) {
		t.Fatalf("duplicate inventory error=%v", err)
	}
	if err := ValidateV2GetDataPayload(V2GetDataPayload{
		Items: []V2InventoryItem{{Kind: "tx", Hash: b}},
	}); !errors.Is(err, ErrV2InvalidInventory) {
		t.Fatalf("wrong inventory kind error=%v", err)
	}
	if err := ValidateV2InvPayload(V2InvPayload{}); err != nil {
		t.Fatalf("empty inv should be valid: %v", err)
	}
}

func hexByte(value int) string {
	const digits = "0123456789abcdef"
	value &= 0xff
	return string([]byte{digits[value>>4], digits[value&0x0f]})
}
