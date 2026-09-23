package utxo

import (
	"errors"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/transaction"
)

func TestBlockFeesRaiseCoinbaseMaximumAndAllowUnderclaim(t *testing.T) {
	ownerKey, ownerAddress := testKeyAndAddress(t)
	_, recipientAddress := testKeyAndAddress(t)
	miner := testCoinbaseAddress(t)
	initialTxID := strings.Repeat("91", 32)

	makeSetAndTx := func(t *testing.T) (*Set, *transaction.Transaction) {
		t.Helper()
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
			config.GenesisTimestamp+60,
		)
		signTransaction(t, tx, ownerKey)
		return set, tx
	}

	t.Run("claim subsidy plus all fees", func(t *testing.T) {
		set, tx := makeSetAndTx(t)
		coinbase, err := transaction.NewCoinbase(
			1,
			miner,
			config.InitialMiningReward+10,
			config.GenesisTimestamp+60,
		)
		if err != nil {
			t.Fatal(err)
		}
		if err := set.ApplyBlockTransactions(
			1,
			[]*transaction.Transaction{coinbase, tx},
			config.InitialMiningReward,
		); err != nil {
			t.Fatal(err)
		}
		got, err := set.Balance(miner)
		if err != nil {
			t.Fatal(err)
		}
		if got != config.InitialMiningReward+10 {
			t.Fatalf("miner balance=%d want=%d", got, config.InitialMiningReward+10)
		}
	})

	t.Run("underclaim burns unclaimed fees", func(t *testing.T) {
		set, tx := makeSetAndTx(t)
		coinbase, err := transaction.NewCoinbase(
			1,
			miner,
			config.InitialMiningReward,
			config.GenesisTimestamp+60,
		)
		if err != nil {
			t.Fatal(err)
		}
		if err := set.ApplyBlockTransactions(
			1,
			[]*transaction.Transaction{coinbase, tx},
			config.InitialMiningReward,
		); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("overclaim rejected", func(t *testing.T) {
		set, tx := makeSetAndTx(t)
		coinbase, err := transaction.NewCoinbase(
			1,
			miner,
			config.InitialMiningReward+11,
			config.GenesisTimestamp+60,
		)
		if err != nil {
			t.Fatal(err)
		}
		err = set.ApplyBlockTransactions(
			1,
			[]*transaction.Transaction{coinbase, tx},
			config.InitialMiningReward,
		)
		if !errors.Is(err, transaction.ErrInvalidCoinbaseReward) {
			t.Fatalf("ApplyBlockTransactions error=%v want ErrInvalidCoinbaseReward", err)
		}
	})
}
