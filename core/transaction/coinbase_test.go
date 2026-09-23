package transaction

import (
	"errors"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

func TestCoinbaseTransaction(t *testing.T) {
	key, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	address, err := valdrcrypto.AddressFromPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := NewCoinbase(
		7,
		address,
		config.InitialMiningReward,
		config.GenesisTimestamp+420,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !tx.IsCoinbase() {
		t.Fatal("coinbase transaction was not recognized")
	}
	if tx.Inputs[0].PreviousTransactionID != CoinbasePreviousTransactionID {
		t.Fatal("coinbase marker transaction id mismatch")
	}
	if tx.Inputs[0].OutputIndex != 7 {
		t.Fatalf("coinbase height marker = %d, want 7", tx.Inputs[0].OutputIndex)
	}
	if err := tx.ValidateCoinbase(7, config.InitialMiningReward); err != nil {
		t.Fatalf("valid coinbase rejected: %v", err)
	}
	if err := tx.Validate(); !errors.Is(err, ErrCoinbaseRequiresBlock) {
		t.Fatalf("normal Validate error = %v, want ErrCoinbaseRequiresBlock", err)
	}
}

func TestCoinbaseWrongRewardRejected(t *testing.T) {
	key, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	address, err := valdrcrypto.AddressFromPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := NewCoinbase(
		1,
		address,
		config.InitialMiningReward+1,
		config.GenesisTimestamp+60,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = tx.ValidateCoinbase(1, config.InitialMiningReward)
	if !errors.Is(err, ErrInvalidCoinbaseReward) {
		t.Fatalf("ValidateCoinbase error = %v, want ErrInvalidCoinbaseReward", err)
	}
}
