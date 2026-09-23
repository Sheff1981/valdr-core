package blockchain

import (
	"errors"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/consensus"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/core/utxo"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestSecurityRejectsDuplicateBlock(t *testing.T) {
	chain := New()
	miner := testMinerAddress(t)

	block1, err := chain.Append(
		config.GenesisTimestamp+60,
		[]*transaction.Transaction{
			testCoinbase(t, 1, miner, config.GenesisTimestamp+60),
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	before := chain.Len()
	err = chain.AddBlock(block1)
	if !errors.Is(err, ErrDuplicateBlock) {
		t.Fatalf("duplicate block error = %v, want ErrDuplicateBlock", err)
	}
	if chain.Len() != before {
		t.Fatalf("chain length changed from %d to %d", before, chain.Len())
	}
}

func TestSecurityRejectsDuplicateConfirmedTransaction(t *testing.T) {
	chain := New()
	minerWallet, err := wallet.New("miner")
	if err != nil {
		t.Fatal(err)
	}
	recipientWallet, err := wallet.New("recipient")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := chain.Append(
		config.GenesisTimestamp+60,
		[]*transaction.Transaction{
			testCoinbase(t, 1, minerWallet.Address, config.GenesisTimestamp+60),
		},
	); err != nil {
		t.Fatal(err)
	}

	available, err := chain.UTXOs(minerWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	payment, err := minerWallet.CreateTransaction(
		available,
		recipientWallet.Address,
		10*config.AtomicUnitsPerVDR,
		config.GenesisTimestamp+90,
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := chain.Append(
		config.GenesisTimestamp+120,
		[]*transaction.Transaction{
			testCoinbase(t, 2, minerWallet.Address, config.GenesisTimestamp+120),
			payment,
		},
	); err != nil {
		t.Fatal(err)
	}

	tip := chain.Tip()
	timestamp := config.GenesisTimestamp + 180
	candidate := block.New(
		3,
		tip.BlockHash,
		timestamp,
		consensus.NextDifficulty(chain.blocks),
		0,
		[]*transaction.Transaction{
			testCoinbase(t, 3, minerWallet.Address, timestamp),
			payment,
		},
		"",
	)
	if err := consensus.Mine(candidate); err != nil {
		t.Fatal(err)
	}

	err = chain.AddBlock(candidate)
	if !errors.Is(err, ErrDuplicateTransaction) {
		t.Fatalf("duplicate transaction error = %v, want ErrDuplicateTransaction", err)
	}
	if chain.Height() != 2 {
		t.Fatalf("height = %d after rejected replay, want 2", chain.Height())
	}
}

func TestSecurityRejectsDuplicateTransactionInsideBlock(t *testing.T) {
	chain := New()
	minerWallet, err := wallet.New("miner")
	if err != nil {
		t.Fatal(err)
	}
	recipientWallet, err := wallet.New("recipient")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := chain.Append(
		config.GenesisTimestamp+60,
		[]*transaction.Transaction{
			testCoinbase(t, 1, minerWallet.Address, config.GenesisTimestamp+60),
		},
	); err != nil {
		t.Fatal(err)
	}

	available, err := chain.UTXOs(minerWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	payment, err := minerWallet.CreateTransaction(
		available,
		recipientWallet.Address,
		10*config.AtomicUnitsPerVDR,
		config.GenesisTimestamp+90,
	)
	if err != nil {
		t.Fatal(err)
	}

	tip := chain.Tip()
	timestamp := config.GenesisTimestamp + 120
	candidate := block.New(
		2,
		tip.BlockHash,
		timestamp,
		consensus.NextDifficulty(chain.blocks),
		0,
		[]*transaction.Transaction{
			testCoinbase(t, 2, minerWallet.Address, timestamp),
			payment,
			payment,
		},
		"",
	)
	if err := consensus.Mine(candidate); err != nil {
		t.Fatal(err)
	}

	err = chain.AddBlock(candidate)
	if !errors.Is(err, ErrDuplicateTransaction) {
		t.Fatalf("duplicate transaction error = %v, want ErrDuplicateTransaction", err)
	}
	if chain.Height() != 1 {
		t.Fatalf("height = %d after rejected block, want 1", chain.Height())
	}
}

func TestSecurityRejectsTamperedNonce(t *testing.T) {
	chain := New()
	miner := testMinerAddress(t)
	timestamp := config.GenesisTimestamp + 60

	candidate := block.New(
		1,
		chain.Tip().BlockHash,
		timestamp,
		consensus.NextDifficulty(chain.blocks),
		0,
		[]*transaction.Transaction{
			testCoinbase(t, 1, miner, timestamp),
		},
		"",
	)
	if err := consensus.Mine(candidate); err != nil {
		t.Fatal(err)
	}

	candidate.Nonce++
	err := chain.AddBlock(candidate)
	if !errors.Is(err, ErrInvalidHash) {
		t.Fatalf("tampered nonce error = %v, want ErrInvalidHash", err)
	}
	if chain.Height() != 0 {
		t.Fatalf("height = %d after bad nonce, want 0", chain.Height())
	}
}

func TestSecurityRejectsCreationOutsideCoinbaseAtomically(t *testing.T) {
	chain := New()
	minerWallet, err := wallet.New("miner")
	if err != nil {
		t.Fatal(err)
	}
	recipientWallet, err := wallet.New("recipient")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := chain.Append(
		config.GenesisTimestamp+60,
		[]*transaction.Transaction{
			testCoinbase(t, 1, minerWallet.Address, config.GenesisTimestamp+60),
		},
	); err != nil {
		t.Fatal(err)
	}

	available, err := chain.UTXOs(minerWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	if len(available) != 1 {
		t.Fatalf("miner UTXO count = %d, want 1", len(available))
	}

	forged := transaction.New(
		[]transaction.Input{{
			PreviousTransactionID: available[0].TransactionID,
			OutputIndex:           available[0].OutputIndex,
		}},
		[]transaction.Output{{
			Amount:    available[0].Amount + 1,
			Recipient: recipientWallet.Address,
		}},
		config.GenesisTimestamp+90,
	)
	key, err := minerWallet.Private()
	if err != nil {
		t.Fatal(err)
	}
	if err := forged.Sign(key); err != nil {
		t.Fatal(err)
	}

	timestamp := config.GenesisTimestamp + 120
	tip := chain.Tip()
	candidate := block.New(
		2,
		tip.BlockHash,
		timestamp,
		consensus.NextDifficulty(chain.blocks),
		0,
		[]*transaction.Transaction{
			testCoinbase(t, 2, minerWallet.Address, timestamp),
			forged,
		},
		"",
	)
	if err := consensus.Mine(candidate); err != nil {
		t.Fatal(err)
	}

	err = chain.AddBlock(candidate)
	if !errors.Is(err, ErrInvalidTransaction) ||
		!errors.Is(err, utxo.ErrInsufficientFunds) {
		t.Fatalf(
			"forged value error = %v, want ErrInvalidTransaction + ErrInsufficientFunds",
			err,
		)
	}

	minerBalance, err := chain.Balance(minerWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	if minerBalance != config.InitialMiningReward {
		t.Fatalf("miner balance = %d after rejected block", minerBalance)
	}
	recipientBalance, err := chain.Balance(recipientWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	if recipientBalance != 0 {
		t.Fatalf("recipient balance = %d after rejected forged value", recipientBalance)
	}
	if chain.Height() != 1 {
		t.Fatalf("height = %d after rejected forged value, want 1", chain.Height())
	}
}

func TestSecurityRejectsDoubleSpendWithinBlockAtomically(t *testing.T) {
	chain := New()
	owner, err := wallet.New("owner")
	if err != nil {
		t.Fatal(err)
	}
	recipientA, err := wallet.New("recipient-a")
	if err != nil {
		t.Fatal(err)
	}
	recipientB, err := wallet.New("recipient-b")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := chain.Append(
		config.GenesisTimestamp+60,
		[]*transaction.Transaction{
			testCoinbase(t, 1, owner.Address, config.GenesisTimestamp+60),
		},
	); err != nil {
		t.Fatal(err)
	}

	available, err := chain.UTXOs(owner.Address)
	if err != nil {
		t.Fatal(err)
	}
	if len(available) != 1 {
		t.Fatalf("owner UTXO count = %d, want 1", len(available))
	}

	input := transaction.Input{
		PreviousTransactionID: available[0].TransactionID,
		OutputIndex:           available[0].OutputIndex,
	}
	key, err := owner.Private()
	if err != nil {
		t.Fatal(err)
	}

	first := transaction.New(
		[]transaction.Input{input},
		[]transaction.Output{{
			Amount:    available[0].Amount,
			Recipient: recipientA.Address,
		}},
		config.GenesisTimestamp+90,
	)
	if err := first.Sign(key); err != nil {
		t.Fatal(err)
	}

	second := transaction.New(
		[]transaction.Input{input},
		[]transaction.Output{{
			Amount:    available[0].Amount,
			Recipient: recipientB.Address,
		}},
		config.GenesisTimestamp+91,
	)
	if err := second.Sign(key); err != nil {
		t.Fatal(err)
	}

	timestamp := config.GenesisTimestamp + 120
	tip := chain.Tip()
	candidate := block.New(
		2,
		tip.BlockHash,
		timestamp,
		consensus.NextDifficulty(chain.blocks),
		0,
		[]*transaction.Transaction{
			testCoinbase(t, 2, owner.Address, timestamp),
			first,
			second,
		},
		"",
	)
	if err := consensus.Mine(candidate); err != nil {
		t.Fatal(err)
	}

	err = chain.AddBlock(candidate)
	if !errors.Is(err, ErrInvalidTransaction) ||
		!errors.Is(err, utxo.ErrUTXONotFound) {
		t.Fatalf(
			"double-spend error = %v, want ErrInvalidTransaction + ErrUTXONotFound",
			err,
		)
	}

	ownerBalance, err := chain.Balance(owner.Address)
	if err != nil {
		t.Fatal(err)
	}
	if ownerBalance != config.InitialMiningReward {
		t.Fatalf("owner balance = %d after rejected double-spend", ownerBalance)
	}
	for name, address := range map[string]string{
		"A": recipientA.Address,
		"B": recipientB.Address,
	} {
		balance, err := chain.Balance(address)
		if err != nil {
			t.Fatal(err)
		}
		if balance != 0 {
			t.Fatalf("recipient %s balance = %d after rejected block", name, balance)
		}
	}
	if chain.Height() != 1 {
		t.Fatalf("height = %d after rejected double-spend, want 1", chain.Height())
	}
}
