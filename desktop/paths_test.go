package desktop

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
)

func TestResolveDesktopPathsByPlatform(t *testing.T) {
	tests := []struct {
		name string
		goos string
		home string
		env  map[string]string
		want string
	}{
		{
			name: "windows-local-app-data",
			goos: "windows",
			home: "C:/Users/Alice",
			env: map[string]string{
				"LOCALAPPDATA": "C:/Users/Alice/AppData/Local",
			},
			want: "C:/Users/Alice/AppData/Local/VALDR",
		},
		{
			name: "macos-application-support",
			goos: "darwin",
			home: "/Users/alice",
			want: "/Users/alice/Library/Application Support/VALDR",
		},
		{
			name: "linux-xdg",
			goos: "linux",
			home: "/home/alice",
			env: map[string]string{
				"XDG_DATA_HOME": "/home/alice/.data",
			},
			want: "/home/alice/.data/valdr",
		},
		{
			name: "linux-default",
			goos: "linux",
			home: "/home/alice",
			want: "/home/alice/.local/share/valdr",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(key string) string {
				return tc.env[key]
			}
			paths, err := resolvePaths(
				tc.goos,
				tc.home,
				getenv,
				config.NetworkTestnetV02,
			)
			if err != nil {
				t.Fatal(err)
			}
			if filepath.ToSlash(paths.Root) != tc.want {
				t.Fatalf(
					"root=%q want=%q",
					filepath.ToSlash(paths.Root),
					tc.want,
				)
			}
			if paths.Network != config.NetworkTestnetV02 {
				t.Fatalf("network=%q", paths.Network)
			}
			if !strings.HasSuffix(
				filepath.ToSlash(paths.NodeData),
				"/node/testnet",
			) {
				t.Fatalf("unexpected node path %q", paths.NodeData)
			}
			if !strings.HasSuffix(
				filepath.ToSlash(paths.Wallets),
				"/wallets",
			) {
				t.Fatalf("unexpected wallet path %q", paths.Wallets)
			}
		})
	}
}

func TestResolveDesktopPathsRejectsUnknownNetwork(t *testing.T) {
	_, err := resolvePaths(
		"linux",
		"/home/alice",
		func(string) string { return "" },
		"mainnet",
	)
	if err == nil {
		t.Fatal("unknown/mainnet network unexpectedly accepted")
	}
}
