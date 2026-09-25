package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/storage"
)

func TestTestnetInitAndVerifyDB(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "testnet-node")
	var out, errOut bytes.Buffer

	code := run([]string{
		"init",
		"--data", dir,
		"--network", config.NetworkTestnetV029,
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"network": "testnet2"`) ||
		!strings.Contains(out.String(), `"chain_id": "valdr-testnet-2"`) {
		t.Fatalf("unexpected init output: %s", out.String())
	}

	raw, err := os.ReadFile(filepath.Join(dir, "node.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"chain_id": "valdr-testnet-2"`) {
		t.Fatalf("node metadata missing testnet chain id: %s", raw)
	}

	store, err := storage.NewBadgerStore(
		dir,
		"valdr-testnet-2",
		config.TestnetV029GenesisHash,
	)
	if err != nil {
		t.Fatal(err)
	}
	info, err := store.Info()
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if info.Network != "valdr-testnet-2" ||
		info.GenesisHash != config.TestnetV029GenesisHash ||
		info.Height != 0 ||
		info.ActiveTip != config.TestnetV029GenesisHash {
		t.Fatalf("unexpected testnet storage info: %+v", info)
	}

	out.Reset()
	errOut.Reset()
	code = run([]string{
		"verify-db",
		"--data", dir,
		"--network", config.NetworkTestnetV029,
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("verify-db exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"valid": true`) ||
		!strings.Contains(out.String(), `"network": "valdr-testnet-2"`) ||
		!strings.Contains(out.String(), config.TestnetV029GenesisHash) {
		t.Fatalf("unexpected verify output: %s", out.String())
	}
}

func TestTestnetDBRejectsLegacyProfile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "testnet-node")
	var out, errOut bytes.Buffer
	if code := run([]string{
		"init",
		"--data", dir,
		"--network", config.NetworkTestnetV029,
	}, &out, &errOut); code != 0 {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	code := run([]string{
		"verify-db",
		"--data", dir,
		"--network", config.NetworkLegacyV01,
	}, &out, &errOut)
	if code == 0 {
		t.Fatalf("legacy profile unexpectedly opened Testnet DB: %s", out.String())
	}
	if !strings.Contains(errOut.String(), "storage network mismatch") {
		t.Fatalf("unexpected mismatch error: %s", errOut.String())
	}
}
