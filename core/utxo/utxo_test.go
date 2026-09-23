package utxo

import (
	"errors"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/core/transaction"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

func testKeyAndAddress(t *testing.T) (privateKeyHex string, address string) {
	t.Helper()

	key, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	privateKeyHex, err = valdrcrypto.EncodePrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	address, err = valdrcrypto.AddressFromPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return privateKeyHex, address
}

func signTransaction(t *testing.T, tx *transaction.Transaction, privateKeyHex string) {
	t.Helper()
	key, err := valdrcrypto.DecodePrivateKey(privateKeyHex)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Sign(key); err != nil {
		t.Fatal(err)
	}
}

func TestSpendCreatesUTXOsAndUpdatesBalances(t *testing.T) {
	ownerKey, ownerAddress := testKeyAndAddress(t)
	_, recipientAddress := testKeyAndAddress(t)

	initialTxID := strings.Repeat("11", 32)
	set, err := New([]UTXO{{
		TransactionID: initialTxID,
		OutputIndex:   0,
		Amount:        100,
		Recipient:     ownerAddress,
	}})
	if err != nil {
		t.Fatal(err)
	}

	tx := transaction.New(
		[]transaction.Input{{
			PreviousTransactionID: initialTxID,
			OutputIndex:           0,
		}},
		[]transaction.Output{
			{Amount: 60, Recipient: recipientAddress},
			{Amount: 40, Recipient: ownerAddress},
		},
		1790121960,
	)
	signTransaction(t, tx, ownerKey)

	if err := set.ApplyTransaction(tx); err != nil {
		t.Fatal(err)
	}

	if _, exists := set.Get(initialTxID, 0); exists {
		t.Fatal("spent UTXO still exists")
	}
	if got, err := set.Balance(recipientAddress); err != nil || got != 60 {
		t.Fatalf("recipient balance = %d, err=%v; want 60", got, err)
	}
	if got, err := set.Balance(ownerAddress); err != nil || got != 40 {
		t.Fatalf("owner change balance = %d, err=%v; want 40", got, err)
	}
	if set.Len() != 2 {
		t.Fatalf("UTXO count = %d, want 2", set.Len())
	}
}

func TestDoubleSpendRejected(t *testing.T) {
	ownerKey, ownerAddress := testKeyAndAddress(t)
	_, recipientAddress := testKeyAndAddress(t)

	initialTxID := strings.Repeat("22", 32)
	set, err := New([]UTXO{{
		TransactionID: initialTxID,
		OutputIndex:   0,
		Amount:        75,
		Recipient:     ownerAddress,
	}})
	if err != nil {
		t.Fatal(err)
	}

	tx := transaction.New(
		[]transaction.Input{{
			PreviousTransactionID: initialTxID,
			OutputIndex:           0,
		}},
		[]transaction.Output{{Amount: 75, Recipient: recipientAddress}},
		1790121961,
	)
	signTransaction(t, tx, ownerKey)

	if err := set.ApplyTransaction(tx); err != nil {
		t.Fatal(err)
	}
	if err := set.ApplyTransaction(tx); !errors.Is(err, ErrUTXONotFound) {
		t.Fatalf("second spend error = %v, want ErrUTXONotFound", err)
	}
}

func TestWrongOwnerRejected(t *testing.T) {
	_, ownerAddress := testKeyAndAddress(t)
	attackerKey, attackerAddress := testKeyAndAddress(t)

	initialTxID := strings.Repeat("33", 32)
	set, err := New([]UTXO{{
		TransactionID: initialTxID,
		OutputIndex:   0,
		Amount:        50,
		Recipient:     ownerAddress,
	}})
	if err != nil {
		t.Fatal(err)
	}

	tx := transaction.New(
		[]transaction.Input{{
			PreviousTransactionID: initialTxID,
			OutputIndex:           0,
		}},
		[]transaction.Output{{Amount: 50, Recipient: attackerAddress}},
		1790121962,
	)
	signTransaction(t, tx, attackerKey)

	if err := set.ApplyTransaction(tx); !errors.Is(err, ErrNotOwner) {
		t.Fatalf("ApplyTransaction error = %v, want ErrNotOwner", err)
	}
}

func TestInsufficientFundsRejected(t *testing.T) {
	ownerKey, ownerAddress := testKeyAndAddress(t)
	_, recipientAddress := testKeyAndAddress(t)

	initialTxID := strings.Repeat("44", 32)
	set, err := New([]UTXO{{
		TransactionID: initialTxID,
		OutputIndex:   0,
		Amount:        100,
		Recipient:     ownerAddress,
	}})
	if err != nil {
		t.Fatal(err)
	}

	tx := transaction.New(
		[]transaction.Input{{
			PreviousTransactionID: initialTxID,
			OutputIndex:           0,
		}},
		[]transaction.Output{{Amount: 101, Recipient: recipientAddress}},
		1790121963,
	)
	signTransaction(t, tx, ownerKey)

	if err := set.ApplyTransaction(tx); !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("ApplyTransaction error = %v, want ErrInsufficientFunds", err)
	}
}

