package p2p

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	peerCacheV2Version      = 2
	peerCacheV2LegacyVersion = 1
	maxPeerCacheV2FileBytes = 1 << 20
)

type PeerCacheV2Entry struct {
	Address        string `json:"address"`
	LastSeenUTC    int64  `json:"last_seen_utc,omitempty"`
	LastSuccessUTC int64  `json:"last_success_utc,omitempty"`
	LastAttemptUTC int64  `json:"last_attempt_utc,omitempty"`
	Successes      uint32 `json:"successes,omitempty"`
	Failures       uint32 `json:"failures,omitempty"`
}

type peerCacheV2File struct {
	Version   int                `json:"version"`
	UpdatedAt int64              `json:"updated_at"`
	Addresses []string           `json:"addresses,omitempty"` // v1 migration only
	Peers     []PeerCacheV2Entry `json:"peers,omitempty"`
}

type PeerAddressBook struct {
	mu      sync.RWMutex
	public  bool
	entries map[string]PeerCacheV2Entry
}

func NewPeerAddressBook(public bool, entries []PeerCacheV2Entry) *PeerAddressBook {
	book := &PeerAddressBook{
		public:  public,
		entries: make(map[string]PeerCacheV2Entry),
	}
	for _, entry := range normalizePeerCacheEntries(entries, public) {
		book.entries[entry.Address] = entry
	}
	return book
}

func LoadPeerAddressBook(path string, public bool) (*PeerAddressBook, error) {
	entries, err := loadPeerCacheV2Entries(path, public)
	if err != nil {
		return nil, err
	}
	return NewPeerAddressBook(public, entries), nil
}

func (b *PeerAddressBook) Addresses() []string {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	entries := make([]PeerCacheV2Entry, 0, len(b.entries))
	for _, entry := range b.entries {
		entries = append(entries, entry)
	}
	b.mu.RUnlock()

	rankPeerCacheEntries(entries)
	addresses := make([]string, 0, len(entries))
	for _, entry := range entries {
		addresses = append(addresses, entry.Address)
	}
	return addresses
}

func (b *PeerAddressBook) Observe(address string, now time.Time) {
	if b == nil {
		return
	}
	address = strings.TrimSpace(address)
	if err := validateDiscoveredAddress(address, b.public); err != nil {
		return
	}
	ts := now.UTC().Unix()
	b.mu.Lock()
	entry := b.entries[address]
	entry.Address = address
	if ts > entry.LastSeenUTC {
		entry.LastSeenUTC = ts
	}
	b.entries[address] = entry
	b.trimLocked()
	b.mu.Unlock()
}

func (b *PeerAddressBook) RecordAttempt(address string, success bool, now time.Time) {
	if b == nil {
		return
	}
	address = strings.TrimSpace(address)
	if err := validateDiscoveredAddress(address, b.public); err != nil {
		return
	}
	ts := now.UTC().Unix()
	b.mu.Lock()
	entry := b.entries[address]
	entry.Address = address
	entry.LastAttemptUTC = ts
	if ts > entry.LastSeenUTC {
		entry.LastSeenUTC = ts
	}
	if success {
		entry.LastSuccessUTC = ts
		if entry.Successes < ^uint32(0) {
			entry.Successes++
		}
	} else if entry.Failures < ^uint32(0) {
		entry.Failures++
	}
	b.entries[address] = entry
	b.trimLocked()
	b.mu.Unlock()
}

func (b *PeerAddressBook) Save(path string) error {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	entries := make([]PeerCacheV2Entry, 0, len(b.entries))
	for _, entry := range b.entries {
		entries = append(entries, entry)
	}
	b.mu.RUnlock()
	return savePeerCacheV2Entries(path, b.public, entries)
}

func (b *PeerAddressBook) trimLocked() {
	if len(b.entries) <= DefaultMaxDiscoveredPeers {
		return
	}
	entries := make([]PeerCacheV2Entry, 0, len(b.entries))
	for _, entry := range b.entries {
		entries = append(entries, entry)
	}
	rankPeerCacheEntries(entries)
	for _, entry := range entries[DefaultMaxDiscoveredPeers:] {
		delete(b.entries, entry.Address)
	}
}

// LoadPeerCacheV2 remains the compatibility API used by callers that only need
// ranked peer addresses. v1 files are migrated in memory automatically.
func LoadPeerCacheV2(path string, public bool) ([]string, error) {
	entries, err := loadPeerCacheV2Entries(path, public)
	if err != nil {
		return nil, err
	}
	rankPeerCacheEntries(entries)
	addresses := make([]string, 0, len(entries))
	for _, entry := range entries {
		addresses = append(addresses, entry.Address)
	}
	return addresses, nil
}

