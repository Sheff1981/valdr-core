package desktop

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

const (
	AddressBookVersion  = 1
	MaxContactLabelSize = 128
)

var (
	ErrAddressBookInvalid   = errors.New("invalid VALDR address-book data")
	ErrAddressBookDuplicate = errors.New("VALDR address already exists in address book")
	ErrAddressBookNotFound  = errors.New("VALDR address-book contact not found")
)

type AddressBookContact struct {
	Label   string `json:"label"`
	Address string `json:"address"`
}

type addressBookFile struct {
	Version  int                  `json:"version"`
	Contacts []AddressBookContact `json:"contacts"`
}

type AddressBookStore struct {
	Path string
	mu   sync.Mutex
}

func NewAddressBookStore(path string) *AddressBookStore {
	return &AddressBookStore{Path: path}
}

func (s *AddressBookStore) List() ([]AddressBookContact, error) {
	if s == nil || strings.TrimSpace(s.Path) == "" {
		return nil, ErrAddressBookInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	book, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	return cloneContacts(book.Contacts), nil
}

func (s *AddressBookStore) Create(label, address string) (AddressBookContact, error) {
	if s == nil || strings.TrimSpace(s.Path) == "" {
		return AddressBookContact{}, ErrAddressBookInvalid
	}
	contact, err := normalizeContact(label, address)
	if err != nil {
		return AddressBookContact{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	book, err := s.loadLocked()
	if err != nil {
		return AddressBookContact{}, err
	}
	for _, existing := range book.Contacts {
		if existing.Address == contact.Address {
			return AddressBookContact{}, ErrAddressBookDuplicate
		}
	}
	book.Contacts = append(book.Contacts, contact)
	sortContacts(book.Contacts)
	if err := s.saveLocked(book); err != nil {
		return AddressBookContact{}, err
	}
	return contact, nil
}

func (s *AddressBookStore) Update(originalAddress, label, address string) (AddressBookContact, error) {
	if s == nil || strings.TrimSpace(s.Path) == "" {
		return AddressBookContact{}, ErrAddressBookInvalid
	}
	originalAddress = strings.TrimSpace(originalAddress)
	if !valdrcrypto.ValidateAddress(originalAddress) {
		return AddressBookContact{}, ErrAddressBookInvalid
	}
	contact, err := normalizeContact(label, address)
	if err != nil {
		return AddressBookContact{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	book, err := s.loadLocked()
	if err != nil {
		return AddressBookContact{}, err
	}
	index := -1
	for i, existing := range book.Contacts {
		if existing.Address == originalAddress {
			index = i
		}
		if existing.Address == contact.Address && existing.Address != originalAddress {
			return AddressBookContact{}, ErrAddressBookDuplicate
		}
	}
	if index < 0 {
		return AddressBookContact{}, ErrAddressBookNotFound
	}
	book.Contacts[index] = contact
	sortContacts(book.Contacts)
	if err := s.saveLocked(book); err != nil {
		return AddressBookContact{}, err
	}
	return contact, nil
}

func (s *AddressBookStore) Delete(address string) error {
	if s == nil || strings.TrimSpace(s.Path) == "" {
		return ErrAddressBookInvalid
	}
	address = strings.TrimSpace(address)
	if !valdrcrypto.ValidateAddress(address) {
		return ErrAddressBookInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	book, err := s.loadLocked()
	if err != nil {
		return err
	}
	index := -1
	for i, existing := range book.Contacts {
		if existing.Address == address {
			index = i
			break
		}
	}
	if index < 0 {
		return ErrAddressBookNotFound
	}
	book.Contacts = append(book.Contacts[:index], book.Contacts[index+1:]...)
	return s.saveLocked(book)
}

func normalizeContact(label, address string) (AddressBookContact, error) {
	label = strings.TrimSpace(label)
	address = strings.TrimSpace(address)
	if label == "" || len([]rune(label)) > MaxContactLabelSize || !valdrcrypto.ValidateAddress(address) {
		return AddressBookContact{}, ErrAddressBookInvalid
	}
	return AddressBookContact{Label: label, Address: address}, nil
}

func (s *AddressBookStore) loadLocked() (addressBookFile, error) {
	file, err := os.Open(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return addressBookFile{Version: AddressBookVersion, Contacts: []AddressBookContact{}}, nil
	}
	if err != nil {
		return addressBookFile{}, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var book addressBookFile
	if err := decoder.Decode(&book); err != nil {
		return addressBookFile{}, ErrAddressBookInvalid
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return addressBookFile{}, ErrAddressBookInvalid
	}
	if book.Version != AddressBookVersion {
		return addressBookFile{}, ErrAddressBookInvalid
	}
	seen := make(map[string]struct{}, len(book.Contacts))
	for i, contact := range book.Contacts {
		normalized, err := normalizeContact(contact.Label, contact.Address)
		if err != nil || normalized != contact {
			return addressBookFile{}, ErrAddressBookInvalid
		}
		if _, exists := seen[contact.Address]; exists {
			return addressBookFile{}, ErrAddressBookInvalid
		}
		seen[contact.Address] = struct{}{}
		book.Contacts[i] = normalized
	}
	sortContacts(book.Contacts)
	return book, nil
}

func (s *AddressBookStore) saveLocked(book addressBookFile) error {
	book.Version = AddressBookVersion
	sortContacts(book.Contacts)
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".address-book-*.tmp")
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
	if err := encoder.Encode(book); err != nil {
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

func sortContacts(contacts []AddressBookContact) {
	sort.Slice(contacts, func(i, j int) bool {
		left := strings.ToLower(contacts[i].Label)
		right := strings.ToLower(contacts[j].Label)
		if left == right {
			return contacts[i].Address < contacts[j].Address
		}
		return left < right
	})
}

func cloneContacts(contacts []AddressBookContact) []AddressBookContact {
	result := make([]AddressBookContact, len(contacts))
	copy(result, contacts)
	return result
}
