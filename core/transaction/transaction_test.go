package transaction

import (
	"crypto/ecdsa"
	"errors"
	"strings"
	"testing"

	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

func testSignerAndRecipient(t *testing.T) (*ecdsa.PrivateKey, string) {
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

	return signer, recipient
}

func TestSignedTransactionRoundTrip(t *testing.T) {
	signer, recipient := testSignerAndRecipient(t)

	tx := New(
		[]Input{{
			PreviousTransactionID: strings.Repeat("ab", 32),
			OutputIndex:           1,
		}},
		[]Output{{
			Amount:    25_000_000,
			Recipient: recipient,
		}},
		1790121900,
	)

	if err := tx.Sign(signer); err != nil {
		t.Fatal(err)
	}
	if tx.Signature == "" {
		t.Fatal("signed transaction has empty signature")
	}
	if len(tx.TransactionID) != 64 {
		t.Fatalf("transaction id length = %d, want 64", len(tx.TransactionID))
	}
	if tx.TransactionID != tx.CalculateID() {
		t.Fatal("transaction id is not the SHA-256 hash of canonical transaction bytes")
	}
	if !tx.VerifySignature() {
		t.Fatal("valid transaction signature rejected")
	}
	if err := tx.Validate(); err != nil {
		t.Fatalf("valid transaction rejected: %v", err)
	}
}

func TestTamperedOutputRejectsSignature(t *testing.T) {
	signer, recipient := testSignerAndRecipient(t)

	tx := New(
		[]Input{{
			PreviousTransactionID: strings.Repeat("cd", 32),
			OutputIndex:           0,
		}},
		[]Output{{
			Amount:    1_000,
			Recipient: recipient,
		}},
		1790121901,
	)
	if err := tx.Sign(signer); err != nil {
		t.Fatal(err)
	}

	tx.Outputs[0].Amount++
	if tx.VerifySignature() {
		t.Fatal("tampered transaction signature validated")
	}
	if err := tx.Validate(); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("Validate error = %v, want ErrInvalidSignature", err)
	}
}

func TestRejectsDuplicateInput(t *testing.T) {
	signer, recipient := testSignerAndRecipient(t)
	input := Input{
		PreviousTransactionID: strings.Repeat("ef", 32),
		OutputIndex:           2,
	}

	tx := New(
		[]Input{input, input},
		[]Output{{
			Amount:    50,
			Recipient: recipient,
		}},
		1790121902,
	)

	if err := tx.Sign(signer); !errors.Is(err, ErrDuplicateInput) {
		t.Fatalf("Sign error = %v, want ErrDuplicateInput", err)
	}
}

func TestRejectsZeroOutputAmount(t *testing.T) {
	signer, recipient := testSignerAndRecipient(t)

	tx := New(
		[]Input{{
			PreviousTransactionID: strings.Repeat("12", 32),
			OutputIndex:           0,
		}},
		[]Output{{
			Amount:    0,
			Recipient: recipient,
		}},
		1790121903,
	)

	if err := tx.Sign(signer); !errors.Is(err, ErrInvalidOutput) {
		t.Fatalf("Sign error = %v, want ErrInvalidOutput", err)
	}
}

func TestRejectsTamperedTransactionID(t *testing.T) {
	signer, recipient := testSignerAndRecipient(t)

	tx := New(
		[]Input{{
			PreviousTransactionID: strings.Repeat("34", 32),
			OutputIndex:           0,
		}},
		[]Output{{
			Amount:    100,
			Recipient: recipient,
		}},
		1790121904,
	)
	if err := tx.Sign(signer); err != nil {
		t.Fatal(err)
	}

	tx.TransactionID = strings.Repeat("00", 32)
	if err := tx.Validate(); !errors.Is(err, ErrInvalidTransactionID) {
		t.Fatalf("Validate error = %v, want ErrInvalidTransactionID", err)
	}
}
