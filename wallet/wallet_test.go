package wallet

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

func TestWalletStoreCreateListExportAndSign(t *testing.T) {
	store := NewStore(t.TempDir())
	passphrase := []byte("correct horse battery staple")

	created, err := store.CreateEncrypted("alice", passphrase)
	if err != nil {
		t.Fatal(err)
	}
	if !valdrcrypto.ValidateAddress(created.Address) {
		t.Fatalf("invalid address %s", created.Address)
	}

	path := filepath.Join(store.Dir, created.Address+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(created.PrivateKey)) ||
		bytes.Contains(raw, []byte("private_key")) {
		t.Fatal("encrypted wallet file contains plaintext private key material")
	}
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

	exported, err := store.Export("alice", passphrase)
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

func TestPlaintextCreateAPIIsDisabled(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.Create("legacy-new"); !errors.Is(err, ErrPassphraseRequired) {
		t.Fatalf("Create error=%v want ErrPassphraseRequired", err)
	}
}
