package consensus

import (
	"math/big"
	"strings"
	"testing"
)

func daaTarget(t *testing.T, hexValue string) *big.Int {
	t.Helper()
	n, ok := new(big.Int).SetString(hexValue, 16)
	if !ok {
		t.Fatal("bad test target")
	}
	return n
}

func daaHistory(count int, interval int64, target *big.Int) []MainnetDAAHeader {
	out := make([]MainnetDAAHeader, 0, count)
	ts := int64(1800000000)
	for i := 0; i < count; i++ {
		out = append(out, MainnetDAAHeader{
			Height:    uint64(i),
			Timestamp: ts,
			Target:    new(big.Int).Set(target),
		})
		ts += interval
	}
	return out
}

func TestMainnetLWMAStableGoldenVector(t *testing.T) {
	initial := daaTarget(t, "0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	powLimit := daaTarget(t, "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	history := daaHistory(31, 600, initial)

	got, err := MainnetNextTargetLWMA(history, initial, powLimit)
	if err != nil {
		t.Fatal(err)
	}
	const want = "00010296a1ff1dbc3240c5fb540752c4167997946c559b5e2135180da2f18af3"
	if got.Text(16) != strings.TrimLeft(want, "0") {
		t.Fatalf("target=%064x want=%s", got, want)
	}
}

func TestMainnetLWMASimulatedHashrateSteps(t *testing.T) {
	initial := daaTarget(t, "0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	powLimit := daaTarget(t, "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	stable, err := MainnetNextTargetLWMA(daaHistory(31, 600, initial), initial, powLimit)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		interval int64
		faster   bool
	}{
		{"2x increase", 300, true},
		{"10x increase", 60, true},
		{"100x increase", 6, true},
		{"2x decrease", 1200, false},
		{"10x decrease", 6000, false},
		{"100x decrease", 60000, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := MainnetNextTargetLWMA(
				daaHistory(31, tc.interval, initial),
				initial,
				powLimit,
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.Cmp(powLimit) > 0 {
				t.Fatalf("target exceeds PowLimit: %064x > %064x", got, powLimit)
			}
			if tc.faster {
				if got.Cmp(stable) >= 0 {
					t.Fatalf("faster blocks must harden target: got=%064x stable=%064x", got, stable)
				}
			} else {
				if got.Cmp(stable) <= 0 {
					t.Fatalf("slower blocks must ease target: got=%064x stable=%064x", got, stable)
				}
			}
		})
	}
}

func TestMainnetLWMAStartupUsesInitialWork(t *testing.T) {
	initial := daaTarget(t, "0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	powLimit := daaTarget(t, "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	gotEmpty, err := MainnetNextTargetLWMA(nil, initial, powLimit)
	if err != nil {
		t.Fatal(err)
	}
	gotOne, err := MainnetNextTargetLWMA(
		[]MainnetDAAHeader{{Height: 0, Timestamp: 1800000000, Target: new(big.Int).Set(initial)}},
		initial,
		powLimit,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gotEmpty.Cmp(gotOne) != 0 {
		t.Fatalf("startup synthetic slots disagree: empty=%064x one=%064x", gotEmpty, gotOne)
	}
}

func TestMainnetLWMAOutOfOrderAndSolveClamp(t *testing.T) {
	initial := daaTarget(t, "0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	powLimit := daaTarget(t, "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	history := daaHistory(31, 600, initial)
	history[29].Timestamp = history[28].Timestamp - 100
	history[30].Timestamp = history[29].Timestamp + 100000

	got, err := MainnetNextTargetLWMA(history, initial, powLimit)
	if err != nil {
		t.Fatal(err)
	}
	if got.Sign() <= 0 || got.Cmp(powLimit) > 0 {
		t.Fatalf("bad clamped target=%064x", got)
	}
}

func TestValidateMainnetTimestampCandidateMTP11AndFuture30m(t *testing.T) {
	target := daaTarget(t, "0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	history := daaHistory(11, 600, target)
	local := history[len(history)-1].Timestamp + 60

	values := make([]int64, 11)
	for i := range history {
		values[i] = history[i].Timestamp
	}
	median := values[5]

	if err := ValidateMainnetTimestampCandidate(history, median, local); err == nil {
		t.Fatal("timestamp equal to MTP must fail")
	}
	if err := ValidateMainnetTimestampCandidate(history, median+1, local); err != nil {
		t.Fatalf("timestamp above MTP rejected: %v", err)
	}
	if err := ValidateMainnetTimestampCandidate(history, local+1801, local); err == nil {
		t.Fatal("timestamp beyond +30m must fail")
	}
}


func TestMainnetLWMARejectsDerivedZeroWork(t *testing.T) {
	maxTarget := daaTarget(t, "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	history := daaHistory(31, MainnetDAAMaxSolveSeconds, maxTarget)

	_, err := MainnetNextTargetLWMA(history, maxTarget, maxTarget)
	if !errors.Is(err, ErrMainnetDAAHistory) {
		t.Fatalf("error=%v want ErrMainnetDAAHistory", err)
	}
}
