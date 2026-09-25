package desktop

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

const DesktopPreferencesVersion = 1

var ErrDesktopPreferences = errors.New("invalid VALDR Desktop preferences")

type DesktopPreferences struct {
	Version               int    `json:"version"`
	Language              string `json:"language"`
	Theme                 string `json:"theme"`
	StartNode             bool   `json:"start_node"`
	Advanced              bool   `json:"advanced"`
	WalletAutoLockMinutes      int    `json:"wallet_auto_lock_minutes"`
	PublicNode                 bool   `json:"public_node"`
	PublicNodeAdvertiseAddress string `json:"public_node_advertise_address"`
}

func DefaultDesktopPreferences() DesktopPreferences {
	return DesktopPreferences{
		Version:               DesktopPreferencesVersion,
		Language:              "en",
		Theme:                 "dark",
		StartNode:             true,
		Advanced:              false,
		WalletAutoLockMinutes: int(DefaultWalletAutoLock / time.Minute),
	}
}

type PreferenceStore struct {
	Path string
}

func NewPreferenceStore(path string) *PreferenceStore {
	return &PreferenceStore{Path: path}
}

func (s *PreferenceStore) Load() (DesktopPreferences, error) {
	if s == nil || s.Path == "" {
		return DesktopPreferences{}, ErrDesktopPreferences
	}
	file, err := os.Open(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return DefaultDesktopPreferences(), nil
	}
	if err != nil {
		return DesktopPreferences{}, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var prefs DesktopPreferences
	if err := decoder.Decode(&prefs); err != nil {
		return DesktopPreferences{}, ErrDesktopPreferences
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return DesktopPreferences{}, ErrDesktopPreferences
	}
	if prefs.WalletAutoLockMinutes == 0 {
		prefs.WalletAutoLockMinutes = int(DefaultWalletAutoLock / time.Minute)
	}
	if err := validateDesktopPreferences(prefs); err != nil {
		return DesktopPreferences{}, err
	}
	return prefs, nil
}

func (s *PreferenceStore) Save(prefs DesktopPreferences) error {
	if s == nil || s.Path == "" {
		return ErrDesktopPreferences
	}
	if err := validateDesktopPreferences(prefs); err != nil {
		return err
	}
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".desktop-settings-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}
	if err := tmp.Chmod(0o600); err != nil {
		cleanup()
		return err
	}
	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(prefs); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, s.Path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Chmod(s.Path, 0o600)
}

func validateDesktopPreferences(prefs DesktopPreferences) error {
	if prefs.Version != DesktopPreferencesVersion {
		return ErrDesktopPreferences
	}
	if prefs.Language != "en" && prefs.Language != "ru" {
		return ErrDesktopPreferences
	}
	if prefs.Theme != "dark" && prefs.Theme != "classic" {
		return ErrDesktopPreferences
	}
	timeout := time.Duration(prefs.WalletAutoLockMinutes) * time.Minute
	if err := validateWalletAutoLock(timeout); err != nil {
		return ErrDesktopPreferences
	}
	if prefs.PublicNode && !prefs.Advanced {
		return ErrDesktopPreferences
	}
	if prefs.PublicNodeAdvertiseAddress != "" {
		if err := ValidatePublicNodeAdvertiseAddress(
			prefs.PublicNodeAdvertiseAddress,
		); err != nil {
			return ErrDesktopPreferences
		}
	}
	if prefs.PublicNode && prefs.PublicNodeAdvertiseAddress == "" {
		return ErrDesktopPreferences
	}
	return nil
}
