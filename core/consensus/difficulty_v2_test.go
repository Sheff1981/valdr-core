package consensus

import (
	"errors"
	"math/big"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
)

func TestDifficultyV2GoldenRetargetVectors(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}

	powLimit, err := PowLimitForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	assertTargetHex(
		t,
		powLimit,
		"000fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	)

	oldTarget := mustTargetHex(
		t,
		"0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	)
	tests := []struct {
		name     string
		actual   int64
		wantHex  string
	}{
		{
			name:    "fast_clamped_quarter",
			actual:  1,
			wantHex: "00003fffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			name:    "exact_target_timespan",
			actual:  3600,
			wantHex: "0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			name:    "slow_clamped_four_x",
			actual:  1_000_000,
			wantHex: "0003fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, err := RetargetV2(oldTarget, tt.actual, profile)
			if err != nil {
				t.Fatal(err)
			}
			assertTargetHex(t, next, tt.wantHex)
		})
	}

	bounded, err := RetargetV2(powLimit, 14_400, profile)
	if err != nil {
		t.Fatal(err)
	}
	if bounded.Cmp(powLimit) != 0 {
		t.Fatalf("pow-limit bound=%064x want=%064x", bounded, powLimit)
	}
}

func TestDifficultyV2BoundaryUsesPreviousWindow(t *testing.T) {
	profile, _ := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	target := mustTargetHex(
		t,
		"0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	)
	history := make([]V2DifficultyHeader, 60)
	for i := range history {
		history[i] = V2DifficultyHeader{
			Height:    uint64(i),
			Timestamp: 1_000 + int64(i)*60,
			Target:    new(big.Int).Set(target),
		}
	}

	next, special, err := NextTargetV2(
		history,
		history[len(history)-1].Timestamp+60,
		profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	if special {
		t.Fatal("devnet retarget unexpectedly marked min-difficulty")
	}
	assertTargetHex(
		t,
		next,
		"0000fbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbba",
	)
}

func TestTimestampV2MedianTimePastAndFutureLimit(t *testing.T) {
	profile, _ := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	target, _ := PowLimitForProfile(profile)
	history := make([]V2DifficultyHeader, 11)
	for i := range history {
		history[i] = V2DifficultyHeader{
			Height:    uint64(i),
			Timestamp: 100 + int64(i),
			Target:    new(big.Int).Set(target),
		}
	}

	median, err := MedianTimePastV2(history, profile.MedianTimePastWindow)
	if err != nil {
		t.Fatal(err)
	}
	if median != 105 {
		t.Fatalf("MTP=%d want=105", median)
	}

	if err := ValidateTimestampV2(history, 105, 1_000, profile); !errors.Is(err, ErrTimestampMedianPast) {
		t.Fatalf("timestamp error=%v want ErrTimestampMedianPast", err)
	}
	if err := ValidateTimestampV2(history, 106, 1_000, profile); err != nil {
		t.Fatalf("valid timestamp rejected: %v", err)
	}
	if err := ValidateTimestampV2(
		history,
		1_000+profile.MaxFutureBlockSeconds+1,
		1_000,
		profile,
	); !errors.Is(err, ErrTimestampTooFarFuture) {
		t.Fatalf("future timestamp error=%v want ErrTimestampTooFarFuture", err)
	}
}

func TestTestnetMinDifficultyEscapeAndRecovery(t *testing.T) {
	profile, _ := config.ResolveNetworkProfile(config.NetworkTestnetV02)
	normal := mustTargetHex(
		t,
		"0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	)
	powLimit, err := PowLimitForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}

	history := []V2DifficultyHeader{
		{Height: 0, Timestamp: 1_000, Target: new(big.Int).Set(normal)},
		{Height: 1, Timestamp: 1_060, Target: new(big.Int).Set(normal)},
	}
	escaped, special, err := NextTargetV2(history, 1_661, profile)
	if err != nil {
		t.Fatal(err)
	}
	if !special || escaped.Cmp(powLimit) != 0 {
		t.Fatalf("escape target=%064x special=%t", escaped, special)
	}

	history = append(history, V2DifficultyHeader{
		Height:               2,
		Timestamp:            1_661,
		Target:               new(big.Int).Set(escaped),
		SpecialMinDifficulty: true,
	})
	recovered, special, err := NextTargetV2(history, 1_720, profile)
	if err != nil {
		t.Fatal(err)
	}
	if special || recovered.Cmp(normal) != 0 {
		t.Fatalf("recovery target=%064x special=%t", recovered, special)
	}
}

func TestTestnetRetargetIgnoresSpecialEscapeAsOldTarget(t *testing.T) {
	profile, _ := config.ResolveNetworkProfile(config.NetworkTestnetV02)
	normal := mustTargetHex(
		t,
		"0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	)
	powLimit, _ := PowLimitForProfile(profile)

	history := make([]V2DifficultyHeader, 60)
	for i := range history {
		history[i] = V2DifficultyHeader{
			Height:    uint64(i),
			Timestamp: 2_000 + int64(i)*60,
			Target:    new(big.Int).Set(normal),
		}
	}
	history[59].Target = new(big.Int).Set(powLimit)
	history[59].SpecialMinDifficulty = true

	next, special, err := NextTargetV2(
		history,
		history[59].Timestamp+60,
		profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	if special {
		t.Fatal("retarget boundary cannot be special min-difficulty")
	}
	assertTargetHex(
		t,
		next,
		"0000fbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbba",
	)
}

func TestDifficultyV2RejectsLegacyProfile(t *testing.T) {
	legacy, _ := config.ResolveNetworkProfile(config.NetworkLegacyV01)
	target, _ := TargetForDifficulty(1)
	if _, err := RetargetV2(target, 3_600, legacy); !errors.Is(err, ErrInvalidDifficultyV2Config) {
		t.Fatalf("legacy retarget error=%v", err)
	}
}

func mustTargetHex(t *testing.T, value string) *big.Int {
	t.Helper()
	target := new(big.Int)
	if _, ok := target.SetString(value, 16); !ok {
		t.Fatalf("invalid test target %q", value)
	}
	return target
}

func assertTargetHex(t *testing.T, target *big.Int, want string) {
	t.Helper()
	got, err := TargetHexV2(target)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("target=%s want=%s", got, want)
	}
}

func TestTestnetV029BootstrapCalibrationAndRetarget(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	powLimit, err := PowLimitForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	assertTargetHex(
		t,
		powLimit,
		"000003ffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	)

	initial := mustTargetHex(t, config.TestnetV029GenesisTarget)
	two256 := new(big.Int).Lsh(big.NewInt(1), 256)
	expectedHashes := new(big.Int).Div(
		two256,
		new(big.Int).Add(initial, big.NewInt(1)),
	)
	// The frozen Testnet2 target was calibrated around 5.4M trials. Under the
	// consensus work formula's integer floor, its exact work is 5,399,999.
	if expectedHashes.Uint64() != 5_399_999 {
		t.Fatalf("bootstrap expected hashes=%s want=5399999", expectedHashes)
	}

	history := make([]V2DifficultyHeader, 10)
	for i := range history {
		history[i] = V2DifficultyHeader{
			Height:    uint64(i),
			Timestamp: 10_000 + int64(i)*15,
			Target:    new(big.Int).Set(initial),
		}
	}
	next, special, err := NextTargetV2(
		history,
		history[len(history)-1].Timestamp+15,
		profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	if special {
		t.Fatal("active Testnet retarget unexpectedly marked min-difficulty")
	}
	assertTargetHex(
		t,
		next,
		"000000c6d750ebfa67b90d1c384cdf0d90ba7649524980e3f7db468b95eaa887",
	)
}
