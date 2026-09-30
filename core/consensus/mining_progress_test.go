package consensus

import (
	"errors"
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
		func(hashes uint64) error {
			if hashes < reported {
				t.Fatalf("progress regressed: got=%d previous=%d", hashes, reported)
			}
			reported = hashes
			return nil
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

func TestMineTargetWithProgressCanCancelLongAttempt(t *testing.T) {
	target := big.NewInt(1)
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
		"cancel-test",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	stop := errors.New("stop mining")
	err = MineTargetWithProgress(
		candidate,
		target,
		func(hashes uint64) error {
			if hashes == 0 {
				t.Fatal("cancellation callback saw zero hashes")
			}
			return stop
		},
	)
	if !errors.Is(err, stop) {
		t.Fatalf("MineTargetWithProgress error=%v want=%v", err, stop)
	}
}
