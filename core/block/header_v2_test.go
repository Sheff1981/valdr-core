package block

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func TestV2HeaderGoldenVector(t *testing.T) {
	const target = "000fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	candidate, err := NewV2(
		1,
		strings.Repeat("a", 64),
		1790121660,
		target,
		7,
		nil,
		"valdr-devnet-2",
		"v2",
	)
	if err != nil {
		t.Fatal(err)
	}

	const wantHeaderHex = "000000020000000000000001000000000000004061616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161000000000000004065336230633434323938666331633134396166626634633839393666623932343237616534316534363439623933346361343935393931623738353262383535000000006ab316bc000fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0000000000000007000000000000000e76616c64722d6465766e65742d3200000000000000027632"
	gotHeader, err := candidate.HeaderBytesChecked()
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(gotHeader); got != wantHeaderHex {
		t.Fatalf("v2 header=%s want=%s", got, wantHeaderHex)
	}
	const wantHash = "773c096448a93ad6a15de632d9f81c1957455c1a2df56a9e540745ea79cd26f5"
	if candidate.BlockHash != wantHash {
		t.Fatalf("v2 hash=%s want=%s", candidate.BlockHash, wantHash)
	}

	header := candidate.Header()
	if header.CalculateHash() != wantHash {
		t.Fatalf("header hash=%s want=%s", header.CalculateHash(), wantHash)
	}
}

func TestV2HeaderRejectsNonCanonicalTargetAndLegacyDifficulty(t *testing.T) {
	const valid = "000fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	if _, err := NewV2(1, "", 1, strings.ToUpper(valid), 0, nil, "valdr-devnet-2", ""); !errors.Is(err, ErrInvalidTargetEncoding) {
		t.Fatalf("uppercase target error=%v want ErrInvalidTargetEncoding", err)
	}

	candidate, err := NewV2(1, "", 1, valid, 0, nil, "valdr-devnet-2", "")
	if err != nil {
		t.Fatal(err)
	}
	candidate.Difficulty = 1
	if _, err := candidate.HeaderBytesChecked(); !errors.Is(err, ErrInvalidTargetEncoding) {
		t.Fatalf("v2 difficulty error=%v want ErrInvalidTargetEncoding", err)
	}
	if candidate.CalculateHash() != "" {
		t.Fatal("invalid v2 header produced a block hash")
	}
}
