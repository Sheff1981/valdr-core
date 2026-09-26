package wallet

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWalletV2RejectsUnknownFieldsAndTrailingJSON(t *testing.T) {
	store := NewStore(t.TempDir())
	passphrase := []byte("strict-wallet-passphrase")
	created, err := store.CreateEncrypted("strict", passphrase)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Dir, created.Address+".json")

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	object["unexpected"] = true
	withUnknown, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, withUnknown, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Unlock("strict", passphrase); !errors.Is(err, ErrUnsupportedWalletFile) {
		t.Fatalf("unknown-field unlock error=%v want ErrUnsupportedWalletFile", err)
	}

	delete(object, "unexpected")
	clean, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	trailing := append(clean, []byte("\n{}")...)
	if err := os.WriteFile(path, trailing, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Unlock("strict", passphrase); !errors.Is(err, ErrUnsupportedWalletFile) {
		t.Fatalf("trailing-json unlock error=%v want ErrUnsupportedWalletFile", err)
	}
}

func TestWalletV2RejectsKDFAndNonceTamper(t *testing.T) {
	store := NewStore(t.TempDir())
	passphrase := []byte("wallet-parameter-passphrase")
	created, err := store.CreateEncrypted("params", passphrase)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Dir, created.Address+".json")

	read := func() WalletFileV2 {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var file WalletFileV2
		if err := json.Unmarshal(raw, &file); err != nil {
			t.Fatal(err)
		}
		return file
	}
	write := func(file WalletFileV2) {
		t.Helper()
		if err := writeWalletFile(path, &file); err != nil {
			t.Fatal(err)
		}
	}

	original := read()

	tamperedKDF := original
	tamperedKDF.KDF.N = original.KDF.N / 2
	write(tamperedKDF)
	if _, err := store.Unlock("params", passphrase); !errors.Is(err, ErrUnsupportedWalletFile) {
		t.Fatalf("KDF tamper error=%v want ErrUnsupportedWalletFile", err)
	}

	tamperedNonce := original
	tamperedNonce.Cipher.Nonce = "00"
	write(tamperedNonce)
	if _, err := store.Unlock("params", passphrase); !errors.Is(err, ErrUnsupportedWalletFile) {
		t.Fatalf("nonce tamper error=%v want ErrUnsupportedWalletFile", err)
	}
}

func TestImportEncryptedVerifiedTamperDoesNotWriteDestination(t *testing.T) {
	source := NewStore(filepath.Join(t.TempDir(), "source"))
	passphrase := []byte("verified-tamper-passphrase")
	created, err := source.CreateEncrypted("verified-tamper", passphrase)
	if err != nil {
		t.Fatal(err)
	}

	backup := filepath.Join(t.TempDir(), "tampered.valdr-wallet")
	if err := source.BackupEncrypted(created.Address, backup); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(backup)
	if err != nil {
		t.Fatal(err)
	}
	var file WalletFileV2
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.Cipher.Ciphertext) < 2 {
		t.Fatal("ciphertext unexpectedly short")
	}
	if file.Cipher.Ciphertext[0] == '0' {
		file.Cipher.Ciphertext = "1" + file.Cipher.Ciphertext[1:]
	} else {
		file.Cipher.Ciphertext = "0" + file.Cipher.Ciphertext[1:]
	}
	if err := writeWalletFile(backup, &file); err != nil {
		t.Fatal(err)
	}

	destination := NewStore(filepath.Join(t.TempDir(), "destination"))
	if _, err := destination.ImportEncryptedVerified(backup, passphrase); !errors.Is(err, ErrWalletAuthentication) {
		t.Fatalf("tampered import error=%v want ErrWalletAuthentication", err)
	}
	items, err := destination.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("tampered import wrote destination wallet: %+v", items)
	}
}
