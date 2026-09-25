package desktop

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDesktopPreferencesDefaultAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "desktop-settings.json")
	store := NewPreferenceStore(path)

	prefs, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if prefs != DefaultDesktopPreferences() {
		t.Fatalf("defaults=%+v want=%+v", prefs, DefaultDesktopPreferences())
	}

	prefs.Language = "ru"
	prefs.Theme = "classic"
	prefs.StartNode = false
	prefs.Advanced = true
	prefs.WalletAutoLockMinutes = 30
	prefs.PublicNode = true
	prefs.PublicNodeAdvertiseAddress = "node.valdr.example:17333"
	if err := store.Save(prefs); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("settings mode=%#o want=0600", info.Mode().Perm())
	}

	reloaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded != prefs {
		t.Fatalf("reloaded=%+v want=%+v", reloaded, prefs)
	}
}

func TestDesktopPreferencesRejectUnsupportedValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "desktop-settings.json")
	store := NewPreferenceStore(path)

	prefs := DefaultDesktopPreferences()
	prefs.Theme = "remote-theme"
	if err := store.Save(prefs); !errors.Is(err, ErrDesktopPreferences) {
		t.Fatalf("Save error=%v want ErrDesktopPreferences", err)
	}
	prefs = DefaultDesktopPreferences()
	prefs.Language = "xx"
	if err := store.Save(prefs); !errors.Is(err, ErrDesktopPreferences) {
		t.Fatalf("unsupported language Save error=%v want ErrDesktopPreferences", err)
	}

	if err := os.WriteFile(
		path,
		[]byte("{\"version\":1,\"language\":\"en\",\"theme\":\"dark\",\"start_node\":true,\"advanced\":false,\"unknown\":1}\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrDesktopPreferences) {
		t.Fatalf("Load error=%v want ErrDesktopPreferences", err)
	}
}


func TestDesktopPreferencesLoadLegacyFileDefaultsAutoLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "desktop-settings.json")
	if err := os.WriteFile(
		path,
		[]byte("{\"version\":1,\"language\":\"en\",\"theme\":\"dark\",\"start_node\":true,\"advanced\":false}\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	prefs, err := NewPreferenceStore(path).Load()
	if err != nil {
		t.Fatal(err)
	}
	if prefs.WalletAutoLockMinutes != 15 {
		t.Fatalf("auto-lock=%d want=15", prefs.WalletAutoLockMinutes)
	}
}


func TestDesktopPreferencesRejectUnsafePublicNodeConfiguration(t *testing.T) {
	store := NewPreferenceStore(filepath.Join(t.TempDir(), "desktop-settings.json"))

	prefs := DefaultDesktopPreferences()
	prefs.PublicNode = true
	prefs.PublicNodeAdvertiseAddress = "node.valdr.example:17333"
	if err := store.Save(prefs); !errors.Is(err, ErrDesktopPreferences) {
		t.Fatalf("public node without Advanced error=%v want ErrDesktopPreferences", err)
	}

	prefs.Advanced = true
	prefs.PublicNodeAdvertiseAddress = "127.0.0.1:17333"
	if err := store.Save(prefs); !errors.Is(err, ErrDesktopPreferences) {
		t.Fatalf("unsafe public address error=%v want ErrDesktopPreferences", err)
	}
}
