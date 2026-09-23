package consensus

import (
	"errors"
	"math/big"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
)

func TestValidateHeaderV2BeforeBody(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}
	target, err := PowLimitForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	targetHex, err := TargetHexV2(target)
	if err != nil {
		t.Fatal(err)
	}

	const parentHash = "1111111111111111111111111111111111111111111111111111111111111111"
	history := []V2DifficultyHeader{{
		Height:    0,
		BlockHash: parentHash,
		Timestamp: config.GenesisTimestamp,
		Target:    new(big.Int).Set(target),
	}}

	candidate, err := block.NewV2(
		1,
		parentHash,
		config.GenesisTimestamp+60,
		targetHex,
		0,
		nil,
		profile.ChainID,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := MineTarget(candidate, target); err != nil {
		t.Fatal(err)
	}

	special, err := ValidateHeaderV2(
		candidate.Header(),
		history,
		profile,
		config.GenesisTimestamp+60,
	)
	if err != nil {
		t.Fatalf("valid v2 header rejected: %v", err)
	}
	if special {
		t.Fatal("devnet header unexpectedly marked special min-difficulty")
	}

	badParent := candidate.Header()
	badParent.PreviousBlockHash = "2222222222222222222222222222222222222222222222222222222222222222"
	badParent.BlockHash = badParent.CalculateHash()
	if _, err := ValidateHeaderV2(
		badParent,
		history,
		profile,
		config.GenesisTimestamp+60,
	); !errors.Is(err, ErrInvalidDifficultyHistory) {
		t.Fatalf("bad parent error=%v want ErrInvalidDifficultyHistory", err)
	}

	badChain := candidate.Header()
	badChain.ChainID = "valdr-testnet-1"
	badChain.BlockHash = badChain.CalculateHash()
	if _, err := ValidateHeaderV2(
		badChain,
		history,
		profile,
		config.GenesisTimestamp+60,
	); !errors.Is(err, ErrHeaderWrongChainID) {
		t.Fatalf("bad chain error=%v want ErrHeaderWrongChainID", err)
	}

	badTarget := candidate.Header()
	easier := new(big.Int).Add(target, big.NewInt(1))
	badTarget.Target, _ = TargetHexV2(easier)
	badTarget.BlockHash = badTarget.CalculateHash()
	if _, err := ValidateHeaderV2(
		badTarget,
		history,
		profile,
		config.GenesisTimestamp+60,
	); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("above pow-limit target error=%v want ErrInvalidTarget", err)
	}
}

func TestExactTargetChainwork(t *testing.T) {
	profile, _ := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	target, err := PowLimitForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	work, err := BlockWorkTarget(target)
	if err != nil {
		t.Fatal(err)
	}
	if work.Sign() <= 0 {
		t.Fatal("exact-target work must be positive")
	}
	two, err := AddTargetWork(work, target)
	if err != nil {
		t.Fatal(err)
	}
	want := new(big.Int).Mul(work, big.NewInt(2))
	if two.Cmp(want) != 0 {
		t.Fatalf("chainwork=%s want=%s", two, want)
	}
}
