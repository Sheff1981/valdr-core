package wallet

import (
	"strings"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/core/mempool"
	"github.com/Sheff1981/valdr-core/core/utxo"
)

func TestCreateTransactionWithFeeRateMeetsExactSerializedPolicy(t *testing.T) {
	source, err := New("fee-source")
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := New("fee-recipient")
	if err != nil {
		t.Fatal(err)
	}

	available := []utxo.UTXO{{
		TransactionID: strings.Repeat("a", 64),
		OutputIndex:   0,
		Amount:        1_000_000,
		Recipient:     source.Address,
	}}
	tx, fee, err := source.CreateTransactionWithFeeRate(
		available,
		recipient.Address,
		100_000,
		1,
		time.Now().UTC().Unix(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Validate(); err != nil {
		t.Fatalf("signed transaction invalid: %v", err)
	}
	if fee < uint64(tx.SerializedSize()) {
		t.Fatalf(
			"fee=%d below exact serialized minimum=%d",
			fee,
			tx.SerializedSize(),
		)
	}
	if len(tx.Outputs) != 2 {
		t.Fatalf("outputs=%d want recipient+change", len(tx.Outputs))
	}
	if tx.Outputs[0].Amount != 100_000 ||
		tx.Outputs[0].Recipient != recipient.Address {
		t.Fatalf("unexpected recipient output: %+v", tx.Outputs[0])
	}
	if tx.Outputs[1].Recipient != source.Address {
		t.Fatalf("unexpected change output: %+v", tx.Outputs[1])
	}

	pool := mempool.NewWithConfig(mempool.Config{
		MinRelayFeePerByte: 1,
	})
	if err := pool.AddWithFee(tx, fee); err != nil {
		t.Fatalf("Testnet-style mempool rejected fee-aware tx: %v", err)
	}
}

func TestCreateTransactionWithFeeRateUsesAdditionalInputForFee(t *testing.T) {
	source, err := New("multi-input-source")
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := New("multi-input-recipient")
	if err != nil {
		t.Fatal(err)
	}

	available := []utxo.UTXO{
		{
			TransactionID: strings.Repeat("1", 64),
			OutputIndex:   0,
			Amount:        50_000,
			Recipient:     source.Address,
		},
		{
			TransactionID: strings.Repeat("2", 64),
			OutputIndex:   0,
			Amount:        50_000,
			Recipient:     source.Address,
		},
	}
	tx, fee, err := source.CreateTransactionWithFeeRate(
		available,
		recipient.Address,
		99_000,
		1,
		time.Now().UTC().Unix(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(tx.Inputs) != 2 {
		t.Fatalf("inputs=%d want=2", len(tx.Inputs))
	}
	if fee < uint64(tx.SerializedSize()) {
		t.Fatalf("fee=%d size=%d", fee, tx.SerializedSize())
	}
}

func TestCreateTransactionWithFeeRateInsufficientForAmountAndFee(t *testing.T) {
	source, err := New("insufficient-source")
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := New("insufficient-recipient")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = source.CreateTransactionWithFeeRate(
		[]utxo.UTXO{{
			TransactionID: strings.Repeat("3", 64),
			OutputIndex:   0,
			Amount:        100_000,
			Recipient:     source.Address,
		}},
		recipient.Address,
		100_000,
		1,
		time.Now().UTC().Unix(),
	)
	if err != ErrInsufficientFunds {
		t.Fatalf("error=%v want ErrInsufficientFunds", err)
	}
}

func TestLegacyCreateTransactionStillUsesZeroFee(t *testing.T) {
	source, err := New("legacy-source")
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := New("legacy-recipient")
	if err != nil {
		t.Fatal(err)
	}
	available := []utxo.UTXO{{
		TransactionID: strings.Repeat("4", 64),
		OutputIndex:   0,
		Amount:        200_000,
		Recipient:     source.Address,
	}}
	tx, err := source.CreateTransaction(
		available,
		recipient.Address,
		100_000,
		time.Now().UTC().Unix(),
	)
	if err != nil {
		t.Fatal(err)
	}
	fee, err := implicitFee(available[0].Amount, tx.Outputs)
	if err != nil {
		t.Fatal(err)
	}
	if fee != 0 {
		t.Fatalf("legacy fee=%d want=0", fee)
	}
}
