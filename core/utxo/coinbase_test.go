package utxo

import (
	"errors"
	"strings"
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

func TestApplyBlockCreditsFeesToCoinbase(t *testing.T) {
	ownerKey, ownerAddress := testKeyAndAddress(t)
	_, recipientAddress := testKeyAndAddress(t)
	minerAddress := testCoinbaseAddress(t)

	initialTxID := strings.Repeat("ab", 32)
	set, err := New([]UTXO{{
		TransactionID: initialTxID,
		OutputIndex:   0,
		Amount:        100,
		Recipient:     ownerAddress,
	}})
	if err != nil {
		t.Fatal(err)
	}

	payment := transaction.New(
		[]transaction.Input{{
			PreviousTransactionID: initialTxID,
			OutputIndex:           0,
		}},
		[]transaction.Output{{Amount: 90, Recipient: recipientAddress}},
		config.GenesisTimestamp+30,
	)
	signTransaction(t, payment, ownerKey)

	coinbaseValue := config.InitialMiningReward + 10
	coinbase, err := transaction.NewCoinbase(
		1,
		minerAddress,
		coinbaseValue,
		config.GenesisTimestamp+60,
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := set.ApplyBlockTransactions(
		1,
		[]*transaction.Transaction{coinbase, payment},
		config.InitialMiningReward,
	); err != nil {
		t.Fatal(err)
	}

	if got, err := set.Balance(minerAddress); err != nil || got != coinbaseValue {
		t.Fatalf("miner balance = %d, err=%v; want %d", got, err, coinbaseValue)
	}
	if got, err := set.Balance(recipientAddress); err != nil || got != 90 {
		t.Fatalf("recipient balance = %d, err=%v; want 90", got, err)
	}
}

func TestApplyBlockRejectsCoinbaseThatDoesNotClaimExactFees(t *testing.T) {
	ownerKey, ownerAddress := testKeyAndAddress(t)
	_, recipientAddress := testKeyAndAddress(t)
	minerAddress := testCoinbaseAddress(t)

	initialTxID := strings.Repeat("cd", 32)
	set, err := New([]UTXO{{
		TransactionID: initialTxID,
		OutputIndex:   0,
		Amount:        100,
		Recipient:     ownerAddress,
	}})
	if err != nil {
		t.Fatal(err)
	}

	payment := transaction.New(
		[]transaction.Input{{
			PreviousTransactionID: initialTxID,
			OutputIndex:           0,
		}},
		[]transaction.Output{{Amount: 90, Recipient: recipientAddress}},
		config.GenesisTimestamp+30,
	)
	signTransaction(t, payment, ownerKey)

	coinbase, err := transaction.NewCoinbase(
		1,
		minerAddress,
		config.InitialMiningReward,
		config.GenesisTimestamp+60,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = set.ApplyBlockTransactions(
		1,
		[]*transaction.Transaction{coinbase, payment},
		config.InitialMiningReward,
	)
	if !errors.Is(err, ErrInvalidCoinbase) ||
		!errors.Is(err, transaction.ErrInvalidCoinbaseReward) {
		t.Fatalf(
			"coinbase fee error = %v, want ErrInvalidCoinbase + ErrInvalidCoinbaseReward",
			err,
		)
	}
}