func TestImplicitFeeIsInputMinusOutput(t *testing.T) {
	ownerKey, ownerAddress := testKeyAndAddress(t)
	_, recipientAddress := testKeyAndAddress(t)

	initialTxID := strings.Repeat("55", 32)
	set, err := New([]UTXO{{
		TransactionID: initialTxID,
		OutputIndex:   0,
		Amount:        100,
		Recipient:     ownerAddress,
	}})
	if err != nil {
		t.Fatal(err)
	}

	tx := transaction.New(
		[]transaction.Input{{
			PreviousTransactionID: initialTxID,
			OutputIndex:           0,
		}},
		[]transaction.Output{{Amount: 90, Recipient: recipientAddress}},
		1790121964,
	)
	signTransaction(t, tx, ownerKey)

	fee, err := set.ApplyTransactionWithFee(tx)
	if err != nil {
		t.Fatal(err)
	}
	if fee != 10 {
		t.Fatalf("fee = %d, want 10", fee)
	}
	if got, err := set.Balance(recipientAddress); err != nil || got != 90 {
		t.Fatalf("recipient balance = %d, err=%v; want 90", got, err)
	}
}

func TestTransactionBatchIsAtomic(t *testing.T) {
	ownerKey, ownerAddress := testKeyAndAddress(t)
	_, recipientAddress := testKeyAndAddress(t)

	initialTxID := strings.Repeat("66", 32)
	set, err := New([]UTXO{{
		TransactionID: initialTxID,
		OutputIndex:   0,
		Amount:        100,
		Recipient:     ownerAddress,
	}})
	if err != nil {
		t.Fatal(err)
	}

	first := transaction.New(
		[]transaction.Input{{
			PreviousTransactionID: initialTxID,
			OutputIndex:           0,
		}},
		[]transaction.Output{
			{Amount: 50, Recipient: recipientAddress},
			{Amount: 50, Recipient: ownerAddress},
		},
		1790121965,
	)
	signTransaction(t, first, ownerKey)

	second := transaction.New(
		[]transaction.Input{{
			PreviousTransactionID: strings.Repeat("77", 32),
			OutputIndex:           0,
		}},
		[]transaction.Output{{Amount: 50, Recipient: recipientAddress}},
		1790121966,
	)
	signTransaction(t, second, ownerKey)

	if err := set.ApplyTransactions([]*transaction.Transaction{first, second}); err == nil {
		t.Fatal("invalid transaction batch was accepted")
	}

	if got, err := set.Balance(ownerAddress); err != nil || got != 100 {
		t.Fatalf("owner balance after rollback = %d, err=%v; want 100", got, err)
	}
	if got, err := set.Balance(recipientAddress); err != nil || got != 0 {
		t.Fatalf("recipient balance after rollback = %d, err=%v; want 0", got, err)
	}
	if _, exists := set.Get(initialTxID, 0); !exists {
		t.Fatal("original UTXO disappeared after failed batch")
	}
}
