package main

import "testing"

func TestBuildCandidateUserLaptopBaseline(t *testing.T) {
	c, err := buildCandidate("418.723", 600, 1, 1, 4)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.InitialWork.String(); got != "251233" {
		t.Fatalf("InitialWork=%s want=251233", got)
	}
	if got := c.PowLimitWork.String(); got != "62808" {
		t.Fatalf("PowLimitWork=%s want=62808", got)
	}
	if got := hex256(c.InitialTarget); got != "000042c78dcd2f09f6673e99417b7b7d0dbb0ddcf732f964dc73fce1c6b50768" {
		t.Fatalf("InitialTarget=%s", got)
	}
	if got := hex256(c.PowLimitTarget); got != "00010b1e7ce2d1034b183a2be3b26504779a86ab0ed6197fb90be6d3c07b200c" {
		t.Fatalf("PowLimitTarget=%s", got)
	}
	if c.PowLimitTarget.Cmp(c.InitialTarget) <= 0 {
		t.Fatal("PowLimit should be easier (larger target)")
	}
}

func TestBuildCandidateConservativeMatrix(t *testing.T) {
	for _, tc := range []struct {
		num, den, factor int64
		work string
	}{
		{1, 1, 2, "251233"},
		{3, 4, 4, "188425"},
		{1, 2, 8, "125616"},
	} {
		c, err := buildCandidate("418.723", 600, tc.num, tc.den, tc.factor)
		if err != nil {
			t.Fatal(err)
		}
		if c.InitialWork.String() != tc.work {
			t.Fatalf("%d/%d work=%s want=%s", tc.num, tc.den, c.InitialWork, tc.work)
		}
	}
}

func TestBuildCandidateRejectsInvalid(t *testing.T) {
	for _, tc := range []struct {
		h string
		s, n, d, p int64
	}{
		{"", 600, 1, 1, 4},
		{"0", 600, 1, 1, 4},
		{"100", 0, 1, 1, 4},
		{"100", 600, 0, 1, 4},
		{"100", 600, 1, 0, 4},
		{"100", 600, 1, 1, 0},
	} {
		if _, err := buildCandidate(tc.h, tc.s, tc.n, tc.d, tc.p); err == nil {
			t.Fatalf("expected error for %+v", tc)
		}
	}
}

func hex256(v interface{ Text(int) string }) string {
	s := v.Text(16)
	for len(s) < 64 {
		s = "0" + s
	}
	return s
}
