package mempool

import (
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/core/transaction"
)

func TestPolicyRejectsBelowMinimumRelayFee(t *testing.T) {
	tx := signedTestTransaction(t, strings.Repeat("41", 32), 0)
	pool := NewWithConfig(Config{MinRelayFeePerByte: 1})

	if err := pool.AddWithFee(tx, uint64(tx.SerializedSize()-1)); !errors.Is(err, ErrMinRelayFee) {
		t.Fatalf("AddWithFee error=%v want ErrMinRelayFee", err)
	}
	if err := pool.AddWithFee(tx, uint64(tx.SerializedSize())); err != nil {
		t.Fatalf("minimum fee transaction rejected: %v", err)
	}
}

func TestPolicyExpiresTransactions(t *testing.T) {
	now := time.Unix(2_000_000_000, 0)
	pool := NewWithConfig(Config{
		Expiry: time.Hour,
		Now:    func() time.Time { return now },
	})
	tx := signedTestTransaction(t, strings.Repeat("42", 32), 0)
	if err := pool.AddWithFee(tx, 1); err != nil {
		t.Fatal(err)
	}
	now = now.Add(59 * time.Minute)
	if pool.Len() != 1 {
		t.Fatal("transaction expired too early")
	}
	now = now.Add(time.Minute)
	if pool.Len() != 0 {
		t.Fatal("transaction did not expire at policy boundary")
	}
}

func TestPolicyRejectsUnconfirmedParentAndSecondSpend(t *testing.T) {
	pool := New()
	parent := signedTestTransaction(t, strings.Repeat("43", 32), 0)
	if err := pool.AddWithFee(parent, 1); err != nil {
		t.Fatal(err)
	}

	child := signedTestTransaction(t, parent.TransactionID, 0)
	if err := pool.AddWithFee(child, 1); !errors.Is(err, ErrUnconfirmedParent) {
		t.Fatalf("child error=%v want ErrUnconfirmedParent", err)
	}

	conflict := signedTestTransaction(t, strings.Repeat("43", 32), 0)
	if err := pool.AddWithFee(conflict, 2); !errors.Is(err, ErrInputConflict) {
		t.Fatalf("conflict error=%v want ErrInputConflict", err)
	}
}

func TestPolicyEvictsLowestFeeRateThenOldest(t *testing.T) {
	now := time.Unix(2_000_000_100, 0)
	low := signedTestTransaction(t, strings.Repeat("44", 32), 0)
	high := signedTestTransaction(t, strings.Repeat("45", 32), 0)
	higher := signedTestTransaction(t, strings.Repeat("46", 32), 0)

	pool := NewWithConfig(Config{
		MaxBytes: high.SerializedSize() + higher.SerializedSize(),
		Now:      func() time.Time { return now },
	})
	if err := pool.AddWithFee(low, uint64(low.SerializedSize())); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	if err := pool.AddWithFee(high, uint64(high.SerializedSize()*2)); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	if err := pool.AddWithFee(higher, uint64(higher.SerializedSize()*3)); err != nil {
		t.Fatal(err)
	}
	if pool.Contains(low.TransactionID) {
		t.Fatal("lowest fee-rate transaction was not evicted")
	}
	if !pool.Contains(high.TransactionID) || !pool.Contains(higher.TransactionID) {
		t.Fatal("higher fee-rate transactions were incorrectly evicted")
	}
	if pool.Bytes() > high.SerializedSize()+higher.SerializedSize() {
		t.Fatalf("mempool bytes=%d exceed configured cap", pool.Bytes())
	}

	oldest := signedTestTransaction(t, strings.Repeat("47", 32), 0)
	newer := signedTestTransaction(t, strings.Repeat("48", 32), 0)
	newest := signedTestTransaction(t, strings.Repeat("49", 32), 0)
	now = time.Unix(2_000_001_000, 0)
	pool = NewWithConfig(Config{
		MaxBytes: newer.SerializedSize() + newest.SerializedSize(),
		Now:      func() time.Time { return now },
	})
	if err := pool.AddWithFee(oldest, uint64(oldest.SerializedSize())); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	if err := pool.AddWithFee(newer, uint64(newer.SerializedSize())); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	if err := pool.AddWithFee(newest, uint64(newest.SerializedSize())); err != nil {
		t.Fatal(err)
	}
	if pool.Contains(oldest.TransactionID) {
		t.Fatal("oldest equal-fee-rate transaction was not evicted")
	}
}

func TestMiningOrderFeeRateDescendingThenTxID(t *testing.T) {
	pool := New()
	low := signedTestTransaction(t, strings.Repeat("51", 32), 0)
	tieA := signedTestTransaction(t, strings.Repeat("52", 32), 0)
	tieB := signedTestTransaction(t, strings.Repeat("53", 32), 0)

	if err := pool.AddWithFee(low, uint64(low.SerializedSize()*2)); err != nil {
		t.Fatal(err)
	}
	if err := pool.AddWithFee(tieA, uint64(tieA.SerializedSize()*5)); err != nil {
		t.Fatal(err)
	}
	if err := pool.AddWithFee(tieB, uint64(tieB.SerializedSize()*5)); err != nil {
		t.Fatal(err)
	}

	ties := []string{tieA.TransactionID, tieB.TransactionID}
	sort.Strings(ties)
	got := pool.MiningTransactions()
	if len(got) != 3 {
		t.Fatalf("mining transaction count=%d want=3", len(got))
	}
	if got[0].TransactionID != ties[0] ||
		got[1].TransactionID != ties[1] ||
		got[2].TransactionID != low.TransactionID {
		t.Fatalf(
			"mining order=%s,%s,%s",
			got[0].TransactionID,
			got[1].TransactionID,
			got[2].TransactionID,
		)
	}
}

func TestRevalidateRemovesInvalidAndRefreshesFees(t *testing.T) {
	pool := NewWithConfig(Config{MinRelayFeePerByte: 1})
	first := signedTestTransaction(t, strings.Repeat("54", 32), 0)
	second := signedTestTransaction(t, strings.Repeat("55", 32), 0)
	if err := pool.AddWithFee(first, uint64(first.SerializedSize())); err != nil {
		t.Fatal(err)
	}
	if err := pool.AddWithFee(second, uint64(second.SerializedSize())); err != nil {
		t.Fatal(err)
	}

	removed := pool.Revalidate(func(tx *transaction.Transaction) (uint64, error) {
		if tx.TransactionID == first.TransactionID {
			return 0, errors.New("spent on active chain")
		}
		return uint64(tx.SerializedSize() * 2), nil
	})
	if removed != 1 || pool.Contains(first.TransactionID) || !pool.Contains(second.TransactionID) {
		t.Fatalf("revalidation removed=%d first=%t second=%t", removed, pool.Contains(first.TransactionID), pool.Contains(second.TransactionID))
	}
}
