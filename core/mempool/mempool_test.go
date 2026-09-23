package mempool

import (
	"errors"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/core/transaction"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

func signedTestTransaction(t *testing.T, previousID string, outputIndex uint32) *transaction.Transaction {
	t.Helper()

	signer, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	recipientKey, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := valdrcrypto.AddressFromPublicKey(&recipientKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	tx := transaction.New(
		[]transaction.Input{{
			PreviousTransactionID: previousID,
			OutputIndex:           outputIndex,
		}},
		[]transaction.Output{{
			Amount:    100,
			Recipient: recipient,
		}},
		1790122200,
	)
	if err := tx.Sign(signer); err != nil {
		t.Fatal(err)
	}
	return tx
}

func TestMempoolAddRemoveAndDuplicate(t *testing.T) {
	pool := New()
	tx := signedTestTransaction(t, strings.Repeat("11", 32), 0)

	if err := pool.Add(tx); err != nil {
		t.Fatal(err)
	}
	if pool.Len() != 1 || !pool.Contains(tx.TransactionID) {
		t.Fatal("transaction was not stored")
	}
	if err := pool.Add(tx); !errors.Is(err, ErrDuplicateTransaction) {
		t.Fatalf("duplicate Add error = %v, want ErrDuplicateTransaction", err)
	}
	if !pool.Remove(tx.TransactionID) {
		t.Fatal("transaction was not removed")
	}
	if pool.Len() != 0 {
		t.Fatalf("mempool length = %d, want 0", pool.Len())
	}
}

func TestMempoolRejectsConflictingInput(t *testing.T) {
	previousID := strings.Repeat("22", 32)
	first := signedTestTransaction(t, previousID, 3)
	second := signedTestTransaction(t, previousID, 3)

	pool := New()
	if err := pool.Add(first); err != nil {
		t.Fatal(err)
	}
	if err := pool.Add(second); !errors.Is(err, ErrInputConflict) {
		t.Fatalf("conflict Add error = %v, want ErrInputConflict", err)
	}
}

func TestMempoolRemovesConfirmedTransactions(t *testing.T) {
	pool := New()
	tx := signedTestTransaction(t, strings.Repeat("33", 32), 1)
	if err := pool.Add(tx); err != nil {
		t.Fatal(err)
	}

	pool.RemoveBlockTransactions([]*transaction.Transaction{tx})
	if pool.Len() != 0 {
		t.Fatalf("mempool length = %d after block confirmation, want 0", pool.Len())
	}
}
