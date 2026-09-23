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

func TestNextDifficultyUsesConfirmedWindow(t *testing.T) {
	if got := NextDifficulty(nil); got != config.GenesisDifficulty {
		t.Fatalf("empty history difficulty = %d, want %d", got, config.GenesisDifficulty)
	}

	stable := difficultyHistory(10, 8, 60)
	if got := NextDifficulty(stable); got != 8 {
		t.Fatalf("stable window difficulty = %d, want 8", got)
	}

	fast := difficultyHistory(10, 8, 15)
	if got := NextDifficulty(fast); got != 32 {
		t.Fatalf("fast window difficulty = %d, want clamp at 32", got)
	}

	slow := difficultyHistory(10, 8, 240)
	if got := NextDifficulty(slow); got != 2 {
		t.Fatalf("slow window difficulty = %d, want clamp at 2", got)
	}
}

func TestNextDifficultyHoldsBetweenWindowBoundaries(t *testing.T) {
	history := difficultyHistory(11, 32, 1)
	if got := NextDifficulty(history); got != 32 {
		t.Fatalf("non-boundary difficulty = %d, want 32", got)
	}
}

func TestNextDifficultyNeedsCompleteWindow(t *testing.T) {
	history := difficultyHistory(9, 7, 1)
	if got := NextDifficulty(history); got != 7 {
		t.Fatalf("short history difficulty = %d, want 7", got)
	}
}

func difficultyHistory(count int, difficulty uint64, interval int64) []*block.Block {
	history := make([]*block.Block, 0, count)
	for i := 0; i < count; i++ {
		history = append(history, &block.Block{
			Height:     uint64(i),
			Timestamp:  config.GenesisTimestamp + int64(i)*interval,
			Difficulty: difficulty,
		})
	}
	return history
}
