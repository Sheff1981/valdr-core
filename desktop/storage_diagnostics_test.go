package desktop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectStorageReportsBoundedDirectoryState(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.dat"), []byte("12345"), 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.dat"), []byte("1234567"), 0o600); err != nil {
		t.Fatal(err)
	}

	got := InspectStorage(root, 100)
	if !got.Ready || got.Error != "" {
		t.Fatalf("unexpected storage diagnostics: %+v", got)
	}
	if got.FileCount != 2 || got.SizeBytes != 12 || got.Truncated {
		t.Fatalf("unexpected storage totals: %+v", got)
	}
}

func TestInspectStorageRejectsMissingPath(t *testing.T) {
	got := InspectStorage(filepath.Join(t.TempDir(), "missing"), 100)
	if got.Ready || got.Error == "" {
		t.Fatalf("missing path unexpectedly ready: %+v", got)
	}
}

func TestInspectStorageCapsDirectoryWalk(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 5; i++ {
		name := filepath.Join(root, string(rune('a'+i))+".dat")
		if err := os.WriteFile(name, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	got := InspectStorage(root, 2)
	if !got.Ready || !got.Truncated {
		t.Fatalf("bounded walk did not report truncation: %+v", got)
	}
	if got.FileCount > 2 {
		t.Fatalf("bounded walk file count=%d want <=2", got.FileCount)
	}
}
