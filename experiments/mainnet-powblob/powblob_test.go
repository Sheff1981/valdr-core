package powblob

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/Sheff1981/valdr-core/core/block"
)

func sampleBlock(t *testing.T) *block.Block {
	t.Helper()
	b, err := block.NewV2(
		262800,
		"0000000000000000000000000000000000000000000000000000000000000000",
		1800000000,
		"4040404040404040404040404040404040404040404040404040404040404040",
		0x0102030405060708,
		nil,
		"valdr-mainnet-1",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	// NewV2 computes the real empty Merkle root. Override it only for this fixed
	// serialization vector; HeaderBytesChecked uses the field as supplied.
	b.MerkleRoot = "2020202020202020202020202020202020202020202020202020202020202020"
	b.BlockHash = b.CalculateHash()
	return b
}

func TestBuildV1Golden(t *testing.T) {
	b := sampleBlock(t)
	blob, err := BuildV1(b)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(blob), 256; got != want {
		t.Fatalf("blob length=%d want=%d", got, want)
	}
	sum := sha256.Sum256(blob)
	if got, want := hex.EncodeToString(sum[:]), "474a5ca610d1573c3f2b0cd0d634448ee0a9a909af40ea0773b9b780ccbf29bf"; got != want {
		t.Fatalf("blob sha256=%s want=%s", got, want)
	}
}

func TestBlockIDGolden(t *testing.T) {
	got, err := BlockID(sampleBlock(t))
	if err != nil {
		t.Fatal(err)
	}
	if want := "9d7838b61a5e10c252d690149eec1a2fe80e32617fd29ebacaaf6d9903144a8a"; hex.EncodeToString(got[:]) != want {
		t.Fatalf("block id=%s want=%s", hex.EncodeToString(got[:]), want)
	}
}

func TestEpoch0SeedGolden(t *testing.T) {
	got := Epoch0Seed()
	if want := "0fca8879c33460a767246a02b156794ec9958113261e04db0b3a9e66fe6febb9"; hex.EncodeToString(got[:]) != want {
		t.Fatalf("epoch0 seed=%s want=%s", hex.EncodeToString(got[:]), want)
	}
}

func TestSeedHeightBoundaries(t *testing.T) {
	tests := []struct {
		height uint64
		want   uint64
		ok     bool
	}{
		{0, 0, false},
		{63, 0, false},
		{64, 0, true},
		{2111, 0, true},
		{2112, 2048, true},
		{4159, 2048, true},
		{4160, 4096, true},
	}
	for _, tc := range tests {
		got, ok := SeedHeight(tc.height)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("height=%d got=(%d,%v) want=(%d,%v)", tc.height, got, ok, tc.want, tc.ok)
		}
	}
}

func TestNonceChangesMiningBlobAndBlockID(t *testing.T) {
	a := sampleBlock(t)
	b := sampleBlock(t)
	b.Nonce++

	blobA, _ := BuildV1(a)
	blobB, _ := BuildV1(b)
	if string(blobA) == string(blobB) {
		t.Fatal("nonce did not change mining blob")
	}

	idA, _ := BlockID(a)
	idB, _ := BlockID(b)
	if idA == idB {
		t.Fatal("nonce did not change block id")
	}
}
