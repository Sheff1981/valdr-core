package consensus

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/core/block"
)

type fixedRandomXHasher struct {
	hash []byte
	err  error
	key  []byte
	blob []byte
}

func (h *fixedRandomXHasher) Hash(key, input []byte) ([]byte, error) {
	h.key = append([]byte(nil), key...)
	h.blob = append([]byte(nil), input...)
	if h.err != nil {
		return nil, h.err
	}
	return append([]byte(nil), h.hash...), nil
}

func mainnetCandidateBlock(t *testing.T, target string) *block.Block {
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

func TestMainnetPoWBlobV1Golden(t *testing.T) {
	b := mainnetCandidateBlock(t, strings.Repeat("40", 32))
	blob, err := MainnetPoWBlobV1(b)
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

func TestValidateMainnetRandomXPoWGoldenHash(t *testing.T) {
	const golden = "85c60db39d4d13caeeb2e3841da2ef26d2ce64c9659a48f843811ba6ea8560f2"
	raw, err := hex.DecodeString(golden)
	if err != nil {
		t.Fatal(err)
	}

	seed, err := hex.DecodeString("0fca8879c33460a767246a02b156794ec9958113261e04db0b3a9e66fe6febb9")
	if err != nil {
		t.Fatal(err)
	}

	h := &fixedRandomXHasher{hash: raw}
	b := mainnetCandidateBlock(t, golden) // equal target must pass
	got, err := ValidateMainnetRandomXPoW(b, seed, h)
	if err != nil {
		t.Fatalf("golden hash rejected: %v", err)
	}
	if hex.EncodeToString(got) != golden {
		t.Fatalf("hash=%x want=%s", got, golden)
	}
	if !strings.HasPrefix(string(h.blob), "VALDR/RANDOMX/POW/V1\x00") {
		t.Fatal("missing Mainnet PoW domain")
	}
	if hex.EncodeToString(h.key) != hex.EncodeToString(seed) {
		t.Fatal("seed key changed at hasher boundary")
	}
}

func TestValidateMainnetRandomXPoWRejectsTargetMiss(t *testing.T) {
	raw, _ := hex.DecodeString("85c60db39d4d13caeeb2e3841da2ef26d2ce64c9659a48f843811ba6ea8560f2")
	h := &fixedRandomXHasher{hash: raw}
	b := mainnetCandidateBlock(t, strings.Repeat("01", 32))
	_, err := ValidateMainnetRandomXPoW(b, []byte("seed"), h)
	if !errors.Is(err, ErrMainnetPoWTargetNotMet) {
		t.Fatalf("error=%v want target miss", err)
	}
}

func TestValidateMainnetRandomXPoWRejectsBadHasherOutput(t *testing.T) {
	h := &fixedRandomXHasher{hash: make([]byte, 31)}
	b := mainnetCandidateBlock(t, strings.Repeat("ff", 32))
	_, err := ValidateMainnetRandomXPoW(b, []byte("seed"), h)
	if !errors.Is(err, ErrMainnetPoWHashSize) {
		t.Fatalf("error=%v want hash size", err)
	}
}
