package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/mempool"
	"github.com/Sheff1981/valdr-core/mining"
	"github.com/Sheff1981/valdr-core/p2p"
	"github.com/Sheff1981/valdr-core/rpc"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestWalletCLIFlow(t *testing.T) {
	dir := t.TempDir()

	var out, errOut bytes.Buffer
	password := "cli-test-passphrase"
	if code := run(
		[]string{
			"wallet", "create",
			"--dir", dir,
			"--name", "alice",
			"--password-fd", testPasswordFD(t, password),
		},
		&out,
		&errOut,
	); code != 0 {
		t.Fatalf("create exit=%d stderr=%s", code, errOut.String())
	}
	if strings.Contains(out.String(), "private_key") {
		t.Fatal("wallet create leaked private key")
	}
	if !strings.Contains(out.String(), "VDR1") {
		t.Fatalf("wallet create output missing VALDR address: %s", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := run(
		[]string{"wallet", "list", "--dir", dir},
		&out,
		&errOut,
	); code != 0 {
		t.Fatalf("list exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "alice") {
		t.Fatalf("wallet list missing alice: %s", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := run(
		[]string{
			"wallet", "export",
			"--dir", dir,
			"--password-fd", testPasswordFD(t, password),
			"alice",
		},
		&out,
		&errOut,
	); code != 0 {
		t.Fatalf("export exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "private_key") {
		t.Fatal("wallet export did not include private key")
	}
	if !strings.Contains(errOut.String(), "WARNING") {
		t.Fatal("wallet export did not warn about private key exposure")
	}
}

func TestRPCBackedCLIStatusBalanceSendAndQueries(t *testing.T) {
	dir := t.TempDir()
	store := wallet.NewStore(dir)
	password := "rpc-cli-passphrase"
	alice, err := store.CreateEncrypted("alice", []byte(password))
	if err != nil {
		t.Fatal(err)
	}
	bob, err := wallet.New("bob")
	if err != nil {
		t.Fatal(err)
	}

	chain := blockchain.New()
	pool := mempool.New()
	block1, err := mining.MineBlock(
		chain,
		alice.Address,
		config.GenesisTimestamp+60,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	node, err := p2p.NewNode(p2p.NodeConfig{
		NodeID:        "cli-test-node",
		ListenAddress: "127.0.0.1:0",
		Blockchain:    chain,
		Mempool:       pool,
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := rpc.NewServer(chain, node)
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	var out, errOut bytes.Buffer
	if code := run(
		[]string{"status", "--node", httpServer.URL},
		&out,
		&errOut,
	); code != 0 {
		t.Fatalf("status exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "\"height\": 1") {
		t.Fatalf("status output = %s", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := run(
		[]string{"block", "get", "--node", httpServer.URL, "1"},
		&out,
		&errOut,
	); code != 0 {
		t.Fatalf("block get exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), block1.BlockHash) {
		t.Fatalf("block output missing hash: %s", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := run(
		[]string{"balance", "--node", httpServer.URL, alice.Address},
		&out,
		&errOut,
	); code != 0 {
		t.Fatalf("balance exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "\"balance_vdr\": \"50\"") {
		t.Fatalf("balance output = %s", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := run(
		[]string{
			"send",
			"--node", httpServer.URL,
			"--wallet-dir", dir,
			"--from", "alice",
			"--to", bob.Address,
			"--amount", "10",
			"--password-fd", testPasswordFD(t, password),
		},
		&out,
		&errOut,
	); code != 0 {
		t.Fatalf("send exit=%d stderr=%s", code, errOut.String())
	}

	var sent rpc.SendTransactionResult
	if err := json.Unmarshal(out.Bytes(), &sent); err != nil {
		t.Fatal(err)
	}
	if sent.TransactionID == "" {
		t.Fatal("send returned empty transaction id")
	}
	if pool.Len() != 1 {
		t.Fatalf("mempool length = %d, want 1", pool.Len())
	}

	out.Reset()
	errOut.Reset()
	if code := run(
		[]string{"tx", "get", "--node", httpServer.URL, sent.TransactionID},
		&out,
		&errOut,
	); code != 0 {
		t.Fatalf("tx get exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "\"confirmed\": false") {
		t.Fatalf("tx output = %s", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := run(
		[]string{"mempool", "--node", httpServer.URL},
		&out,
		&errOut,
	); code != 0 {
		t.Fatalf("mempool exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), sent.TransactionID) {
		t.Fatalf("mempool output missing txid: %s", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := run(
		[]string{"mining", "info", "--node", httpServer.URL},
		&out,
		&errOut,
	); code != 0 {
		t.Fatalf("mining info exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "\"next_height\": 2") {
		t.Fatalf("mining output = %s", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := run(
		[]string{"peers", "--node", httpServer.URL},
		&out,
		&errOut,
	); code != 0 {
		t.Fatalf("peers exit=%d stderr=%s", code, errOut.String())
	}
	if strings.TrimSpace(out.String()) != "[]" {
		t.Fatalf("peers output = %s, want []", out.String())
	}
}

func TestWalletCLIRejectsPasswordArgument(t *testing.T) {
	dir := t.TempDir()
	var out, errOut bytes.Buffer
	code := run(
		[]string{
			"wallet", "create",
			"--dir", dir,
			"--name", "alice",
			"--password", "must-not-be-accepted",
		},
		&out,
		&errOut,
	)
	if code != 2 {
		t.Fatalf("exit=%d want=2 stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "flag provided but not defined") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func testPasswordFD(t *testing.T, password string) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "valdr-password-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	if err := file.Chmod(0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(password + "\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	return strconv.Itoa(int(file.Fd()))
}

func TestParseAndFormatVDR(t *testing.T) {
	tests := map[string]uint64{
		"10":         10 * config.AtomicUnitsPerVDR,
		"0.00000001": 1,
		"1.25":       125_000_000,
	}
	for input, want := range tests {
		got, err := parseVDR(input)
		if err != nil {
			t.Fatalf("parseVDR(%q): %v", input, err)
		}
		if got != want {
			t.Fatalf("parseVDR(%q) = %d, want %d", input, got, want)
		}
	}
	if _, err := parseVDR("1.000000001"); err == nil {
		t.Fatal("parseVDR accepted more than 8 decimal places")
	}
	if got := formatVDR(125_000_000); got != "1.25" {
		t.Fatalf("formatVDR = %q, want 1.25", got)
	}
}
