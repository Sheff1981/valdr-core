package wallet

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/core/utxo"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

func TestWalletStoreCreateListExportAndSign(t *testing.T) {
	store := NewStore(t.TempDir())

	created, err := store.Create("alice")
	if err != nil {
		t.Fatal(err)
	}
	if !valdrcrypto.ValidateAddress(created.Address) {
		t.Fatalf("invalid address %s", created.Address)
	}

	path := filepath.Join(store.Dir, created.Address+".json")
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0o077 != 0 {
			t.Fatalf(
				"wallet permissions = %o, want no group/other permissions",
				info.Mode().Perm(),
			)
		}
	}

	items, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "alice" {
		t.Fatalf("wallet list = %+v", items)
	}

	exported, err := store.Export("alice")
	if err != nil {
		t.Fatal(err)
	}
	if exported.PrivateKey == "" || exported.PublicKey == "" {
		t.Fatal("exported wallet missing keys")
	}

	message := []byte("VALDR wallet signature")
	sig, err := exported.Sign(message)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := valdrcrypto.DecodePublicKey(exported.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if !valdrcrypto.Verify(pub, message, sig) {
		t.Fatal("wallet signature did not verify")
	}
}

func TestCreateTransactionWithFeeReducesChange(t *testing.T) {
	sender, err := New("sender")
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := New("recipient")
	if err != nil {
		t.Fatal(err)
	}

	tx, err := sender.CreateTransactionWithFee(
		[]utxo.UTXO{{
			TransactionID: strings.Repeat("11", 32),
			OutputIndex:   0,
			Amount:        100,
			Recipient:     sender.Address,
		}},
		recipient.Address,
		60,
		10,
		1790121960,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(tx.Outputs) != 2 {
		t.Fatalf("output count = %d, want 2", len(tx.Outputs))
	}
	if tx.Outputs[0].Amount != 60 || tx.Outputs[0].Recipient != recipient.Address {
		t.Fatalf("recipient output = %+v", tx.Outputs[0])
	}
	if tx.Outputs[1].Amount != 30 || tx.Outputs[1].Recipient != sender.Address {
		t.Fatalf("change output = %+v, want 30 back to sender", tx.Outputs[1])
	}
	if err := tx.Validate(); err != nil {
		t.Fatalf("signed fee transaction invalid: %v", err)
	}
}
