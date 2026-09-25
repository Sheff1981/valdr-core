package desktop

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Sheff1981/valdr-core/config"
)

var ErrDesktopPath = errors.New("invalid VALDR Desktop path configuration")

type Paths struct {
	Root      string `json:"root"`
	NodeData  string `json:"node_data"`
	Wallets   string `json:"wallets"`
	Logs      string `json:"logs"`
	Network   string `json:"network"`
}

func DefaultPaths(network string) (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	return resolvePaths(runtime.GOOS, home, os.Getenv, network)
}

func resolvePaths(
	goos string,
	home string,
	getenv func(string) string,
	network string,
) (Paths, error) {
	profile, err := config.ResolveNetworkProfile(network)
	if err != nil {
		return Paths{}, err
	}
	home = strings.TrimSpace(home)
	if home == "" {
		return Paths{}, ErrDesktopPath
	}
	if getenv == nil {
		getenv = func(string) string { return "" }
	}

	var root string
	switch goos {
	case "windows":
		base := strings.TrimSpace(getenv("LOCALAPPDATA"))
		if base == "" {
			base = filepath.Join(home, "AppData", "Local")
		}
		root = filepath.Join(base, "VALDR")
	case "darwin":
		root = filepath.Join(
			home,
			"Library",
			"Application Support",
			"VALDR",
		)
	case "linux":
		base := strings.TrimSpace(getenv("XDG_DATA_HOME"))
		if base == "" {
			base = filepath.Join(home, ".local", "share")
		}
		root = filepath.Join(base, "valdr")
	default:
		root = filepath.Join(home, ".valdr")
	}

	return Paths{
		Root:     root,
		NodeData: filepath.Join(root, "node", profile.Name),
		Wallets:  filepath.Join(root, "wallets"),
		Logs:     filepath.Join(root, "logs"),
		Network:  profile.Name,
	}, nil
}

func NormalizeNodeDataDirectory(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return "", ErrDesktopPath
	}
	clean := filepath.Clean(path)
	if filepath.Dir(clean) == clean {
		return "", ErrDesktopPath
	}
	if info, err := os.Stat(clean); err == nil {
		if !info.IsDir() {
			return "", ErrDesktopPath
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return clean, nil
}

func PrepareNodeDataDirectory(path string) (string, error) {
	clean, err := NormalizeNodeDataDirectory(path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(clean, 0o700); err != nil {
		return "", err
	}
	probe, err := os.CreateTemp(clean, ".valdr-write-test-*")
	if err != nil {
		return "", err
	}
	probeName := probe.Name()
	if err := probe.Close(); err != nil {
		_ = os.Remove(probeName)
		return "", err
	}
	if err := os.Remove(probeName); err != nil {
		return "", err
	}
	return clean, nil
}

func (p Paths) Ensure() error {
	for _, dir := range []string{p.Root, p.NodeData, p.Wallets, p.Logs} {
		if strings.TrimSpace(dir) == "" {
			return ErrDesktopPath
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}
