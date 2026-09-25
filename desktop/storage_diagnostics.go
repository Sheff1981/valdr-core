package desktop

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const DefaultStorageDiagnosticsMaxEntries = 10000

type StorageDiagnostics struct {
	Path      string `json:"path"`
	Ready     bool   `json:"ready"`
	FileCount uint64 `json:"file_count"`
	SizeBytes uint64 `json:"size_bytes"`
	Truncated bool   `json:"truncated"`
	Error     string `json:"error,omitempty"`
}

func InspectStorage(path string, maxEntries int) StorageDiagnostics {
	path = strings.TrimSpace(path)
	result := StorageDiagnostics{Path: path}
	if path == "" {
		result.Error = "node data directory is not configured"
		return result
	}
	if maxEntries <= 0 {
		maxEntries = DefaultStorageDiagnosticsMaxEntries
	}

	info, err := os.Stat(path)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if !info.IsDir() {
		result.Error = "node data path is not a directory"
		return result
	}

	entries := 0
	err = filepath.WalkDir(path, func(
		current string,
		entry fs.DirEntry,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		if current == path || entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		entries++
		if entries > maxEntries {
			result.Truncated = true
			return fs.SkipAll
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		result.FileCount++
		if info.Size() > 0 {
			result.SizeBytes += uint64(info.Size())
		}
		return nil
	})
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Ready = true
	return result
}
