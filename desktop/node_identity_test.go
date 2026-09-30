package desktop

import (
	"path/filepath"
	"testing"
)

func TestLoadOrCreateNodeIDStableAndUnique(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "a", "node-id")
	secondPath := filepath.Join(dir, "b", "node-id")

	first, err := LoadOrCreateNodeID(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	again, err := LoadOrCreateNodeID(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	if first != again {
		t.Fatalf("node id changed across reload: %q != %q", first, again)
	}
	if !validDesktopNodeID(first) {
		t.Fatalf("invalid generated node id: %q", first)
	}

	second, err := LoadOrCreateNodeID(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("independent installations received the same node id: %q", first)
	}
}
