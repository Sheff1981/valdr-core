package desktop

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/wallet"
)

func TestWalletSessionManagerAutoLockUsesInactivity(t *testing.T) {
	store := wallet.NewStore(t.TempDir())
	passphrase := []byte("stage12-auto-lock-passphrase")
	created, err := store.CreateEncrypted("auto-lock", passphrase)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 9, 25, 20, 0, 0, 0, time.UTC)
	manager, err := newWalletSessionManager(
		store,
		5*time.Minute,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Unlock(created.Address, passphrase); err != nil {
		t.Fatal(err)
	}
	if !manager.IsUnlocked(created.Address) {
		t.Fatal("wallet should be unlocked immediately after unlock")
	}

	now = now.Add(4 * time.Minute)
	secret, err := manager.Passphrase(created.Address)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(secret, passphrase) {
		t.Fatal("session returned unexpected passphrase")
	}
	clearBytes(secret)

	now = now.Add(4 * time.Minute)
	if !manager.IsUnlocked(created.Address) {
		t.Fatal("wallet auto-locked before timeout from last activity")
	}

	now = now.Add(2 * time.Minute)
	if manager.IsUnlocked(created.Address) {
		t.Fatal("wallet remained unlocked after inactivity timeout")
	}
	if _, err := manager.Passphrase(created.Address); !errors.Is(
		err,
		ErrWalletLocked,
	) {
		t.Fatalf("Passphrase error=%v want ErrWalletLocked", err)
	}
}

func TestWalletSessionManagerClearsSecretsOnLockAndReplacement(t *testing.T) {
	store := wallet.NewStore(t.TempDir())
	passphrase := []byte("stage12-secret-zeroization-passphrase")
	created, err := store.CreateEncrypted("zeroize", passphrase)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 9, 25, 20, 0, 0, 0, time.UTC)
	manager, err := newWalletSessionManager(
		store,
		15*time.Minute,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Unlock(created.Address, passphrase); err != nil {
		t.Fatal(err)
	}
	first := manager.sessions[created.Address].passphrase
	if len(first) == 0 {
		t.Fatal("test session passphrase unexpectedly empty")
	}

	if _, err := manager.Unlock(created.Address, passphrase); err != nil {
		t.Fatal(err)
	}
	for i, b := range first {
		if b != 0 {
			t.Fatalf("replaced session secret byte %d was not cleared", i)
		}
	}

	second := manager.sessions[created.Address].passphrase
	manager.Lock(created.Address)
	for i, b := range second {
		if b != 0 {
			t.Fatalf("locked session secret byte %d was not cleared", i)
		}
	}
	if manager.IsUnlocked(created.Address) {
		t.Fatal("wallet remained unlocked after explicit lock")
	}
}

func TestWalletSessionManagerTimeoutChangeLocksExpiredSessions(t *testing.T) {
	store := wallet.NewStore(t.TempDir())
	passphrase := []byte("stage12-timeout-change-passphrase")
	created, err := store.CreateEncrypted("timeout-change", passphrase)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 9, 25, 20, 0, 0, 0, time.UTC)
	manager, err := newWalletSessionManager(
		store,
		10*time.Minute,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Unlock(created.Address, passphrase); err != nil {
		t.Fatal(err)
	}

	now = now.Add(6 * time.Minute)
	if err := manager.SetTimeout(5 * time.Minute); err != nil {
		t.Fatal(err)
	}
	if manager.IsUnlocked(created.Address) {
		t.Fatal("shorter timeout did not lock an already-expired session")
	}

	if err := manager.SetTimeout(MinWalletAutoLock - time.Second); !errors.Is(
		err,
		ErrWalletAutoLock,
	) {
		t.Fatalf("short timeout error=%v want ErrWalletAutoLock", err)
	}
	if err := manager.SetTimeout(MaxWalletAutoLock + time.Second); !errors.Is(
		err,
		ErrWalletAutoLock,
	) {
		t.Fatalf("long timeout error=%v want ErrWalletAutoLock", err)
	}
}
