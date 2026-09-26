package wallet

import (
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/core/utxo"
)

func TestCreateTransactionForChainBuildsV2NetworkBoundTransaction(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	source, err := New("network-source")
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := New("network-recipient")
	if err != nil {
		t.Fatal(err)
	}

	tx, fee, err := source.CreateTransactionForChain(
		profile.ChainID,
		[]utxo.UTXO{{
			TransactionID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			OutputIndex:   0,
			Amount:        1_000_000,
			Recipient:     source.Address,
		}},
		recipient.Address,
		100_000,
		profile.MinRelayFeePerByte,
		time.Now().UTC().Unix(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if tx.Version != transaction.VersionV2 ||
		tx.ChainID != profile.ChainID {
		t.Fatalf("unexpected transaction network identity: %+v", tx)
	}
	if err := tx.ValidateForChain(profile.ChainID); err != nil {
		t.Fatal(err)
	}
	if fee < uint64(tx.SerializedSize())*profile.MinRelayFeePerByte {
		t.Fatalf("fee=%d size=%d rate=%d", fee, tx.SerializedSize(), profile.MinRelayFeePerByte)
	}
}
