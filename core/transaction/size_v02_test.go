package transaction

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func TestCanonicalSerializationGoldenVector(t *testing.T) {
	tx := &Transaction{
		Version: Version,
		Inputs: []Input{{
			PreviousTransactionID: strings.Repeat("aa", 32),
			OutputIndex:           7,
		}},
		Outputs: []Output{{
			Amount:    42,
			Recipient: "VDR1golden",
		}},
		Timestamp: 123456789,
		PublicKey: "pub",
		Signature: "sig",
	}

	const wantHex = "000000000000000e76616c64722d6465766e65742d310000000100000000075bcd150000000000000001000000000000004061616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161000000070000000000000001000000000000002a000000000000000a56445231676f6c64656e00000000000000037075620000000000000003736967"
	got := hex.EncodeToString(tx.CanonicalBytes())
	if got != wantHex {
		t.Fatalf("canonical tx bytes=%s want=%s", got, wantHex)
	}
	if tx.SerializedSize() != 174 {
		t.Fatalf("serialized size=%d want=174", tx.SerializedSize())
	}
}

func TestSignRejectsOversizedTransaction(t *testing.T) {
	signer, recipient := testSignerAndRecipient(t)
	outputs := make([]Output, 0, 2_500)
	for i := 0; i < 2_500; i++ {
		outputs = append(outputs, Output{Amount: 1, Recipient: recipient})
	}
	tx := New(
		[]Input{{
			PreviousTransactionID: strings.Repeat("ab", 32),
			OutputIndex:           0,
		}},
		outputs,
		1790121999,
	)

	err := tx.Sign(signer)
	if !errors.Is(err, ErrTransactionTooLarge) {
		t.Fatalf("Sign error=%v want ErrTransactionTooLarge", err)
	}
}
