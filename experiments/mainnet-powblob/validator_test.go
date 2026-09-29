package powblob

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/core/block"
)

const mainnetGoldenRandomXHash = "85c60db39d4d13caeeb2e3841da2ef26d2ce64c9659a48f843811ba6ea8560f2"

func blockWithTarget(t *testing.T, target string) *block.Block {
	t.Helper()
	b, err := block.NewV2(
		1,
		strings.Repeat("00", 32),
		1800000000,
		target,
		1,
		nil,
		"valdr-mainnet-1",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestMainnetGoldenRandomXHashTargetComparison(t *testing.T) {
	raw, err := hex.DecodeString(mainnetGoldenRandomXHash)
	if err != nil {
		t.Fatal(err)
	}

	// A target equal to the hash must be valid.
	if err := ValidatePoWHash(blockWithTarget(t, mainnetGoldenRandomXHash), raw); err != nil {
		t.Fatalf("equal target rejected: %v", err)
	}

	// 0xff..ff is easier than the golden hash and must pass.
	if err := ValidatePoWHash(blockWithTarget(t, strings.Repeat("ff", 32)), raw); err != nil {
		t.Fatalf("easy target rejected: %v", err)
	}

	// 0x01..01 is harder than the golden hash and must fail.
	err = ValidatePoWHash(blockWithTarget(t, strings.Repeat("01", 32)), raw)
	if !errors.Is(err, ErrTargetNotMet) {
		t.Fatalf("hard target error=%v want ErrTargetNotMet", err)
	}
}

func TestValidatePoWHashRejectsMalformedInput(t *testing.T) {
	b := blockWithTarget(t, strings.Repeat("ff", 32))

	if err := ValidatePoWHash(nil, make([]byte, 32)); !errors.Is(err, ErrInvalidPoWHash) {
		t.Fatalf("nil block error=%v", err)
	}
	if err := ValidatePoWHash(b, make([]byte, 31)); !errors.Is(err, ErrInvalidPoWHash) {
		t.Fatalf("short hash error=%v", err)
	}

	b.Target = strings.Repeat("00", 32)
	if err := ValidatePoWHash(b, make([]byte, 32)); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("zero target error=%v", err)
	}
}
