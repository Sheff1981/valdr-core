package blockchain

import (
	"errors"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/transaction"
)

func TestRejectsOversizedBlockBeforeExpensiveValidation(t *testing.T) {
	chain := New()
	oversized := &transaction.Transaction{
		Version:   transaction.Version,
		PublicKey: strings.Repeat("x", block.MaxSerializedSize),
	}
	candidate := block.New(
		1,
		chain.Tip().BlockHash,
		config.GenesisTimestamp+60,
		1,
		0,
		[]*transaction.Transaction{oversized},
		"",
	)

	err := chain.AddBlock(candidate)
	if !errors.Is(err, ErrBlockTooLarge) {
		t.Fatalf("AddBlock error=%v want ErrBlockTooLarge", err)
	}
}
