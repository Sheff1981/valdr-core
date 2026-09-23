package utxo

import (
	"errors"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/transaction"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

func testCoinbaseAddress(t *testing.T) string {
	t.Helper()
	key, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	address, err := valdrcrypto.AddressFromPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return address
}

func TestApplyBlockCoinbaseCreatesRewardUTXO(t *testing.T) {
	address := testCoinbaseAddress(t)
	coinbase, err := transaction.NewCoinbase(
		1,
		address,
		config.InitialMiningReward,
		config.GenesisTimestamp+60,
	)
	if err != nil {
		t.Fatal(err)
	}

	set := NewEmpty()
	if err := set.ApplyBlockTransactions(
		1,
		[]*transaction.Transaction{coinbase},
		config.InitialMiningReward,
	); err != nil {
		t.Fatal(err)
	}

	balance, err := set.Balance(address)
	if err != nil {
		t.Fatal(err)
	}
	if balance != config.InitialMiningReward {
		t.Fatalf("balance = %d, want %d", balance, config.InitialMiningReward)
	}
}

func TestApplyBlockRejectsSecondCoinbase(t *testing.T) {
	address := testCoinbaseAddress(t)
	first, err := transaction.NewCoinbase(
		1,
		address,
		config.InitialMiningReward,
		config.GenesisTimestamp+60,
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := transaction.NewCoinbase(
		2,
		address,
		config.InitialMiningReward,
		config.GenesisTimestamp+60,
	)
	if err != nil {
		t.Fatal(err)
	}

	set := NewEmpty()
	err = set.ApplyBlockTransactions(
		1,
		[]*transaction.Transaction{first, second},
		config.InitialMiningReward,
	)
	if !errors.Is(err, ErrUnexpectedCoinbase) {
		t.Fatalf("ApplyBlockTransactions error = %v, want ErrUnexpectedCoinbase", err)
	}
}
