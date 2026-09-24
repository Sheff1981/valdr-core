package wallet

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWalletV2WrongPassphraseRejected(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.CreateEncrypted("alice", []byte("correct-passphrase")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Export("alice", []byte("wrong-passphrase")); !errors.Is(err, ErrWalletAuthentication) {
		t.Fatalf("Export error=%v want ErrWalletAuthentication", err)
	}
}

func TestWalletV2CiphertextTamperRejected(t *testing.T) {
	store := NewStore(t.TempDir())
	passphrase := []byte("tamper-test-passphrase")
	created, err := store.CreateEncrypted("alice", passphrase)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(store.Dir, created.Address+".json")
	raw, err := os.ReadFile(path)
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
	if err := writeWalletFile(path, &file); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Export("alice", passphrase); !errors.Is(err, ErrWalletAuthentication) {
		t.Fatalf("tampered Export error=%v want ErrWalletAuthentication", err)
	}
}

func TestWalletV2MetadataTamperRejected(t *testing.T) {
	store := NewStore(t.TempDir())
	passphrase := []byte("metadata-tamper-passphrase")
	created, err := store.CreateEncrypted("alice", passphrase)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(store.Dir, created.Address+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var file WalletFileV2
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	file.Name = "mallory"
	if err := writeWalletFile(path, &file); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Export("mallory", passphrase); !errors.Is(err, ErrWalletAuthentication) {
		t.Fatalf("metadata-tampered Export error=%v want ErrWalletAuthentication", err)
	}
}

func TestWalletV01MigrationEncryptsInPlace(t *testing.T) {
	store := NewStore(t.TempDir())
	legacy, err := New("legacy")
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(store.Dir, legacy.Address+".json")
	if err := writeWalletFile(path, legacy); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(before, []byte(legacy.PrivateKey)) ||
		!bytes.Contains(before, []byte("private_key")) {
		t.Fatal("legacy fixture is not plaintext as expected")
	}

	passphrase := []byte("new-migration-passphrase")
	meta, err := store.Migrate("legacy", passphrase)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Address != legacy.Address ||
		meta.PublicKey != legacy.PublicKey ||
		meta.Name != legacy.Name {
		t.Fatalf("migration metadata changed: got=%+v legacy=%+v", meta, legacy.Metadata())
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(after, []byte(legacy.PrivateKey)) ||
		bytes.Contains(after, []byte("private_key")) {
		t.Fatal("migrated wallet still contains plaintext private key")
	}
	if !bytes.Contains(after, []byte(`"version": 2`)) {
		t.Fatalf("migrated wallet missing v2 marker: %s", after)
	}

	unlocked, err := store.Export("legacy", passphrase)
	if err != nil {
		t.Fatal(err)
	}
	if unlocked.PrivateKey != legacy.PrivateKey {
		t.Fatal("migration changed private key")
	}
	if _, err := store.Migrate("legacy", passphrase); !errors.Is(err, ErrWalletAlreadyV2) {
		t.Fatalf("second migration error=%v want ErrWalletAlreadyV2", err)
	}
}

func TestLegacyWalletRequiresMigrationBeforeUnlock(t *testing.T) {
	store := NewStore(t.TempDir())
	legacy, err := New("legacy")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Dir, legacy.Address+".json")
	if err := writeWalletFile(path, legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Unlock("legacy", []byte("anything")); !errors.Is(err, ErrWalletMigrationRequired) {
		t.Fatalf("Unlock error=%v want ErrWalletMigrationRequired", err)
	}
}

func TestWalletV2UsesRandomSaltAndNonce(t *testing.T) {
	store := NewStore(t.TempDir())
	passphrase := []byte("same-passphrase")
	first, err := store.CreateEncrypted("first", passphrase)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateEncrypted("second", passphrase)
	if err != nil {
		t.Fatal(err)
	}

	read := func(address string) WalletFileV2 {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(store.Dir, address+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var file WalletFileV2
		if err := json.Unmarshal(raw, &file); err != nil {
			t.Fatal(err)
		}
		return file
	}
	a := read(first.Address)
	b := read(second.Address)
	if a.KDF.Salt == b.KDF.Salt {
		t.Fatal("wallets reused scrypt salt")
	}
	if a.Cipher.Nonce == b.Cipher.Nonce {
		t.Fatal("wallets reused AES-GCM nonce")
	}
}
