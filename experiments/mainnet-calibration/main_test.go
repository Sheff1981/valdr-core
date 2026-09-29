package main

import "testing"

func TestCalibrationTargetExactInteger(t *testing.T) {
	target, work, err := calibrationTarget("100", 600, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if work.String() != "60000" {
		t.Fatalf("work=%s want=60000", work)
	}
	const want = "0001179ec9cbd821dc3a6faf2c19ab138636571bb75d40948c5b344ad1fcff0a"
	if got := fmtTarget(target); got != want {
		t.Fatalf("target=%s want=%s", got, want)
	}
}

func TestCalibrationTargetDecimalAndExplicitSafety(t *testing.T) {
	target, work, err := calibrationTarget("68.2954", 600, 9, 10)
	if err != nil {
		t.Fatal(err)
	}
	if work.String() != "36879" {
		t.Fatalf("work=%s want=36879", work)
	}
	if target.Sign() <= 0 || target.BitLen() > 256 {
		t.Fatal("invalid target")
	}
}

func TestCalibrationTargetRejectsInvalidInputs(t *testing.T) {
	for _, tc := range []struct {
		rate string
		sec  int64
		num  int64
		den  int64
	}{
		{"", 600, 1, 1},
		{"0", 600, 1, 1},
		{"100", 0, 1, 1},
		{"100", 600, 0, 1},
		{"100", 600, 1, 0},
	} {
		if _, _, err := calibrationTarget(tc.rate, tc.sec, tc.num, tc.den); err == nil {
			t.Fatalf("expected error for %+v", tc)
		}
	}
}

func fmtTarget(v interface{ Text(int) string }) string {
	s := v.Text(16)
	for len(s) < 64 {
		s = "0" + s
	}
	return s
}
