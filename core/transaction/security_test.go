package transaction

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

func TestSecurityNegativeAmountCannotDecode(t *testing.T) {
	raw := []byte(`{
		"version":1,
		"inputs":[],
		"outputs":[{"amount":-1,"recipient":"VDR1INVALID"}],
		"timestamp":1790121900,
		"public_key":"",
		"signature":"",
		"transaction_id":""
	}`)

	var tx Transaction
	if err := json.Unmarshal(raw, &tx); err == nil {
		t.Fatal("negative output amount decoded into uint64 transaction field")
	}
}

func TestSecurityMalformedSignatureRejected(t *testing.T) {
	signer, recipient := testSignerAndRecipient(t)
	tx := New(
		[]Input{{
			PreviousTransactionID: strings.Repeat("aa", 32),
			OutputIndex:           0,
		}},
		[]Output{{
			Amount:    100,
			Recipient: recipient,
		}},
		1790121905,
	)
	if err := tx.Sign(signer); err != nil {
		t.Fatal(err)
	}

	tx.Signature = "not-hex"
	tx.TransactionID = tx.CalculateID()
	if err := tx.Validate(); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("malformed signature error = %v, want ErrInvalidSignature", err)
	}
}

func TestSecuritySignatureFromDifferentKeyRejected(t *testing.T) {
	signer, recipient := testSignerAndRecipient(t)
	tx := New(
		[]Input{{
			PreviousTransactionID: strings.Repeat("bb", 32),
			OutputIndex:           0,
		}},
		[]Output{{
			Amount:    100,
			Recipient: recipient,
		}},
		1790121906,
	)
	if err := tx.Sign(signer); err != nil {
		t.Fatal(err)
	}

	attacker, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	signature, err := valdrcrypto.Sign(attacker, tx.SigningBytes())
	if err != nil {
		t.Fatal(err)
	}
	tx.Signature = fmtHex(signature)
	tx.TransactionID = tx.CalculateID()

	if err := tx.Validate(); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("wrong-key signature error = %v, want ErrInvalidSignature", err)
	}
}

func fmtHex(value []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, len(value)*2)
	for i, b := range value {
		out[i*2] = digits[b>>4]
		out[i*2+1] = digits[b&0x0f]
	}
	return string(out)
}
