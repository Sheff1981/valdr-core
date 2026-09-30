package desktop

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const nodeIDPrefix = "valdr-"

// LoadOrCreateNodeID returns a stable per-installation P2P identity.
// Every independent VALDR installation must have a distinct node ID; using a
// compiled constant would make two Desktop instances reject each other as a
// self-connection.
func LoadOrCreateNodeID(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("node id path is required")
	}
	if raw, err := os.ReadFile(path); err == nil {
		value := strings.TrimSpace(string(raw))
		if validDesktopNodeID(value) {
			return value, nil
		}
		return "", errors.New("invalid persisted VALDR node id")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	value := nodeIDPrefix + hex.EncodeToString(random)

	tmp, err := os.CreateTemp(filepath.Dir(path), ".node-id-*.tmp")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}
	if err := tmp.Chmod(0o600); err != nil {
		cleanup()
		return "", err
	}
	if _, err := tmp.WriteString(value + "\n"); err != nil {
		cleanup()
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return "", err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return "", err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return "", err
	}
	return value, nil
}

func validDesktopNodeID(value string) bool {
	if !strings.HasPrefix(value, nodeIDPrefix) || len(value) != len(nodeIDPrefix)+32 {
		return false
	}
	_, err := hex.DecodeString(value[len(nodeIDPrefix):])
	return err == nil
}