// SavePeerCacheV2 remains the compatibility API. It writes the v2 peer format.
func SavePeerCacheV2(path string, public bool, addresses []string) error {
	now := time.Now().UTC().Unix()
	entries := make([]PeerCacheV2Entry, 0, len(addresses))
	for _, address := range addresses {
		entries = append(entries, PeerCacheV2Entry{
			Address:     address,
			LastSeenUTC: now,
		})
	}
	return savePeerCacheV2Entries(path, public, entries)
}

func loadPeerCacheV2Entries(path string, public bool) ([]PeerCacheV2Entry, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > maxPeerCacheV2FileBytes {
		return nil, fmt.Errorf("peer cache exceeds %d bytes", maxPeerCacheV2FileBytes)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxPeerCacheV2FileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxPeerCacheV2FileBytes {
		return nil, fmt.Errorf("peer cache exceeds %d bytes", maxPeerCacheV2FileBytes)
	}

	var cache peerCacheV2File
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	switch cache.Version {
	case peerCacheV2LegacyVersion:
		entries := make([]PeerCacheV2Entry, 0, len(cache.Addresses))
		for _, address := range cache.Addresses {
			entries = append(entries, PeerCacheV2Entry{
				Address:     address,
				LastSeenUTC: cache.UpdatedAt,
			})
		}
		return normalizePeerCacheEntries(entries, public), nil
	case peerCacheV2Version:
		return normalizePeerCacheEntries(cache.Peers, public), nil
	default:
		return nil, fmt.Errorf("unsupported peer cache version %d", cache.Version)
	}
}

func savePeerCacheV2Entries(path string, public bool, entries []PeerCacheV2Entry) error {
	entries = normalizePeerCacheEntries(entries, public)
	rankPeerCacheEntries(entries)
	cache := peerCacheV2File{
		Version:   peerCacheV2Version,
		UpdatedAt: time.Now().UTC().Unix(),
		Peers:     entries,
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	temp := path + ".tmp"
	if err := os.WriteFile(temp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(temp, path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return os.Chmod(path, 0o600)
}

func normalizePeerCacheEntries(entries []PeerCacheV2Entry, public bool) []PeerCacheV2Entry {
	merged := make(map[string]PeerCacheV2Entry, len(entries))
	for _, entry := range entries {
		address := strings.TrimSpace(entry.Address)
		if address == "" {
			continue
		}
		if err := validateDiscoveredAddress(address, public); err != nil {
			continue
		}
		entry.Address = address
		current, exists := merged[address]
		if !exists {
			merged[address] = entry
			continue
		}
		if entry.LastSeenUTC > current.LastSeenUTC {
			current.LastSeenUTC = entry.LastSeenUTC
		}
		if entry.LastSuccessUTC > current.LastSuccessUTC {
			current.LastSuccessUTC = entry.LastSuccessUTC
		}
		if entry.LastAttemptUTC > current.LastAttemptUTC {
			current.LastAttemptUTC = entry.LastAttemptUTC
		}
		if entry.Successes > current.Successes {
			current.Successes = entry.Successes
		}
		if entry.Failures > current.Failures {
			current.Failures = entry.Failures
		}
		merged[address] = current
	}
	result := make([]PeerCacheV2Entry, 0, len(merged))
	for _, entry := range merged {
		result = append(result, entry)
	}
	rankPeerCacheEntries(result)
	if len(result) > DefaultMaxDiscoveredPeers {
		result = result[:DefaultMaxDiscoveredPeers]
	}
	return result
}

func rankPeerCacheEntries(entries []PeerCacheV2Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		aHasSuccess := a.LastSuccessUTC > 0
		bHasSuccess := b.LastSuccessUTC > 0
		if aHasSuccess != bHasSuccess {
			return aHasSuccess
		}
		if a.LastSuccessUTC != b.LastSuccessUTC {
			return a.LastSuccessUTC > b.LastSuccessUTC
		}
		if a.Successes != b.Successes {
			return a.Successes > b.Successes
		}
		if a.Failures != b.Failures {
			return a.Failures < b.Failures
		}
		if a.LastSeenUTC != b.LastSeenUTC {
			return a.LastSeenUTC > b.LastSeenUTC
		}
		return a.Address < b.Address
	})
}

func normalizePeerCacheAddresses(addresses []string, public bool) []string {
	entries := make([]PeerCacheV2Entry, 0, len(addresses))
	for _, address := range addresses {
		entries = append(entries, PeerCacheV2Entry{Address: address})
	}
	entries = normalizePeerCacheEntries(entries, public)
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry.Address)
	}
	return result
}
