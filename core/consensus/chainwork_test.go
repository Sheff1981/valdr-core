package consensus

import (
	"math/big"
	"testing"
)

func TestBlockWorkVectors(t *testing.T) {
	for difficulty, want := range map[uint64]int64{
		1: 4096,
		2: 8192,
		4: 16384,
		16: 65536,
	} {
		got, err := BlockWork(difficulty)
		if err != nil {
			t.Fatalf("difficulty %d: %v", difficulty, err)
		}
		if got.Cmp(big.NewInt(want)) != 0 {
			t.Fatalf("difficulty %d work=%s want=%d", difficulty, got, want)
		}
	}

	first, err := BlockWork(1)
	if err != nil {
		t.Fatal(err)
	}
	total, err := AddWork(first, 4)
	if err != nil {
		t.Fatal(err)
	}
	if total.Cmp(big.NewInt(20480)) != 0 {
		t.Fatalf("cumulative work=%s want=20480", total)
	}
	if got := ChainworkHex(total); got != "0000000000000000000000000000000000000000000000000000000000005000" {
		t.Fatalf("chainwork hex=%s", got)
	}
}
