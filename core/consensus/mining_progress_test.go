package consensus

import (
	"math/big"
	"testing"

	"github.com/Sheff1981/valdr-core/core/block"
)

func TestMineTargetWithProgressReportsFinalHashCount(t *testing.T) {
	target := new(big.Int).Sub(
		new(big.Int).Lsh(big.NewInt(1), 256),
		big.NewInt(1),
	)
	targetHex, err := TargetHexV2(target)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := block.NewV2(
		1,
		"",
		1,
		targetHex,
		0,
		nil,
		"progress-test",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	var reported uint64
	if err := MineTargetWithProgress(
		candidate,
		target,
		func(hashes uint64) {
			if hashes < reported {
				t.Fatalf("progress regressed: got=%d previous=%d", hashes, reported)
			}
			reported = hashes
		},
	); err != nil {
		t.Fatal(err)
	}

	want := candidate.Nonce + 1
	if reported != want {
		t.Fatalf("reported hashes=%d want=%d", reported, want)
	}
	if reported == 0 {
		t.Fatal("progress callback did not report work")
	}
}
