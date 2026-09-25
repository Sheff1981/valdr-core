package p2p

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const peerCacheV2Version = 1

type peerCacheV2File struct {
	Version   int      `json:"version"`
	UpdatedAt int64    `json:"updated_at"`
	Addresses []string `json:"addresses"`
}

// LoadPeerCacheV2 loads non-consensus peer addresses learned by a v2 node.
// Missing cache files are normal on first start.
func LoadPeerCacheV2(path string, public bool) ([]string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var cache peerCacheV2File
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	if cache.Version != peerCacheV2Version {
		return nil, fmt.Errorf("unsupported peer cache version %d", cache.Version)
	}
	return normalizePeerCacheAddresses(cache.Addresses, public), nil
}

// SavePeerCacheV2 persists learned addresses atomically. It is operational
// metadata only and never participates in consensus.
func SavePeerCacheV2(path string, public bool, addresses []string) error {
	addresses = normalizePeerCacheAddresses(addresses, public)
	cache := peerCacheV2File{
		Version:   peerCacheV2Version,
		UpdatedAt: time.Now().UTC().Unix(),
		Addresses: addresses,
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

func normalizePeerCacheAddresses(addresses []string, public bool) []string {
	seen := make(map[string]struct{}, len(addresses))
	result := make([]string, 0, len(addresses))
	for _, address := range addresses {
		address = strings.TrimSpace(address)
		if address == "" {
			continue
		}
		if err := validateDiscoveredAddress(address, public); err != nil {
			continue
		}
		if _, exists := seen[address]; exists {
			continue
		}
		seen[address] = struct{}{}
		result = append(result, address)
	}
	sort.Strings(result)
	return result
}
