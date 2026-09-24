package wallet

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptedWalletBackupAndRestore(t *testing.T) {
	sourceStore := NewStore(filepath.Join(t.TempDir(), "source"))
	passphrase := []byte("backup-restore-passphrase")
	created, err := sourceStore.CreateEncrypted("primary", passphrase)
	if err != nil {
		t.Fatal(err)
	}

	backupPath := filepath.Join(t.TempDir(), "primary.valdr-wallet")
	if err := sourceStore.BackupEncrypted("primary", backupPath); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(created.PrivateKey)) ||
		bytes.Contains(raw, []byte("private_key")) {
		t.Fatal("encrypted backup leaked private key material")
	}
	var envelope WalletFileV2
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Version != WalletFileVersionV2 ||
		envelope.Address != created.Address {
		t.Fatalf("unexpected backup envelope: %+v", envelope)
	}

	restoreStore := NewStore(filepath.Join(t.TempDir(), "restored"))
	meta, err := restoreStore.ImportEncrypted(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Address != created.Address ||
		meta.PublicKey != created.PublicKey ||
		meta.Name != created.Name {
		t.Fatalf("restored metadata=%+v created=%+v", meta, created.Metadata())
	}

	unlocked, err := restoreStore.Unlock(meta.Address, passphrase)
	if err != nil {
		t.Fatal(err)
	}
	if unlocked.PrivateKey != created.PrivateKey {
		t.Fatal("restored wallet private key differs after decrypt")
	}
}

func TestImportEncryptedRejectsLegacyPlaintextWallet(t *testing.T) {
	legacy, err := New("legacy-backup")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "legacy.json")
	raw, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}

	store := NewStore(filepath.Join(t.TempDir(), "destination"))
	if _, err := store.ImportEncrypted(path); !errors.Is(err, ErrWalletMigrationRequired) {
		t.Fatalf("ImportEncrypted error=%v want ErrWalletMigrationRequired", err)
	}
	items, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("legacy import created wallet: %+v", items)
	}
}

func TestImportEncryptedRejectsDuplicateWallet(t *testing.T) {
	source := NewStore(filepath.Join(t.TempDir(), "source"))
	passphrase := []byte("duplicate-backup-passphrase")
	created, err := source.CreateEncrypted("duplicate", passphrase)
	if err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(t.TempDir(), "duplicate.valdr-wallet")
	if err := source.BackupEncrypted(created.Address, backupPath); err != nil {
		t.Fatal(err)
	}

	destination := NewStore(filepath.Join(t.TempDir(), "destination"))
	if _, err := destination.ImportEncrypted(backupPath); err != nil {
		t.Fatal(err)
	}
	if _, err := destination.ImportEncrypted(backupPath); !errors.Is(err, ErrWalletExists) {
		t.Fatalf("second import error=%v want ErrWalletExists", err)
	}
}
