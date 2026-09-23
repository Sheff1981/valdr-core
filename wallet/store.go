package wallet

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	ErrWalletNotFound = errors.New("wallet not found")
	ErrWalletExists   = errors.New("wallet already exists")
)

type Store struct{ Dir string }

func DefaultDir() (string, error) {
	if override := os.Getenv("VALDR_WALLET_DIR"); override != "" {
		return override, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".valdr", "wallets"), nil
}

func NewStore(dir string) *Store { return &Store{Dir: dir} }

func (s *Store) Create(name string) (*Wallet, error) {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return nil, err
	}
	if err := os.Chmod(s.Dir, 0o700); err != nil {
		return nil, err
	}

	if name != "" {
		items, err := s.List()
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item.Name == name {
				return nil, fmt.Errorf("%w: name %q", ErrWalletExists, name)
			}
		}
	}

	w, err := New(name)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(s.Dir, w.Address+".json")
	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("%w: %s", ErrWalletExists, w.Address)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	if err := writeWallet(path, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Store) List() ([]Metadata, error) {
	entries, err := os.ReadDir(s.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return []Metadata{}, nil
	}
	if err != nil {
		return nil, err
	}

	items := make([]Metadata, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		w, err := readWallet(filepath.Join(s.Dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if _, err := w.Private(); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		items = append(items, w.Metadata())
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items, nil
}

func (s *Store) Export(selector string) (*Wallet, error) {
	if selector == "" {
		return nil, ErrWalletNotFound
	}

	entries, err := os.ReadDir(s.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrWalletNotFound
	}
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		w, err := readWallet(filepath.Join(s.Dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if w.Address == selector || w.Name == selector {
			if _, err := w.Private(); err != nil {
				return nil, err
			}
			return w, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrWalletNotFound, selector)
}

func writeWallet(path string, w *Wallet) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".wallet-*.tmp")
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

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(w); err != nil {
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
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Chmod(path, 0o600)
}

func readWallet(path string) (*Wallet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var w Wallet
	if err := json.Unmarshal(raw, &w); err != nil {
		return nil, err
	}
	w.Name = strings.TrimSpace(w.Name)
	return &w, nil
}
