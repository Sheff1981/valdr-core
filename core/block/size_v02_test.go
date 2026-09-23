package block

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/core/transaction"
)

func TestCanonicalSerializationGoldenVector(t *testing.T) {
	tx := &transaction.Transaction{
		Version: transaction.Version,
		Inputs: []transaction.Input{{
			PreviousTransactionID: strings.Repeat("aa", 32),
			OutputIndex:           7,
		}},
		Outputs: []transaction.Output{{
			Amount:    42,
			Recipient: "VDR1golden",
		}},
		Timestamp: 123456789,
		PublicKey: "pub",
		Signature: "sig",
	}
	candidate := &Block{
		Version:           1,
		Height:            2,
		PreviousBlockHash: "prev",
		MerkleRoot:        "merkle",
		Timestamp:         123,
		Difficulty:        4,
		Nonce:             9,
		Transactions:      []*transaction.Transaction{tx},
		ChainID:           "valdr-devnet-1",
		ExtraData:         "x",
	}

	const wantHex = "00000001000000000000000200000000000000047072657600000000000000066d65726b6c65000000000000007b00000000000000040000000000000009000000000000000e76616c64722d6465766e65742d31000000000000000178000000000000000100000000000000ae000000000000000e76616c64722d6465766e65742d310000000100000000075bcd150000000000000001000000000000004061616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161000000070000000000000001000000000000002a000000000000000a56445231676f6c64656e00000000000000037075620000000000000003736967"
	got := hex.EncodeToString(candidate.CanonicalBytes())
	if got != wantHex {
		t.Fatalf("canonical block bytes=%s want=%s", got, wantHex)
	}
	if candidate.SerializedSize() != 283 {
		t.Fatalf("serialized size=%d want=283", candidate.SerializedSize())
	}
}
