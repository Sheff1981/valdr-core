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

func buildMainnetCandidateBlock(t *testing.T, target string) *block.Block {
	t.Helper()
	b, err := block.NewV2(
		262800,
		strings.Repeat("0", 64),
		1800000000,
		target,
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
	return b
}

func epoch0Seed(t *testing.T) []byte {
	t.Helper()
	seed, err := hex.DecodeString("0fca8879c33460a767246a02b156794ec9958113261e04db0b3a9e66fe6febb9")
	if err != nil {
		t.Fatal(err)
	}
	return seed
}

// This reproduces the cross-platform C++ prototype golden vector exactly.
// The target is part of the mining blob, so changing the target necessarily
// changes the RandomX output.
func TestNativeRandomXMainnetGoldenVector(t *testing.T) {
	b := buildMainnetCandidateBlock(t, strings.Repeat("40", 32))

	blob, err := consensus.MainnetPoWBlobV1(b)
	if err != nil {
		t.Fatal(err)
	}

	h, err := randomxnative.New()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	got, err := h.Hash(epoch0Seed(t), blob)
	if err != nil {
		t.Fatal(err)
	}

	const want = "85c60db39d4d13caeeb2e3841da2ef26d2ce64c9659a48f843811ba6ea8560f2"
	if hex.EncodeToString(got) != want {
		t.Fatalf("hash=%x want=%s", got, want)
	}
}

// This exercises the production candidate validator itself with an easy target.
// Because target is consensus data inside the mining blob, this vector has a
// different RandomX hash from the target=0x40 serialization vector above.
func TestNativeRandomXMainnetConsensusValidation(t *testing.T) {
	b := buildMainnetCandidateBlock(t, strings.Repeat("ff", 32))

	h, err := randomxnative.New()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	got, err := consensus.ValidateMainnetRandomXPoW(b, epoch0Seed(t), h)
	if err != nil {
		t.Fatal(err)
	}

	const want = "af8163cbce14b58222df29283ac440e7c3ada3b5b11c224b9466557ffc45a50d"
	if hex.EncodeToString(got) != want {
		t.Fatalf("hash=%x want=%s", got, want)
	}
}
