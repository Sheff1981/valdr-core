package wallet

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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

// Create is retained only to prevent legacy callers from silently writing a
// plaintext v0.1 wallet. Stage 9 requires an explicit passphrase.
func (s *Store) Create(name string) (*Wallet, error) {
	return nil, ErrPassphraseRequired
}

func (s *Store) CreateEncrypted(
	name string,
	passphrase []byte,
) (*Wallet, error) {
	if len(passphrase) == 0 {
		return nil, ErrPassphraseRequired
	}
	if err := s.ensureDir(); err != nil {
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

	file, err := encryptWalletV2(w, passphrase)
	if err != nil {
		return nil, err
	}
	if err := writeWalletFile(path, file); err != nil {
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
		record, err := readWalletRecord(filepath.Join(s.Dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		items = append(items, record.metadata)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items, nil
}

func (s *Store) Unlock(
	selector string,
	passphrase []byte,
) (*Wallet, error) {
	record, err := s.find(selector)
	if err != nil {
		return nil, err
	}
	if record.v2 == nil {
		return nil, ErrWalletMigrationRequired
	}
	return decryptWalletV2(record.v2, passphrase)
}

func (s *Store) Export(
	selector string,
	passphrase []byte,
) (*Wallet, error) {
	return s.Unlock(selector, passphrase)
}

func (s *Store) BackupEncrypted(
	selector string,
	destination string,
) error {
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return ErrWalletNotFound
	}
	record, err := s.find(selector)
	if err != nil {
		return err
	}
	if record.v2 == nil {
		return ErrWalletMigrationRequired
	}
	return writeWalletFile(destination, record.v2)
}

func (s *Store) ImportEncrypted(
	sourcePath string,
) (Metadata, error) {
	record, err := readEncryptedImportRecord(sourcePath)
	if err != nil {
		return Metadata{}, err
	}
	return s.importEncryptedRecord(record)
}

func (s *Store) ImportEncryptedVerified(
	sourcePath string,
	passphrase []byte,
) (Metadata, error) {
	if len(passphrase) == 0 {
		return Metadata{}, ErrPassphraseRequired
	}
	record, err := readEncryptedImportRecord(sourcePath)
	if err != nil {
		return Metadata{}, err
	}
	unlocked, err := decryptWalletV2(record.v2, passphrase)
	if err != nil {
		return Metadata{}, err
	}
	unlocked.PrivateKey = ""
	return s.importEncryptedRecord(record)
}

func readEncryptedImportRecord(
	sourcePath string,
) (*walletRecord, error) {
	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		return nil, ErrWalletNotFound
	}
	record, err := readWalletRecord(sourcePath)
	if err != nil {
		return nil, err
	}
	if record.v2 == nil {
		return nil, ErrWalletMigrationRequired
	}
	return record, nil
}

func (s *Store) importEncryptedRecord(
	record *walletRecord,
) (Metadata, error) {
	if err := s.ensureDir(); err != nil {
		return Metadata{}, err
	}

	items, err := s.List()
	if err != nil {
		return Metadata{}, err
	}
	for _, item := range items {
		if item.Address == record.metadata.Address {
			return Metadata{}, fmt.Errorf(
				"%w: address %s",
				ErrWalletExists,
				item.Address,
			)
		}
		if record.metadata.Name != "" &&
			item.Name == record.metadata.Name {
			return Metadata{}, fmt.Errorf(
				"%w: name %q",
				ErrWalletExists,
				item.Name,
			)
		}
	}

	destination := filepath.Join(
		s.Dir,
		record.metadata.Address+".json",
	)
	if err := writeWalletFile(destination, record.v2); err != nil {
		return Metadata{}, err
	}
	return record.metadata, nil
}

func (s *Store) Migrate(
	selector string,
	passphrase []byte,
) (Metadata, error) {
	if len(passphrase) == 0 {
		return Metadata{}, ErrPassphraseRequired
	}
	if err := s.ensureDir(); err != nil {
		return Metadata{}, err
	}
	record, err := s.find(selector)
	if err != nil {
		return Metadata{}, err
	}
	if record.v2 != nil {
		return Metadata{}, ErrWalletAlreadyV2
	}
	if record.legacy == nil {
		return Metadata{}, ErrUnsupportedWalletFile
	}
	if _, err := record.legacy.Private(); err != nil {
		return Metadata{}, err
	}

	file, err := encryptWalletV2(record.legacy, passphrase)
	if err != nil {
		return Metadata{}, err
	}
	if err := writeWalletFile(record.path, file); err != nil {
		return Metadata{}, err
	}
	return file.Metadata(), nil
}

func (s *Store) ensureDir() error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	return os.Chmod(s.Dir, 0o700)
}

type walletRecord struct {
	path     string
	metadata Metadata
	v2       *WalletFileV2
	legacy   *Wallet
}

func (s *Store) find(selector string) (*walletRecord, error) {
	selector = strings.TrimSpace(selector)
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
		record, err := readWalletRecord(filepath.Join(s.Dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if record.metadata.Address == selector ||
			record.metadata.Name == selector {
			return record, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrWalletNotFound, selector)
}

func readWalletRecord(path string) (*walletRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var object map[string]json.RawMessage
	if err := decodeJSONStrict(raw, &object, false); err != nil {
		return nil, ErrUnsupportedWalletFile
	}
	if versionRaw, exists := object["version"]; exists {
		var version int
		if err := json.Unmarshal(versionRaw, &version); err != nil {
			return nil, ErrUnsupportedWalletFile
		}
		if version != WalletFileVersionV2 {
			return nil, ErrUnsupportedWalletFile
		}
		var file WalletFileV2
		if err := decodeWalletV2Strict(raw, &file); err != nil {
			return nil, ErrUnsupportedWalletFile
		}
		if err := validateWalletV2Parameters(&file); err != nil {
			return nil, err
		}
		meta := file.Metadata()
		meta.Name = strings.TrimSpace(meta.Name)
		file.Name = meta.Name
		if err := validateMetadata(meta); err != nil {
			return nil, err
		}
		return &walletRecord{
			path:     path,
			metadata: meta,
			v2:       &file,
		}, nil
	}

	var legacy Wallet
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return nil, err
	}
	legacy.Name = strings.TrimSpace(legacy.Name)
	if _, err := legacy.Private(); err != nil {
		return nil, err
	}
	return &walletRecord{
		path:     path,
		metadata: legacy.Metadata(),
		legacy:   &legacy,
	}, nil
}

func writeWalletFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
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
	if err := enc.Encode(value); err != nil {
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


func decodeWalletV2Strict(raw []byte, target *WalletFileV2) error {
	if target == nil {
		return ErrUnsupportedWalletFile
	}
	return decodeJSONStrict(raw, target, true)
}

func decodeJSONStrict(raw []byte, target any, disallowUnknown bool) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if disallowUnknown {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return ErrUnsupportedWalletFile
		}
		return err
	}
	return nil
}
