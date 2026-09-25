package p2p

import (
	"os"
	"path/filepath"
	"reflect"
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
	if info.Mode().Perm()&0o077 != 0 {
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
