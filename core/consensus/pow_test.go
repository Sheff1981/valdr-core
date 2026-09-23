package consensus

import (
	"errors"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
)

func TestTargetShrinksAsDifficultyIncreases(t *testing.T) {
	easy, err := TargetForDifficulty(1)
	if err != nil {
		t.Fatal(err)
	}
	harder, err := TargetForDifficulty(2)
	if err != nil {
		t.Fatal(err)
	}
	if harder.Cmp(easy) >= 0 {
		t.Fatal("difficulty 2 target must be smaller than difficulty 1 target")
	}
}

func TestTargetHexHas256Bits(t *testing.T) {
	target, err := TargetHex(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(target) != 64 {
		t.Fatalf("target hex length = %d, want 64", len(target))
	}
	if !strings.HasPrefix(target, "000") {
		t.Fatalf("target = %s, want devnet 12-bit PoW limit", target)
	}
}

func TestMineAndValidatePoW(t *testing.T) {
	b := block.New(
		1,
		block.NewGenesis().BlockHash,
		config.GenesisTimestamp+60,
		1,
		0,
		nil,
		"",
	)
	if err := Mine(b); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePoW(b); err != nil {
		t.Fatal(err)
	}
}

func TestRejectsHashAboveTarget(t *testing.T) {
	b := block.New(1, "previous", config.GenesisTimestamp+60, 1, 0, nil, "")
	b.BlockHash = strings.Repeat("f", 64)

	err := ValidatePoW(b)
	if !errors.Is(err, ErrInvalidPoW) {
		t.Fatalf("ValidatePoW error = %v, want ErrInvalidPoW", err)
	}
}

func TestNextDifficulty(t *testing.T) {
	if got := NextDifficulty(1, 1000, 1060); got != 1 {
		t.Fatalf("60-second interval difficulty = %d, want 1", got)
	}
	if got := NextDifficulty(1, 1000, 1030); got != 2 {
		t.Fatalf("30-second interval difficulty = %d, want 2", got)
	}
	if got := NextDifficulty(8, 1000, 1120); got != 4 {
		t.Fatalf("120-second interval difficulty = %d, want 4", got)
	}
	if got := NextDifficulty(8, 1000, 1001); got != 32 {
		t.Fatalf("very fast interval difficulty = %d, want clamp at 32", got)
	}
}
