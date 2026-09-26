package mempool

import (
	"strings"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/transaction"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

func TestMempoolRejectsTransactionFromAnotherV2Network(t *testing.T) {
	testnet, _ := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	devnet2, _ := config.ResolveNetworkProfile(config.NetworkDevnetV02)

	signer, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	recipientKey, err := valdrcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := valdrcrypto.AddressFromPublicKey(&recipientKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	tx := transaction.NewForChain(
		testnet.ChainID,
		[]transaction.Input{{
			PreviousTransactionID: strings.Repeat("ab", 32),
			OutputIndex:           0,
		}},
		[]transaction.Output{{
			Amount:    1_000,
			Recipient: recipient,
		}},
		time.Now().UTC().Unix(),
	)
	if err := tx.Sign(signer); err != nil {
		t.Fatal(err)
	}

	testnetPool := NewWithConfig(Config{ChainID: testnet.ChainID})
	if err := testnetPool.AddWithFee(tx, 0); err != nil {
		t.Fatalf("Testnet pool rejected Testnet transaction: %v", err)
	}

	devnetPool := NewWithConfig(Config{ChainID: devnet2.ChainID})
	err = devnetPool.AddWithFee(tx, 0)
	if err == nil || !strings.Contains(err.Error(), "chain id") {
		t.Fatalf("Devnet2 pool error=%v want chain-id rejection", err)
	}
}
