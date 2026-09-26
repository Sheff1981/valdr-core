package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/mining"
	"github.com/Sheff1981/valdr-core/storage"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestMigrateAndVerifyDBCommands(t *testing.T) {
	dir := t.TempDir()
	legacyStore, err := storage.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	legacyChain, err := blockchain.NewPersistent(legacyStore)
	if err != nil {
		t.Fatal(err)
	}
	minerWallet, err := wallet.New("miner")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mining.MineBlock(
		legacyChain,
		minerWallet.Address,
		config.GenesisTimestamp+60,
		nil,
	); err != nil {
		t.Fatal(err)
	}

	legacyPath := filepath.Join(dir, "blockchain.json")
	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	code := run([]string{
		"migrate",
		"--from-v0.1", dir,
		"--network", "valdr-devnet-1",
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("migrate exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "\"original_preserved\": true") ||
		!strings.Contains(out.String(), "\"height\": 1") {
		t.Fatalf("unexpected migrate output: %s", out.String())
	}
	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatalf("legacy file disappeared: %v", err)
	}

	out.Reset()
	errOut.Reset()
	code = run([]string{
		"verify-db",
		"--data", dir,
		"--network", config.NetworkLegacyV01,
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("legacy verify-db exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "\"valid\": true") ||
		!strings.Contains(out.String(), "\"schema_version\": 2") ||
		!strings.Contains(out.String(), "\"height\": 1") {
		t.Fatalf("unexpected verify output: %s", out.String())
	}
}

func TestInitRejectsUnmigratedLegacyData(t *testing.T) {
	dir := t.TempDir()
	legacyStore, err := storage.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := blockchain.NewPersistent(legacyStore); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	code := run([]string{"init", "--data", dir}, &out, &errOut)
	if code != 1 {
		t.Fatalf("init exit=%d, want 1; stdout=%s stderr=%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "run: valdrd migrate") {
		t.Fatalf("missing migration guidance: %s", errOut.String())
	}
	if _, err := os.Stat(storage.BadgerPath(dir)); !os.IsNotExist(err) {
		t.Fatalf("init created v0.2 database before migration: err=%v", err)
	}
}


func TestMigratedLegacyDBDoesNotOpenAsDefaultTestnet2(t *testing.T) {
	dir := t.TempDir()
	legacyStore, err := storage.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := blockchain.NewPersistent(legacyStore); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if code := run([]string{
		"migrate",
		"--from-v0.1", dir,
		"--network", "valdr-devnet-1",
	}, &out, &errOut); code != 0 {
		t.Fatalf("migrate exit=%d stderr=%s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if code := run([]string{"verify-db", "--data", dir}, &out, &errOut); code == 0 {
		t.Fatalf("default Testnet2 verify unexpectedly opened legacy DB: %s", out.String())
	}
	if !strings.Contains(errOut.String(), "storage network mismatch") {
		t.Fatalf("unexpected default-network error: %s", errOut.String())
	}
}
