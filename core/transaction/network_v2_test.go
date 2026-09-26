package transaction

import (
	"errors"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
)

func TestV2TransactionIsBoundToExpectedChain(t *testing.T) {
	signer, recipient := testSignerAndRecipient(t)
	testnet, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	devnet2, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}

	tx := NewForChain(
		testnet.ChainID,
		[]Input{{
			PreviousTransactionID: strings.Repeat("ab", 32),
			OutputIndex:           1,
		}},
		[]Output{{
			Amount:    25_000_000,
			Recipient: recipient,
		}},
		testnet.GenesisTimestamp+60,
	)
	if err := tx.Sign(signer); err != nil {
		t.Fatal(err)
	}
	if tx.Version != VersionV2 || tx.ChainID != testnet.ChainID {
		t.Fatalf("unexpected v2 transaction identity: %+v", tx)
	}
	if err := tx.ValidateForChain(testnet.ChainID); err != nil {
		t.Fatalf("Testnet transaction rejected on Testnet: %v", err)
	}
	if err := tx.ValidateForChain(devnet2.ChainID); !errors.Is(err, ErrInvalidChainID) {
		t.Fatalf("cross-network validation error=%v want ErrInvalidChainID", err)
	}
	if err := tx.Validate(); !errors.Is(err, ErrInvalidChainID) {
		t.Fatalf("legacy Validate accepted v2 transaction: %v", err)
	}
}

func TestV2CoinbaseIsBoundToExpectedChain(t *testing.T) {
	_, recipient := testSignerAndRecipient(t)
	testnet, _ := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	devnet2, _ := config.ResolveNetworkProfile(config.NetworkDevnetV02)

	coinbase, err := NewCoinbaseForChain(
		testnet.ChainID,
		1,
		recipient,
		testnet.InitialSubsidyVDR*config.AtomicUnitsPerVDR,
		testnet.GenesisTimestamp+60,
	)
	if err != nil {
		t.Fatal(err)
	}
	if coinbase.Version != VersionV2 ||
		coinbase.ChainID != testnet.ChainID {
		t.Fatalf("unexpected v2 coinbase: %+v", coinbase)
	}
	if err := coinbase.ValidateCoinbaseForChain(
		1,
		testnet.InitialSubsidyVDR*config.AtomicUnitsPerVDR,
		testnet.ChainID,
	); err != nil {
		t.Fatal(err)
	}
	if err := coinbase.ValidateCoinbaseForChain(
		1,
		testnet.InitialSubsidyVDR*config.AtomicUnitsPerVDR,
		devnet2.ChainID,
	); err == nil {
		t.Fatal("Testnet coinbase accepted on Devnet2")
	}
}

func TestLegacyTransactionStillOmitsExplicitChainID(t *testing.T) {
	signer, recipient := testSignerAndRecipient(t)
	tx := New(
		[]Input{{
			PreviousTransactionID: strings.Repeat("cd", 32),
			OutputIndex:           0,
		}},
		[]Output{{
			Amount:    1_000,
			Recipient: recipient,
		}},
		config.GenesisTimestamp+60,
	)
	if err := tx.Sign(signer); err != nil {
		t.Fatal(err)
	}
	if tx.Version != VersionLegacy || tx.ChainID != "" {
		t.Fatalf("legacy transaction format changed: %+v", tx)
	}
	if err := tx.ValidateForChain(config.ChainID); err != nil {
		t.Fatal(err)
	}
}
