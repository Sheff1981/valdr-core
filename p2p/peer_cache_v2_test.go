package p2p

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestPeerCacheV2MissingFileIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "peers-v2.json")
	got, err := LoadPeerCacheV2(path, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got=%v want empty cache", got)
	}
}

func TestPeerCacheV2RoundTripFiltersAndDeduplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "peers-v2.json")
	input := []string{
		"203.0.113.10:17333",
		"seed.example.org:17333",
		"203.0.113.10:17333",
		"127.0.0.1:17333",
		"0.0.0.0:17333",
		"",
	}
	if err := SavePeerCacheV2(path, true, input); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("peer cache permissions=%o want private", info.Mode().Perm())
	}

	got, err := LoadPeerCacheV2(path, true)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"203.0.113.10:17333", "seed.example.org:17333"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%v want=%v", got, want)
	}
}

func TestPeerCacheV2DevnetAllowsLoopback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "peers-v2.json")
	want := []string{"127.0.0.1:7333"}
	if err := SavePeerCacheV2(path, false, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadPeerCacheV2(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%v want=%v", got, want)
	}
}


func TestPeerCacheV2RejectsOversizeFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "peers-v2.json")
	if err := os.WriteFile(path, make([]byte, maxPeerCacheV2FileBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPeerCacheV2(path, false); err == nil {
		t.Fatal("oversize peer cache was accepted")
	}
}

func TestPeerCacheV2CapsPersistedAddresses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "peers-v2.json")
	input := make([]string, 0, DefaultMaxDiscoveredPeers+32)
	for i := 0; i < DefaultMaxDiscoveredPeers+32; i++ {
		input = append(input, fmt.Sprintf("peer-%04d.example.org:17333", i))
	}
	if err := SavePeerCacheV2(path, false, input); err != nil {
		t.Fatal(err)
	}
	got, err := LoadPeerCacheV2(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != DefaultMaxDiscoveredPeers {
		t.Fatalf("cache size=%d want=%d", len(got), DefaultMaxDiscoveredPeers)
	}
}
