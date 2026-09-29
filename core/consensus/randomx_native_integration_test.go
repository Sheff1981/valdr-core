//go:build randomx_native && cgo

package consensus_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus"
	"github.com/Sheff1981/valdr-core/core/consensus/randomxnative"
)

func TestNativeRandomXMainnetGoldenVector(t *testing.T) {
	b, err := block.NewV2(
		262800,
		strings.Repeat("0", 64),
		1800000000,
		strings.Repeat("ff", 32),
		0x0102030405060708,
		nil,
		"valdr-mainnet-1",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	b.MerkleRoot = strings.Repeat("20", 32)
	b.BlockHash = b.CalculateHash()

	seed, err := hex.DecodeString("0fca8879c33460a767246a02b156794ec9958113261e04db0b3a9e66fe6febb9")
	if err != nil {
		t.Fatal(err)
	}

	h, err := randomxnative.New()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	got, err := consensus.ValidateMainnetRandomXPoW(b, seed, h)
	if err != nil {
		t.Fatal(err)
	}
	const want = "85c60db39d4d13caeeb2e3841da2ef26d2ce64c9659a48f843811ba6ea8560f2"
	if hex.EncodeToString(got) != want {
		t.Fatalf("hash=%x want=%s", got, want)
	}
}
