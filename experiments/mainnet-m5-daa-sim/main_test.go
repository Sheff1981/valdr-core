package main

import "testing"

func TestPowLimitCandidateComparison(t *testing.T) {
	down2 := scenario{"2x_down", 1, 2}
	r2, err := simulate(2, down2, 120)
	if err != nil { t.Fatal(err) }
	r4, err := simulate(4, down2, 120)
	if err != nil { t.Fatal(err) }
	r8, err := simulate(8, down2, 120)
	if err != nil { t.Fatal(err) }

	if r2.powLimitHits == 0 {
		t.Fatal("2x PowLimit must hit the cap after a sustained 2x hashrate decrease")
	}
	if r4.powLimitHits != 0 {
		t.Fatalf("4x PowLimit unexpectedly hit cap %d times after 2x decrease", r4.powLimitHits)
	}
	if r8.powLimitHits != 0 {
		t.Fatalf("8x PowLimit unexpectedly hit cap %d times after 2x decrease", r8.powLimitHits)
	}
	if got := r4.last30Avg.FloatString(3); got != "594.333" {
		t.Fatalf("4x/2x-down last30 avg=%s", got)
	}
}

func TestPowLimitSevereDropMatrix(t *testing.T) {
	down10 := scenario{"10x_down", 1, 10}
	want := map[int64]string{
		2: "3000.000",
		4: "1500.000",
		8: "750.000",
	}
	for _, factor := range []int64{2,4,8} {
		r, err := simulate(factor, down10, 120)
		if err != nil { t.Fatal(err) }
		if got := r.last30Avg.FloatString(3); got != want[factor] {
			t.Fatalf("%dx 10x-down avg=%s want=%s", factor, got, want[factor])
		}
		if r.powLimitHits == 0 {
			t.Fatalf("%dx should eventually hit PowLimit under 10x drop", factor)
		}
	}
}

func TestStableAndHashrateIncreaseDoNotHitPowLimit(t *testing.T) {
	for _, s := range []scenario{
		{"stable",1,1},
		{"2x_up",2,1},
		{"10x_up",10,1},
		{"100x_up",100,1},
	} {
		r, err := simulate(4, s, 120)
		if err != nil { t.Fatal(err) }
		if r.powLimitHits != 0 {
			t.Fatalf("%s hit PowLimit %d times", s.name, r.powLimitHits)
		}
	}
}
